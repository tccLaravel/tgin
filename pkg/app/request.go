package app

import (
	"github.com/astaxie/beego/validation"
	"tgin/pkg/logging"
)

func LogError(errors []*validation.Error) {
	for _, err := range errors {
		logging.Info(err.Key, err.Message)
	}
	return
}
