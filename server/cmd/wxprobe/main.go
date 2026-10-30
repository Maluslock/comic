package main

import (
	"context"
	"log"
	"os"

	"github.com/Maluslock/comic/server/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")
	wx := service.NewWxMsgService()
	if wx == nil {
		log.Fatal("wx service disabled: check WX_APPID/WX_APP_SECRET")
	}
	tmpl := service.LoadTmplIDs()
	data := map[string]string{}
	vals := []string{"光影行者", "已确认接单", "2026-11-20 16:00", "基础套餐", "#101"}
	for i, f := range tmpl.ResFields {
		if i < len(vals) {
			data[f] = vals[i]
		}
	}
	log.Printf("probe openid=%s tmpl=%s fields=%v", os.Getenv("PROBE_OPENID"), tmpl.Result, tmpl.ResFields)
	wx.SendSubscribe(context.Background(), os.Getenv("PROBE_OPENID"), tmpl.Result, "pages/order/detail?id=101", data)
	log.Println("probe done (见上一行错误码；43101/40003 属预期，说明请求已到微信)")
}
