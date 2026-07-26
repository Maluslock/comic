package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Maluslock/comic/server/internal/config"
	"github.com/Maluslock/comic/server/internal/handler"
	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

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
			{"id": 1, "imageUrl": "https://picsum.photos/seed/comic1/750/360", "title": "上海 CP30", "linkType": "event", "linkId": nil},
			{"id": 2, "imageUrl": "https://picsum.photos/seed/comic2/750/360", "title": "成都 CD28", "linkType": "event", "linkId": nil},
			{"id": 3, "imageUrl": "https://picsum.photos/seed/comic3/750/360", "title": "广州萤火虫", "linkType": "event", "linkId": nil},
		},
		"upcomingEvents": []gin.H{
			{"id": 1, "name": "上海 CP30", "location": "上海", "venue": "国家会展中心", "startDate": "2026-07-14T00:00:00+08:00", "endDate": "2026-07-16T00:00:00+08:00", "coverUrl": "https://picsum.photos/seed/comic1/750/360", "tags": []string{"综合", "同人", "cosplay"}, "status": "upcoming", "typeName": "综合"},
			{"id": 2, "name": "成都 CD28", "location": "成都", "venue": "世纪城新国际会展中心", "startDate": "2026-07-21T00:00:00+08:00", "endDate": "2026-07-22T00:00:00+08:00", "coverUrl": "https://picsum.photos/seed/comic2/750/360", "tags": []string{"综合", "游戏", "音乐"}, "status": "upcoming", "typeName": "综合"},
			{"id": 3, "name": "广州萤火虫", "location": "广州", "venue": "保利世贸博览馆", "startDate": "2026-07-10T00:00:00+08:00", "endDate": "2026-07-12T00:00:00+08:00", "coverUrl": "https://picsum.photos/seed/comic3/750/360", "tags": []string{"动漫", "游戏", "同人"}, "status": "upcoming", "typeName": "综合"},
			{"id": 4, "name": "北京 IDO42", "location": "北京", "venue": "国家会议中心", "startDate": "2026-07-28T00:00:00+08:00", "endDate": "2026-07-29T00:00:00+08:00", "coverUrl": "https://picsum.photos/seed/comic4/750/360", "tags": []string{"综合", "原创", "汉服"}, "status": "upcoming", "typeName": "综合"},
			{"id": 5, "name": "杭州 CJ漫展", "location": "杭州", "venue": "白马湖国际会展中心", "startDate": "2026-07-17T00:00:00+08:00", "endDate": "2026-07-18T00:00:00+08:00", "coverUrl": "https://picsum.photos/seed/comic5/750/360", "tags": []string{"动漫", "cosplay", "游戏"}, "status": "upcoming", "typeName": "综合"},
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
			{"id": 1, "name": "光影行者", "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=photographer1&backgroundColor=b6e3f4", "location": "北京", "rating": 4.9, "reviewCount": 234, "orderCount": 567, "tags": []string{"日系", "古风", "科幻"}},
			{"id": 2, "name": "樱花落", "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=photographer2&backgroundColor=ffd5dc", "location": "上海", "rating": 4.8, "reviewCount": 186, "orderCount": 423, "tags": []string{"日系", "清新", "少女"}},
			{"id": 3, "name": "暗夜骑士", "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=photographer3&backgroundColor=c0aede", "location": "广州", "rating": 4.7, "reviewCount": 156, "orderCount": 312, "tags": []string{"暗黑", "赛博朋克", "哥特"}},
			{"id": 4, "name": "古风公子", "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=photographer4&backgroundColor=d1d4f9", "location": "杭州", "rating": 4.9, "reviewCount": 298, "orderCount": 678, "tags": []string{"古风", "汉服", "仙侠"}},
		},
		"featuredWorks": []gin.H{
			{"id": 1, "title": "原神 - 雷电将军", "images": []string{"https://picsum.photos/seed/coswork1/600/450", "https://picsum.photos/seed/coswork2/600/450"}, "photographerName": "光影行者"},
			{"id": 2, "title": "鬼灭之刃 - 祢豆子", "images": []string{"https://picsum.photos/seed/coswork3/600/450"}, "photographerName": "光影行者"},
			{"id": 3, "title": "魔卡少女樱", "images": []string{"https://picsum.photos/seed/coswork4/600/450"}, "photographerName": "樱花落"},
			{"id": 4, "title": "古风仙侠", "images": []string{"https://picsum.photos/seed/coswork5/600/450"}, "photographerName": "古风公子"},
		},
	})
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Check mock mode
	if os.Getenv("USE_MOCK") == "true" {
		log.Println("⚠️  USE_MOCK=true — running without database, returning seed data")
		router.GET("/api/v1/home", mockHomeHandler)
	} else {
		// Initialize database connection pool
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		pool, err := pgxpool.New(ctx, cfg.DSN())
		if err != nil {
			log.Fatalf("failed to connect to database: %v", err)
		}
		defer pool.Close()

		queries := repository.New(pool)
		homeRepo := repository.NewHomeRepository(queries)
		homeSvc := service.NewHomeService(homeRepo)
		homeH := handler.NewHomeHandler(homeSvc)
		router.GET("/api/v1/home", homeH.GetHome)
	}

	// Create HTTP server
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", addr)
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
