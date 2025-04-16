package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"steam-reviewer/router"
)

func main() {
	_ = os.MkdirAll("./data", 0755)
	if err := godotenv.Load(); err != nil {
		log.Fatal("加载.env失败:", err)
	}

	r := router.SetupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "9010"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Minute, // 等 DeepSeek 慢慢来
		IdleTimeout:  2 * time.Minute,
	}

	log.Printf("启动服务：http://localhost:%s", port)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务启动失败: %v", err)
	}
}
