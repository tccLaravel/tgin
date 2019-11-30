package main

import (
	"fmt"
	"tgin/pkg/setting"
	"tgin/routers"
)

func main() {
	g := routers.InitRouter()
	g.Run(fmt.Sprintf(":%d", setting.HTTPPort))
}
