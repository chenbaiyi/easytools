package main

import (
	_ "easytools/routers"
	"github.com/astaxie/beego"
)

func init() {
	beego.SetStaticPath("/static", "static")
	beego.SetStaticPath("/images", "static/img")
	beego.SetStaticPath("/css", "static/css")
	beego.SetStaticPath("/js", "static/js")
	beego.SetStaticPath("/font", "static/font")
}

func main() {
	beego.Run()
}
