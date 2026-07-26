package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var mockEvents = []map[string]interface{}{
	{"id": 1, "name": "上海 CP30", "location": "上海", "venue": "国家会展中心", "startDate": "2026-07-14T00:00:00+08:00", "endDate": "2026-07-16T00:00:00+08:00", "coverUrl": "https://picsum.photos/seed/comic1/750/360", "tags": []string{"综合", "同人", "cosplay"}, "status": "upcoming", "typeName": "综合", "description": "中国最大的同人创作交流展会，汇集全国优秀同人创作者"},
	{"id": 2, "name": "成都 CD28", "location": "成都", "venue": "世纪城新国际会展中心", "startDate": "2026-07-21T00:00:00+08:00", "endDate": "2026-07-22T00:00:00+08:00", "coverUrl": "https://picsum.photos/seed/comic2/750/360", "tags": []string{"综合", "游戏", "音乐"}, "status": "upcoming", "typeName": "综合", "description": "西南地区最具影响力的动漫游戏展"},
	{"id": 3, "name": "广州萤火虫", "location": "广州", "venue": "保利世贸博览馆", "startDate": "2026-07-10T00:00:00+08:00", "endDate": "2026-07-12T00:00:00+08:00", "coverUrl": "https://picsum.photos/seed/comic3/750/360", "tags": []string{"动漫", "游戏", "同人"}, "status": "upcoming", "typeName": "综合", "description": "华南地区规模最大的动漫游戏展"},
	{"id": 4, "name": "北京 IDO42", "location": "北京", "venue": "国家会议中心", "startDate": "2026-07-28T00:00:00+08:00", "endDate": "2026-07-29T00:00:00+08:00", "coverUrl": "https://picsum.photos/seed/comic4/750/360", "tags": []string{"综合", "原创", "汉服"}, "status": "upcoming", "typeName": "综合", "description": "北方最大的同人创作交流展会"},
	{"id": 5, "name": "杭州 CJ漫展", "location": "杭州", "venue": "白马湖国际会展中心", "startDate": "2026-07-17T00:00:00+08:00", "endDate": "2026-07-18T00:00:00+08:00", "coverUrl": "https://picsum.photos/seed/comic5/750/360", "tags": []string{"动漫", "cosplay", "游戏"}, "status": "upcoming", "typeName": "综合", "description": "长三角地区新锐动漫展"},
}

var mockPhotographers = []map[string]interface{}{
	{"id": 1, "name": "光影行者", "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=photographer1&backgroundColor=b6e3f4", "location": "北京", "rating": 4.9, "reviewCount": 234, "orderCount": 567, "tags": []string{"日系", "古风", "科幻"}},
	{"id": 2, "name": "樱花落", "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=photographer2&backgroundColor=ffd5dc", "location": "上海", "rating": 4.8, "reviewCount": 186, "orderCount": 423, "tags": []string{"日系", "清新", "少女"}},
	{"id": 3, "name": "暗夜骑士", "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=photographer3&backgroundColor=c0aede", "location": "广州", "rating": 4.7, "reviewCount": 156, "orderCount": 312, "tags": []string{"暗黑", "赛博朋克", "哥特"}},
	{"id": 4, "name": "古风公子", "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=photographer4&backgroundColor=d1d4f9", "location": "杭州", "rating": 4.9, "reviewCount": 298, "orderCount": 678, "tags": []string{"古风", "汉服", "仙侠"}},
}

var mockWorks = []map[string]interface{}{
	{"id": 1, "title": "原神 - 雷电将军", "images": []string{"https://picsum.photos/seed/coswork1/600/450", "https://picsum.photos/seed/coswork2/600/450"}, "photographerName": "光影行者"},
	{"id": 2, "title": "鬼灭之刃 - 祢豆子", "images": []string{"https://picsum.photos/seed/coswork3/600/450"}, "photographerName": "光影行者"},
	{"id": 3, "title": "魔卡少女樱", "images": []string{"https://picsum.photos/seed/coswork4/600/450"}, "photographerName": "樱花落"},
	{"id": 4, "title": "古风仙侠", "images": []string{"https://picsum.photos/seed/coswork5/600/450"}, "photographerName": "古风公子"},
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/home", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"banners": []map[string]interface{}{
				{"id": 1, "imageUrl": "https://picsum.photos/seed/comic1/750/360", "title": "上海 CP30", "linkType": "event", "linkId": nil},
				{"id": 2, "imageUrl": "https://picsum.photos/seed/comic2/750/360", "title": "成都 CD28", "linkType": "event", "linkId": nil},
				{"id": 3, "imageUrl": "https://picsum.photos/seed/comic3/750/360", "title": "广州萤火虫", "linkType": "event", "linkId": nil},
			},
			"upcomingEvents":           mockEvents,
			"hotTags": []map[string]interface{}{
				{"name": "日系", "usageCount": 156},
				{"name": "古风", "usageCount": 142},
				{"name": "暗黑", "usageCount": 98},
				{"name": "清新", "usageCount": 87},
				{"name": "科幻", "usageCount": 76},
				{"name": "赛博朋克", "usageCount": 65},
				{"name": "哥特", "usageCount": 54},
				{"name": "少女", "usageCount": 43},
			},
			"recommendedPhotographers": mockPhotographers,
			"featuredWorks":            mockWorks,
		})
	})

	mux.HandleFunc("/api/v1/events/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/events/")
		// Reject paths like /api/v1/events/1/extra or empty
		if idStr == "" || strings.Contains(idStr, "/") {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "event not found"})
			return
		}

		eventID, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "event not found"})
			return
		}

		var event map[string]interface{}
		for _, e := range mockEvents {
			if e["id"] == eventID {
				event = e
				break
			}
		}

		if event == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "event not found"})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            event["id"],
			"name":          event["name"],
			"location":      event["location"],
			"venue":         event["venue"],
			"startDate":     event["startDate"],
			"endDate":       event["endDate"],
			"coverUrl":      event["coverUrl"],
			"tags":          event["tags"],
			"status":        event["status"],
			"typeName":      event["typeName"],
			"description":   event["description"],
			"photographers": mockPhotographers,
			"featuredWorks": mockWorks,
		})
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	log.Println("🚀 米拉漫展 mock server listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", cors(mux)))
}
