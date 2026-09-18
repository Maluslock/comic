package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Maluslock/comic/server/internal/cache"
	"github.com/Maluslock/comic/server/internal/config"
	"github.com/Maluslock/comic/server/internal/handler"
	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

func derefStrPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// mockHomeHandler returns hardcoded seed data when DB is unavailable
func mockHomeHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"banners": []gin.H{
			{"id": 1, "imageUrl": "/static/img/cover-chinajoy.jpg", "title": "ChinaJoy 2026", "linkType": "event", "linkId": 1},
			{"id": 2, "imageUrl": "data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3E%E7%AC%AC40%E5%B1%8A%E8%90%A4%E7%81%AB%E8%99%AB%E6%BC%AB%E5%B1%95%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E5%B9%BF%E5%B7%9E%20%C2%B7%20%E4%BF%9D%E5%88%A9%E4%B8%96%E8%B4%B8%E5%8D%9A%E8%A7%88%E9%A6%86%3C%2Ftext%3E%3C%2Fsvg%3E", "title": "第40届萤火虫漫展", "linkType": "event", "linkId": 2},
			{"id": 3, "imageUrl": "data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3ECP33%20%E7%BB%BC%E5%90%88%E5%90%8C%E4%BA%BA%E5%B1%95%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E6%9D%AD%E5%B7%9E%20%C2%B7%20%E6%9D%AD%E5%B7%9E%E5%A4%A7%E4%BC%9A%E5%B1%95%E4%B8%AD%E5%BF%83%3C%2Ftext%3E%3C%2Fsvg%3E", "title": "CP33 综合同人展", "linkType": "event", "linkId": 3},
		},
		"upcomingEvents": []gin.H{
			{"id": 1, "name": "ChinaJoy 2026", "location": "上海", "venue": "上海新国际博览中心", "startDate": "2026-08-01T00:00:00+08:00", "endDate": "2026-08-04T00:00:00+08:00", "coverUrl": "/static/img/cover-chinajoy.jpg", "tags": []string{"游戏", "数码", "cosplay", "电竞"}, "status": "upcoming", "typeName": "游戏"},
			{"id": 2, "name": "第40届萤火虫漫展", "location": "广州", "venue": "保利世贸博览馆", "startDate": "2026-08-14T00:00:00+08:00", "endDate": "2026-08-17T00:00:00+08:00", "coverUrl": "data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3E%E7%AC%AC40%E5%B1%8A%E8%90%A4%E7%81%AB%E8%99%AB%E6%BC%AB%E5%B1%95%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E5%B9%BF%E5%B7%9E%20%C2%B7%20%E4%BF%9D%E5%88%A9%E4%B8%96%E8%B4%B8%E5%8D%9A%E8%A7%88%E9%A6%86%3C%2Ftext%3E%3C%2Fsvg%3E", "tags": []string{"动漫", "游戏", "cosplay"}, "status": "upcoming", "typeName": "综合"},
			{"id": 3, "name": "CP33 综合同人展", "location": "杭州", "venue": "杭州大会展中心", "startDate": "2026-09-12T00:00:00+08:00", "endDate": "2026-09-15T00:00:00+08:00", "coverUrl": "data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3ECP33%20%E7%BB%BC%E5%90%88%E5%90%8C%E4%BA%BA%E5%B1%95%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E6%9D%AD%E5%B7%9E%20%C2%B7%20%E6%9D%AD%E5%B7%9E%E5%A4%A7%E4%BC%9A%E5%B1%95%E4%B8%AD%E5%BF%83%3C%2Ftext%3E%3C%2Fsvg%3E", "tags": []string{"同人", "创作", "cosplay"}, "status": "upcoming", "typeName": "同人"},
			{"id": 4, "name": "西安梦乡动漫展", "location": "西安", "venue": "西安国际会展中心", "startDate": "2026-09-11T00:00:00+08:00", "endDate": "2026-09-14T00:00:00+08:00", "coverUrl": "data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3E%E8%A5%BF%E5%AE%89%E6%A2%A6%E4%B9%A1%E5%8A%A8%E6%BC%AB%E5%B1%95%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E8%A5%BF%E5%AE%89%20%C2%B7%20%E8%A5%BF%E5%AE%89%E5%9B%BD%E9%99%85%E4%BC%9A%E5%B1%95%E4%B8%AD%E5%BF%83%3C%2Ftext%3E%3C%2Fsvg%3E", "tags": []string{"动漫", "同人", "汉服"}, "status": "upcoming", "typeName": "综合"},
			{"id": 5, "name": "IJOY国际动漫节", "location": "北京", "venue": "北京国家会议中心", "startDate": "2026-10-01T00:00:00+08:00", "endDate": "2026-10-03T00:00:00+08:00", "coverUrl": "data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3EIJOY%E5%9B%BD%E9%99%85%E5%8A%A8%E6%BC%AB%E8%8A%82%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E5%8C%97%E4%BA%AC%20%C2%B7%20%E5%8C%97%E4%BA%AC%E5%9B%BD%E5%AE%B6%E4%BC%9A%E8%AE%AE%E4%B8%AD%E5%BF%83%3C%2Ftext%3E%3C%2Fsvg%3E", "tags": []string{"综合", "国潮", "cosplay"}, "status": "upcoming", "typeName": "综合"},
			{"id": 6, "name": "成都第二十四届世界线动漫展", "location": "成都", "venue": "中国西部国际博览城", "startDate": "2026-09-26T00:00:00+08:00", "endDate": "2026-09-28T00:00:00+08:00", "coverUrl": "data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3E%E6%88%90%E9%83%BD%E7%AC%AC%E4%BA%8C%E5%8D%81%E5%9B%9B%E5%B1%8A%E4%B8%96%E7%95%8C%E7%BA%BF%E5%8A%A8%E6%BC%AB%E5%B1%95%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E6%88%90%E9%83%BD%20%C2%B7%20%E4%B8%AD%E5%9B%BD%E8%A5%BF%E9%83%A8%E5%9B%BD%E9%99%85%E5%8D%9A%E8%A7%88%E5%9F%8E%3C%2Ftext%3E%3C%2Fsvg%3E", "tags": []string{"动漫", "cosplay", "游戏"}, "status": "upcoming", "typeName": "综合"},
			{"id": 7, "name": "COMICUP 33 新青年", "location": "杭州", "venue": "杭州大会展中心", "startDate": "2026-10-10T00:00:00+08:00", "endDate": "2026-10-12T00:00:00+08:00", "coverUrl": "data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3ECOMICUP%2033%20%E6%96%B0%E9%9D%92%E5%B9%B4%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E6%9D%AD%E5%B7%9E%20%C2%B7%20%E6%9D%AD%E5%B7%9E%E5%A4%A7%E4%BC%9A%E5%B1%95%E4%B8%AD%E5%BF%83%3C%2Ftext%3E%3C%2Fsvg%3E", "tags": []string{"同人", "创作", "动漫"}, "status": "upcoming", "typeName": "同人"},
			{"id": 8, "name": "2026上海CCG EXPO", "location": "上海", "venue": "上海跨国采购会展中心", "startDate": "2026-10-15T00:00:00+08:00", "endDate": "2026-10-18T00:00:00+08:00", "coverUrl": "/static/img/cover-ccg.jpg", "tags": []string{"综合", "游戏", "动漫"}, "status": "upcoming", "typeName": "综合"},
		},
		"hotTags": []gin.H{
			{"name": "日系", "usageCount": 156},
			{"name": "古风", "usageCount": 142},
			{"name": "暗黑", "usageCount": 98},
			{"name": "清新", "usageCount": 87},
			{"name": "科幻", "usageCount": 76},
			{"name": "赛博朋克", "usageCount": 65},
			{"name": "哥特", "usageCount": 54},
			{"name": "少女", "usageCount": 43},
		},
		"recommendedPhotographers": []gin.H{
			{"id": 1, "name": "光影行者", "avatar": "/static/img/avatar-photographer1.svg", "location": "北京", "rating": 4.9, "reviewCount": 234, "orderCount": 567, "tags": []string{"日系", "古风", "科幻"}},
			{"id": 2, "name": "樱花落", "avatar": "/static/img/avatar-photographer2.svg", "location": "上海", "rating": 4.8, "reviewCount": 186, "orderCount": 423, "tags": []string{"日系", "清新", "少女"}},
			{"id": 3, "name": "暗夜骑士", "avatar": "/static/img/avatar-photographer3.svg", "location": "广州", "rating": 4.7, "reviewCount": 156, "orderCount": 312, "tags": []string{"暗黑", "赛博朋克", "哥特"}},
			{"id": 4, "name": "古风公子", "avatar": "/static/img/avatar-photographer4.svg", "location": "杭州", "rating": 4.9, "reviewCount": 298, "orderCount": 678, "tags": []string{"古风", "汉服", "仙侠"}},
		},
		"featuredWorks": []gin.H{
			{"id": 1, "title": "原神 - 雷电将军", "images": []string{"/static/img/work-1.jpg", "/static/img/work-2.jpg"}, "photographerName": "光影行者"},
			{"id": 2, "title": "鬼灭之刃 - 祢豆子", "images": []string{"/static/img/work-3.jpg"}, "photographerName": "光影行者"},
			{"id": 3, "title": "魔卡少女樱", "images": []string{"/static/img/work-4.jpg"}, "photographerName": "樱花落"},
			{"id": 4, "title": "古风仙侠", "images": []string{"/static/img/work-5.jpg"}, "photographerName": "古风公子"},
		},
	})
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.Use(corsMiddleware())

	// Static path is relative to CWD: run the API from server/ so /static works.
	router.Static("/static", "./static")

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Initialize Redis cache (best-effort; app works without it)
	redisCache := cache.NewRedisCache(cfg.RedisAddr(), "", 0)
	if redisCache != nil {
		defer redisCache.Close()
	}

	if os.Getenv("USE_MOCK") == "true" {
		log.Println("USE_MOCK=true — running without database, returning seed data")
		router.GET("/api/v1/home", mockHomeHandler)
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		pool, err := pgxpool.New(ctx, cfg.DSN())
		if err != nil {
			log.Fatalf("failed to connect to database: %v", err)
		}
		defer pool.Close()

		queries := repository.New(pool)
		userRepo := repository.NewUserRepo(pool)
		bookingSvc := service.NewBookingService(queries)

		// Home
		homeRepo := repository.NewHomeRepository(queries)
		homeSvc := service.NewHomeService(homeRepo)
		homeH := handler.NewHomeHandler(homeSvc, redisCache)
		router.GET("/api/v1/home", homeH.GetHome)

		// Photographers
		photographerSvc := service.NewPhotographerService(queries)
		photographerH := handler.NewPhotographerHandler(photographerSvc, redisCache, bookingSvc)
		svcH := handler.NewPhotographerServiceHandler(photographerSvc)
		router.GET("/api/v1/photographers/services/mine", middleware.AuthRequired(userRepo), svcH.MyServices)
		router.POST("/api/v1/photographers/services", middleware.AuthRequired(userRepo), svcH.Create)
		router.PUT("/api/v1/photographers/services/:id", middleware.AuthRequired(userRepo), svcH.Update)
		router.DELETE("/api/v1/photographers/services/:id", middleware.AuthRequired(userRepo), svcH.Delete)
		router.GET("/api/v1/services/templates", svcH.Templates)
		router.GET("/api/v1/photographers", middleware.AuthOptional(userRepo), photographerH.List)
		router.GET("/api/v1/photographers/:id", photographerH.Detail)
		router.GET("/api/v1/photographers/:id/timeslots", photographerH.TimeSlots)
		router.POST("/api/v1/photographers/activate", middleware.AuthRequired(userRepo), photographerH.Activate)
		router.GET("/api/v1/photographers/by-user/:userId", middleware.AuthRequired(userRepo), photographerH.GetByUser)
		router.PUT("/api/v1/photographers/profile", middleware.AuthRequired(userRepo), photographerH.UpdateProfile)
		router.GET("/api/v1/photographers/profile/mine", middleware.AuthRequired(userRepo), photographerH.MyProfile)

		// Photographer works (self-service create/mine/delete)
		workH := handler.NewPhotographerWorkHandler(photographerSvc)
		router.POST("/api/v1/photographers/works", middleware.AuthRequired(userRepo), workH.CreateWork)
		router.GET("/api/v1/photographers/works/mine", middleware.AuthRequired(userRepo), workH.MyWorks)
		router.PUT("/api/v1/photographers/works/:id", middleware.AuthRequired(userRepo), workH.UpdateWork)
		router.DELETE("/api/v1/photographers/works/:id", middleware.AuthRequired(userRepo), workH.DeleteWork)

		// Events
		eventSvc := service.NewEventService(queries)
		eventH := handler.NewEventHandler(eventSvc, redisCache)
		router.GET("/api/v1/events", eventH.List)
		router.GET("/api/v1/events/:id", eventH.Detail)

		// Bookings
		bookingH := handler.NewBookingHandler(bookingSvc)
		router.POST("/api/v1/bookings", middleware.AuthRequired(userRepo), bookingH.Create)
		router.PUT("/api/v1/bookings/:id/status", middleware.AuthRequired(userRepo), bookingH.UpdateStatus)
	router.POST("/api/v1/bookings/:id/quote", middleware.AuthRequired(userRepo), bookingH.Quote)
	router.POST("/api/v1/bookings/:id/quote/respond", middleware.AuthRequired(userRepo), bookingH.RespondQuote)
		router.GET("/api/v1/bookings/:userId", middleware.AuthRequired(userRepo), bookingH.ListByUser)
		router.GET("/api/v1/bookings/photographer/:photographerId", middleware.AuthRequired(userRepo), bookingH.ListByPhotographer)

		// Reviews
		reviewSvc := service.NewReviewService(queries)
		reviewH := handler.NewReviewHandler(reviewSvc, userRepo)
		router.POST("/api/v1/reviews", middleware.AuthRequired(userRepo), reviewH.Create)
		router.GET("/api/v1/reviews/mine", middleware.AuthRequired(userRepo), reviewH.MyReviews)
		router.GET("/api/v1/reviews/:photographerId", reviewH.ListByPhotographer)

		// Auth
		authSvc := service.NewAuthService(userRepo, queries)
		authH := handler.NewAuthHandler(authSvc)
		router.POST("/api/v1/login", authH.Login)
		router.GET("/api/v1/me", middleware.AuthRequired(userRepo), func(c *gin.Context) {
			u, err := userRepo.GetByID(c.Request.Context(), middleware.UserID(c))
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
				return
			}
			resp := gin.H{"id": u.ID, "name": u.Name, "phone": u.Phone, "avatar": u.Avatar, "bio": derefStrPtr(u.Bio)}
			if profile, err := queries.GetPhotographerByUserID(c.Request.Context(), u.ID); err == nil {
				resp["photographerId"] = profile.ID
			}
			c.JSON(http.StatusOK, resp)
		})
		router.POST("/api/v1/me/logout-all", middleware.AuthRequired(userRepo), func(c *gin.Context) {
			if err := userRepo.DeleteTokensByUser(c.Request.Context(), middleware.UserID(c)); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
		router.PUT("/api/v1/me/profile", middleware.AuthRequired(userRepo), func(c *gin.Context) {
			var req struct {
				Name   string `json:"name" binding:"required"`
				Avatar string `json:"avatar"`
				Bio    string `json:"bio"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
				return
			}
			avatar := req.Avatar
			bio := req.Bio
			if avatar == "" || bio == "" {
				u, err := userRepo.GetByID(c.Request.Context(), middleware.UserID(c))
				if err != nil {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
					return
				}
				if avatar == "" {
					avatar = u.Avatar
				}
				if bio == "" {
					bio = derefStrPtr(u.Bio)
				}
			}
			err := userRepo.UpdateProfile(c.Request.Context(), middleware.UserID(c), req.Name, avatar, bio)
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		// Favorites
		favoriteSvc := service.NewFavoriteService(queries)
		favoriteH := handler.NewFavoriteHandler(favoriteSvc)
		router.POST("/api/v1/favorites", middleware.AuthRequired(userRepo), favoriteH.Add)
		router.DELETE("/api/v1/favorites/:userId/:photographerId", middleware.AuthRequired(userRepo), favoriteH.Remove)
		router.GET("/api/v1/favorites/:userId", middleware.AuthRequired(userRepo), favoriteH.List)

		// Chat
		chatSvc := service.NewChatService(queries)
		chatH := handler.NewChatHandler(chatSvc)
		router.POST("/api/v1/chat/sessions", middleware.AuthRequired(userRepo), chatH.CreateSession)
		router.GET("/api/v1/chat/sessions/:userId", middleware.AuthRequired(userRepo), chatH.ListSessions)
		router.GET("/api/v1/chat/messages/:sessionId", middleware.AuthRequired(userRepo), chatH.ListMessages)
		router.POST("/api/v1/chat/messages", middleware.AuthRequired(userRepo), chatH.SendMessage)
		router.POST("/api/v1/chat/sessions/:sessionId/read", middleware.AuthRequired(userRepo), chatH.MarkRead)
		router.GET("/api/v1/chat/unread/:userId", middleware.AuthRequired(userRepo), chatH.Unread)

		// Blocks
		blockSvc := service.NewBlockService(queries)
		blockH := handler.NewBlockHandler(blockSvc)
		router.POST("/api/v1/blocks", middleware.AuthRequired(userRepo), blockH.Add)
		router.DELETE("/api/v1/blocks/:blockedUserId", middleware.AuthRequired(userRepo), blockH.Remove)
		router.GET("/api/v1/blocks", middleware.AuthRequired(userRepo), blockH.List)

		// Tags
		tagSvc := service.NewTagService(queries)
		tagH := handler.NewTagHandler(tagSvc)
		router.GET("/api/v1/tags", tagH.GetAll)

		// Follows & Subscriptions
		followSvc := service.NewFollowService(queries)
		followH := handler.NewFollowHandler(followSvc)
		router.POST("/api/v1/follows", middleware.AuthRequired(userRepo), followH.Follow)
		router.DELETE("/api/v1/follows/:userId/:eventId", middleware.AuthRequired(userRepo), followH.Unfollow)
		router.GET("/api/v1/follows/:userId", middleware.AuthRequired(userRepo), followH.ListFollows)
		router.POST("/api/v1/subscribe", middleware.AuthRequired(userRepo), followH.Subscribe)

		// Notifications
		notificationSvc := service.NewNotificationService(queries)
		notificationH := handler.NewNotificationHandler(notificationSvc)
		router.POST("/api/v1/notifications", middleware.AuthRequired(userRepo), notificationH.Create)
		router.GET("/api/v1/notifications/:userId", middleware.AuthRequired(userRepo), notificationH.List)
		router.POST("/api/v1/notifications/:id/read", middleware.AuthRequired(userRepo), notificationH.MarkRead)

		// Admin
		adminsRepo := repository.NewAdminRepo(pool)
		adminSvc := service.NewAdminService(adminsRepo)
		adminH := handler.NewAdminHandler(adminSvc)
		adminGroup := router.Group("/api/admin/v1")
		adminGroup.POST("/login", adminH.Login)
		adminGroup.Use(middleware.AdminAuthRequired(adminsRepo))
		adminAuditRepo := repository.NewAdminAuditRepo(pool)
		adminGroup.Use(middleware.AdminAuditLogger(adminAuditRepo))
		adminAuditH := handler.NewAdminAuditHandler(adminAuditRepo)
		adminGroup.GET("/audit-logs", adminAuditH.List)
		adminStatsRepo := repository.NewAdminStatsRepo(pool)
		adminStatsSvc := service.NewAdminStatsService(adminStatsRepo)
		adminDashboardH := handler.NewAdminDashboardHandler(adminStatsSvc)
		adminGroup.GET("/dashboard", adminDashboardH.Dashboard)
		adminManageRepo := repository.NewAdminManageRepo(pool)
		adminManageSvc := service.NewAdminManageService(adminManageRepo, bookingSvc)
		adminManageH := handler.NewAdminManageHandler(adminManageSvc)
		adminGroup.GET("/photographers", adminManageH.ListPhotographers)
		adminGroup.GET("/photographers/:id", adminManageH.Detail)
		adminGroup.PUT("/photographers/:id/certified", adminManageH.SetCertified)
		adminGroup.GET("/orders", adminManageH.ListOrders)
		adminGroup.PUT("/orders/:id/status", adminManageH.SetOrderStatus)
		adminGroup.GET("/events", adminManageH.ListEvents)
		adminGroup.PUT("/events/:id/status", adminManageH.SetEventStatus)
		adminUserRepo := repository.NewAdminUserRepo(pool)
		adminUserSvc := service.NewAdminUserService(adminUserRepo)
		adminUserH := handler.NewAdminUserHandler(adminUserSvc)
		adminGroup.GET("/users", adminUserH.List)
		adminGroup.GET("/users/:id", adminUserH.Detail)
		adminGroup.PUT("/users/:id/status", adminUserH.SetStatus)

		// Certification applications (C-end submit + admin review)
		certRepo := repository.NewCertApplicationRepo(pool)
		certSvc := service.NewCertApplicationService(certRepo, queries, notificationSvc)
		certH := handler.NewCertApplicationHandler(certSvc)
		router.POST("/api/v1/photographers/cert-apply", middleware.AuthRequired(userRepo), certH.Submit)
		router.GET("/api/v1/photographers/cert-application", middleware.AuthRequired(userRepo), certH.MyApplication)
		router.GET("/api/v1/photographers/cert-applications", middleware.AuthRequired(userRepo), certH.MyApplications)
		adminCertH := handler.NewAdminCertHandler(certSvc)
		adminGroup.GET("/cert-applications", adminCertH.List)
		adminGroup.GET("/cert-applications/:id", adminCertH.Detail)
		adminGroup.PUT("/cert-applications/:id/review", adminCertH.Review)

		// Banner management
		adminBannerRepo := repository.NewAdminBannerRepo(pool)
		adminBannerSvc := service.NewAdminBannerService(adminBannerRepo)
		adminBannerH := handler.NewAdminBannerHandler(adminBannerSvc)
		adminGroup.GET("/banners", adminBannerH.List)
		adminGroup.POST("/banners", adminBannerH.Create)
		adminGroup.PUT("/banners/:id", adminBannerH.Update)
		adminGroup.PUT("/banners/:id/status", adminBannerH.SetStatus)

		// Admin account management
		adminAdminH := handler.NewAdminAdminHandler(adminSvc)
		adminGroup.GET("/admins", adminAdminH.List)
		adminGroup.POST("/admins", adminAdminH.Create)
		adminGroup.PUT("/admins/:id/status", adminAdminH.SetStatus)
		adminGroup.PUT("/admins/:id/password", adminAdminH.ResetPassword)

		// Notification broadcast (admin)
		notifAdminSvc := service.NewNotificationAdminService(queries, notificationSvc, userRepo)
		notifAdminH := handler.NewAdminNotificationHandler(notifAdminSvc)
		adminGroup.POST("/notifications", notifAdminH.Publish)
		adminGroup.GET("/notifications", notifAdminH.List)
		adminGroup.DELETE("/notifications/:id", notifAdminH.Delete)

		// Content management (admin)
		contentRepo := repository.NewAdminContentRepo(pool)
		contentSvc := service.NewAdminContentService(contentRepo)
		contentH := handler.NewAdminContentHandler(contentSvc, redisCache)
		adminGroup.GET("/works", contentH.ListWorks)
		adminGroup.PUT("/works/:id/status", contentH.SetWorkStatus)
		adminGroup.GET("/reviews", contentH.ListReviews)
		adminGroup.DELETE("/reviews/:id", contentH.DeleteReview)
		adminGroup.GET("/tags", contentH.ListTags)
		adminGroup.POST("/tags", contentH.CreateTag)
		adminGroup.PUT("/tags/:id", contentH.UpdateTag)
		adminGroup.DELETE("/tags/:id", contentH.DeleteTag)
		adminGroup.POST("/tags/merge", contentH.MergeTag)

		uploadH := handler.NewUploadHandler(os.Getenv("UPLOAD_DIR"))
		router.POST("/api/v1/upload", middleware.AuthRequired(userRepo), uploadH.Upload)

		adminExportH := handler.NewAdminExportHandler(adminManageSvc, adminUserSvc)
		adminGroup.GET("/export/orders", adminExportH.Orders)
		adminGroup.GET("/export/users", adminExportH.Users)
		adminGroup.GET("/export/photographers", adminExportH.Photographers)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}
