package controllers

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/astaxie/beego"
	"strings"
)

type MainController struct {
	beego.Controller
}

type FastMd5Controller struct {
	beego.Controller
}

type fastMd5Result struct {
	Capital_32 string `json:"capital_32"`
	Lowercase_32 string `json:"lowercase_32"`
	Capital_16 string `json:"capital_16"`
	Lowercase_16 string `json:"lowercase_16"`
}

func (c *MainController) Get() {
	c.TplName = "index.html"
}

func (c *FastMd5Controller) Get() {
	c.TplName = "fast-md5.html"
}

func (c *FastMd5Controller) Calc()  {
	if (c.GetString("s") == ""){
		return
	}

	md5 := md5.New()
	md5.Write([]byte(c.GetString("s")))
	hex_md5_str :=  hex.EncodeToString(md5.Sum(nil))

	result := fastMd5Result{
		strings.ToUpper(hex_md5_str),
		hex_md5_str,
		strings.ToUpper(hex_md5_str)[8:24],
		hex_md5_str[8:24],
	}

	json_data, _ := json.Marshal(result)
	beego.Debug("fast-md5 | ", c.GetString("s"), " | ", string(json_data))
	c.Ctx.ResponseWriter.Write(json_data)
}
