package api

import (
	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
	"net/http"
	"runtime"
	"tgin/models"
	"tgin/pkg/app"
	"tgin/pkg/e"
	"tgin/pkg/util"
)

type auth struct {
	Username string `valid:"Required; MaxSize(50)"`
	Password string `valid:"Required; MaxSize(50)"`
}

func GetAuth(c *gin.Context) {
	appG := app.Gin{C: c}
	username := c.PostForm("username")
	password := c.PostForm("password")
	valid := validation.Validation{}
	a := auth{Username: username, Password: password}
	ok, _ := valid.Valid(&a)
	data := make(map[string]interface{})
	code := e.INVALID_PARAMS
	if !ok {
		app.LogError(valid.Errors)
		appG.Response(http.StatusOK, code, data)
		return
	}
	isExist := models.CheckAuth(username, password)
	if isExist {
		token, err := util.GenerateToken(username, password)
		if err != nil {
			code = e.ERROR_AUTH_TOKEN
			_, file, line, _ := runtime.Caller(1)
			_ = valid.SetError(file+" ,line: "+string(line), e.GetMsg(code))
			app.LogError(valid.Errors)
		} else {
			data["token"] = token
			code = e.SUCCESS
		}
	} else {
		code = e.ERROR_AUTH
		_, file, line, _ := runtime.Caller(1)
		_ = valid.SetError(file+" ,line: "+string(line), e.GetMsg(code))
		app.LogError(valid.Errors)
	}
	appG.Response(http.StatusOK, code, data)
}
