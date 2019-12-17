package tredis

import (
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"os"
	"os/signal"
	"syscall"
	"tgin/pkg/setting"
	"time"
)

type TCache struct {
	pool      *redis.Pool
	prefix    string
	marshal   func(v interface{}) ([]byte, error)
	unmarshal func(data []byte, v interface{}) error
}

// Options redis配置参数
type Options struct {
	Network     string                                 // 通讯协议，默认为 tcp
	Addr        string                                 // redis服务的地址，默认为 127.0.0.1:6379
	Password    string                                 // redis鉴权密码
	Db          int                                    // 数据库
	MaxActive   int                                    // 最大活动连接数，值为0时表示不限制
	MaxIdle     int                                    // 最大空闲连接数
	IdleTimeout int                                    // 空闲连接的超时时间，超过该时间则关闭连接。单位为秒。默认值是5分钟。值为0时表示不关闭空闲连接。此值应该总是大于redis服务的超时时间。
	Prefix      string                                 // 键名前缀
	Marshal     func(v interface{}) ([]byte, error)    // 数据序列化方法，默认使用json.Marshal序列化
	Unmarshal   func(data []byte, v interface{}) error // 数据反序列化方法，默认使用json.Unmarshal序列化
}

func New(options Options) (*TCache, error) {
	r := &TCache{}
	if options.Network == "" {
		options.Network = setting.RedisSetting.Addr
	}
	if options.Addr == "" {
		options.Addr = setting.RedisSetting.Addr
	}
	if options.MaxIdle == 0 {
		options.MaxIdle = setting.RedisSetting.MaxIdle
	}
	if options.IdleTimeout == 0 {
		options.IdleTimeout = setting.RedisSetting.IdleTimeout
	}
	if options.Prefix == "" {
		options.Prefix = setting.RedisSetting.Prefix
	}
	if options.Marshal == nil {
		r.marshal = json.Marshal
	}
	if options.Unmarshal == nil {
		r.unmarshal = json.Unmarshal
	}
	err := r.StartAndGC(options)
	return r, err
}

func (c *TCache) StartAndGC(options Options) error {
	pool := &redis.Pool{
		MaxActive:   setting.RedisSetting.MaxActive,
		MaxIdle:     setting.RedisSetting.MaxIdle,
		IdleTimeout: time.Duration(setting.RedisSetting.IdleTimeout) * time.Second,

		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial(setting.RedisSetting.Network, setting.RedisSetting.Addr)
			if err != nil {
				return nil, err
			}
			if setting.RedisSetting.Password != "" {
				if _, err := conn.Do("AUTH", setting.RedisSetting.Password); err != nil {
					_ = conn.Close()
					return nil, err
				}
			}
			if _, err := conn.Do("SELECT", setting.DatabaseSetting.Name); err != nil {
				_ = conn.Close()
				return nil, err
			}
			return conn, err
		},

		TestOnBorrow: func(conn redis.Conn, t time.Time) error {
			_, err := conn.Do("PING")
			return err
		},
	}

	c.pool = pool
	c.closePool()
	return nil
}

// Do 执行redis命令并返回结果。执行时从连接池获取连接并在执行完命令后关闭连接。
func (c *TCache) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	return conn.Do(commandName, args)
}

// getKey 将健名加上指定的前缀。
func (c *TCache) getKey(key string) string {
	return c.prefix + key
}

// Get 获取键值。一般不直接使用该值，而是配合下面的工具类方法获取具体类型的值，或者直接使用github.com/gomodule/redigo/redis包的工具方法。
func (c *TCache) Get(key string) (interface{}, error) {
	return c.Do("GET", c.getKey(key))
}

// GetString 获取string类型的键值
func (c *TCache) GetString(key string) (string, error) {
	return String(c.Get(key))
}

// GetInt 获取int类型的键值
func (c *TCache) GetInt(key string) (int, error) {
	return Int(c.Get(key))
}

// GetInt64 获取int64类型的键值
func (c *TCache) GetInt64(key string) (int64, error) {
	return Int64(c.Get(key))
}

// GetBool 获取bool类型的键值
func (c *TCache) GetBool(key string) (bool, error) {
	return Bool(c.Get(key))
}

// GetObject 获取非基本类型struct的键值。在实现上，使用json的Marshal和Unmarshal做序列化存取。
func (c *TCache) GetObject(key string, val interface{}) error {
	reply, err := c.Get(key)
	return c.decode(reply, err, val)
}

// Set 存并设置有效时长。时长的单位为秒。
// 基础类型直接保存，其他用json.Marshal后转成string保存。
func (c *TCache) Set(key string, val interface{}, expire int64) error {
	value, err := c.encode(val)
	if err != nil {
		return err
	}
	if expire > 0 {
		_, err := c.Do("SETEX", c.getKey(key), expire, value)
		return err
	}
	_, err = c.Do("SET", c.getKey(key), value)
	return err
}

// Exists 检查键是否存在
func (c *TCache) Exists(key string) (bool, error) {
	return Bool(c.Do("EXISTS", c.getKey(key)))
}

//Del 删除键
func (c *TCache) Del(key string) error {
	_, err := c.Do("DEL", c.getKey(key))
	return err
}

// Flush 清空当前数据库中的所有 key，慎用！
func (c *TCache) Flush() error {
	_, err := c.Do("FLUSHDB")
	return err
}

// 获取过期时间
// TTL 以秒为单位。当 key 不存在时，返回 -2 。 当 key 存在但没有设置剩余生存时间时，返回 -1
func (c *TCache) TTL(key string) (int64, error) {
	return Int64(c.Do("TTL", c.getKey(key)))
}

// Expire 设置键过期时间，expire的单位为秒
func (c *TCache) Expire(key string, expire int64) error {
	_, err := Bool(c.Do("EXPIRE", c.getKey(key)))
	return err
}

// Incr 将 key 中储存的数字值增一
func (c *TCache) Incr(key string) (val int64, err error) {
	return Int64(c.Do("INCR", c.getKey(key)))
}

// IncrBy 将 key 所储存的值加上给定的增量值（increment）。
func (c *TCache) IncrBy(key string, amount int64) (val int64, err error) {
	return Int64(c.Do("INCRBY", c.getKey(key), amount))
}

// Decr 将 key 中储存的数字值减一。
func (c *TCache) Decr(key string) (val int64, err error) {
	return Int64(c.Do("DECR", c.getKey(key)))
}

// DecrBy key 所储存的值减去给定的减量值（decrement）。
func (c *TCache) DecrBy(key string, amount int64) (val int64, err error) {
	return Int64(c.Do("DECRBY", c.getKey(key), amount))
}

// HMSet 将一个map存到Redis hash，同时设置有效期，单位：秒
func (c *TCache) HMSet(key string, val interface{}, expire int) (err error) {
	conn := c.pool.Get()
	defer conn.Close()
	err = conn.Send("HMSET", redis.Args{}.Add(c.getKey(key)).AddFlat(val)...)
	if err != nil {
		return
	}
	if expire > 0 {
		err = conn.Send("EXPIRE", c.getKey(key), int64(expire))
	}
	if err != nil {
		return
	}
	_ = conn.Flush()
	_, err = conn.Receive()
	return
}

/** Redis hash 是一个string类型的field和value的映射表，hash特别适合用于存储对象。 **/
// HSet 将哈希表 key 中的字段 field 的值设为 val
func (c *TCache) HSet(key, field string, val interface{}) (interface{}, error) {
	value, err := c.encode(val)
	if err != nil {
		return nil, err
	}
	return c.Do("HSET", c.getKey(key), field, value)
}

// HGet 获取存储在哈希表中指定字段的值
func (c *TCache) HGet(key, field string) (reply interface{}, err error) {
	reply, err = c.Do("HGET", c.getKey(key), field)
	return
}

// HGetString HGet的工具方法，当字段值为字符串类型时使用
func (c *TCache) HGetString(key, field string) (reply string, err error) {
	reply, err = String(c.HGet(key, field))
	return
}

// HGetInt HGet的工具方法，当字段值为int类型时使用
func (c *TCache) HGetInt(key, field string) (reply int, err error) {
	reply, err = Int(c.HGet(key, field))
	return
}

// HGetInt64 HGet的工具方法，当字段值为int64类型时使用
func (c *TCache) HGetInt64(key, field string) (reply int64, err error) {
	reply, err = Int64(c.HGet(key, field))
	return
}

// HGetBool HGet的工具方法，当字段值为bool类型时使用
func (c *TCache) HGetBool(key, field string) (reply bool, err error) {
	reply, err = Bool(c.HGet(key, field))
	return
}

// HGetObject HGet的工具方法，当字段值为非基本类型的stuct时使用
func (c *TCache) HGetObject(key, field string, valPtr interface{}) error {
	reply, err := c.HGet(key, field)
	return c.decode(reply, err, valPtr)
}

// HGetAll HGetAll("key", &val)
func (c *TCache) HGetAll(key string, valPtr interface{}) error {
	v, err := redis.Values(c.Do("HGETALL", c.getKey(key)))
	if err != nil {
		return err
	}
	if err := redis.ScanStruct(v, valPtr); err != nil {
		fmt.Println(err)
	}
	//fmt.Printf("%+v\n", val)
	return err
}

/** Redis列表是简单的字符串列表，按照插入顺序排序。你可以添加一个元素到列表的头部（左边）或者尾部（右边）*/

// BLPop 它是 LPOP 命令的阻塞版本，当给定列表内没有任何元素可供弹出的时候，连接将被 BLPOP 命令阻塞，直到等待超时或发现可弹出元素为止。
// 超时参数 timeout 接受一个以秒为单位的数字作为值。超时参数设为 0 表示阻塞时间可以无限期延长(block indefinitely) 。
func (c *TCache) BLPop(key string, timeout int) (interface{}, error) {
	values, err := redis.Values(c.Do("BLPOP", c.getKey(key), timeout))
	if err != nil {
		return nil, err
	}
	if len(values) != 2 {
		return nil, fmt.Errorf("redisgo: unexpected number of values, got %d", len(values))
	}
	return values[1], err
}

// encode 序列化要保存的值
func (c *TCache) encode(val interface{}) (interface{}, error) {
	var value interface{}
	switch v := val.(type) {
	case string, int, uint, int8, int16, int32, int64, float32, float64, bool:
		value = v
	default:
		b, err := c.marshal(v)
		if err != nil {
			return nil, err
		}
		value = string(b)
	}
	return value, nil
}

// decode 反序列化保存的struct对象
func (c *TCache) decode(reply interface{}, err error, val interface{}) error {
	str, err := String(reply, err)
	if err != nil {
		return err
	}
	return c.unmarshal([]byte(str), val)
}

func (c *TCache) closePool() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	signal.Notify(ch, syscall.SIGTERM)
	signal.Notify(ch, syscall.SIGKILL)
	go func() {
		<-ch
		_ = c.pool.Close()
		os.Exit(0)
	}()
}
