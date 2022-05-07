package routers

import (
	"easytools/controllers"
	"github.com/astaxie/beego"
)

func init() {
    beego.Router("/", &controllers.MainController{})
    beego.Router("/fast-md5", &controllers.FastMd5Controller{})
	beego.Router("/fast-md5/calc",&controllers.FastMd5Controller{}, "Get:Calc")
    beego.Router("/chatroom", &controllers.ChatRoomController{})
	beego.Router("/chatroom/sendmsg", &controllers.WebsocketController{})
}
