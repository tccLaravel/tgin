package upload

import (
	"fmt"
	"mime/multipart"
	"os"
	"strings"
	"tgin/pkg/file"
	"tgin/pkg/logging"
	"tgin/pkg/setting"
	"tgin/pkg/util"
)

func GetImageName(name string) string {
	ext := file.GetFileExt(name)
	fileName := strings.TrimSuffix(name, ext)
	fileName = util.EncodeMd5(fileName)
	return fileName + ext
}

func GetImageFullUrl(name string) string {
	return setting.AppSetting.ImagePrefixUrl + "/" + GetImagePath() + name
}

func GetImagePath() string {
	return setting.AppSetting.ImageSavePath
}

func GetImageFullPath() string {
	return setting.AppSetting.RuntimeRootPath + GetImagePath()
}

func CheckImageExt(filename string) bool {
	ext := file.GetFileExt(filename)
	logging.Info(setting.AppSetting.ImageAllowExts)
	for _, allowExt := range setting.AppSetting.ImageAllowExts {
		if strings.ToUpper(allowExt) == strings.ToUpper(ext) {
			return true
		}
	}
	return false
}

func CheckImageSize(f multipart.File) bool {
	size, err := file.GetSize(f)
	if err != nil {
		logging.Warn(err)
		return false
	}
	return size <= setting.AppSetting.ImageMaxSize
}

func CheckImage(src string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("os.getwd err: %v \n", err)
	}
	err = file.IsNotExistMkDir(dir + "/" + src)
	if err != nil {
		return fmt.Errorf("file.IsNotExistMkDir err: %v", err)
	}
	perm := file.CheckPermission(src)
	if perm == true {
		return fmt.Errorf("file.CheckPermission Permission denied src: %s", src)
	}
	return nil
}
