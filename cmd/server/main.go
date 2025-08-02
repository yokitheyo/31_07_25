package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yokitheyo/31_07_25/internal/api"
	"github.com/yokitheyo/31_07_25/internal/config"
	"github.com/yokitheyo/31_07_25/internal/taskmgr"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	if err := os.MkdirAll("archives", 0755); err != nil {
		log.Fatalf("failed to create archives directory: %v", err)
	}

	cleanupArchives()
	go func() {
		for {
			time.Sleep(time.Hour)
			cleanupArchives()
		}
	}()

	tm := taskmgr.NewTaskManager(cfg)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	api.RegisterHandlers(r, tm)

	log.Printf("Server starting on :%d...", cfg.Server.Port)
	r.Run(fmt.Sprintf(":%d", cfg.Server.Port))
}

func cleanupArchives() {
	files, err := filepath.Glob("archives/*.zip")
	if err != nil {
		log.Printf("archive cleanup error: %v", err)
		return
	}
	cutoff := time.Now().Add(-time.Hour)
	for _, f := range files {
		info, err := os.Stat(f)
		if err == nil && info.ModTime().Before(cutoff) {
			os.Remove(f)
		}
	}
}
