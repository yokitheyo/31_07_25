package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yokitheyo/31_07_25/internal/api"
	"github.com/yokitheyo/31_07_25/internal/archive"
	"github.com/yokitheyo/31_07_25/internal/config"
	"github.com/yokitheyo/31_07_25/internal/taskmgr"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	if err := os.MkdirAll(cfg.ArchiveDir, 0755); err != nil {
		log.Fatalf("failed to create archives directory: %v", err)
	}

	cleaned, err := archive.CleanOldArchives(cfg.ArchiveDir, 2*time.Hour, log.Default())
	if err != nil {
		log.Printf("initial archive cleanup failed: %v", err)
	} else if cleaned > 0 {
		log.Printf("initially cleaned %d old archives", cleaned)
	}

	tm := taskmgr.NewTaskManager(cfg)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	if os.Getenv("GIN_MODE") != "release" {
		r.Use(gin.Logger())
	}

	api.RegisterHandlers(r, tm)

	r.POST("/admin/cleanup", func(c *gin.Context) {
		archived, _ := archive.CleanOldArchives(cfg.ArchiveDir, 2*time.Hour, log.Default())
		tm.CleanupOldTasks(1 * time.Second)
		c.JSON(http.StatusOK, gin.H{
			"status":              "cleanup triggered",
			"removed_archives":    archived,
			"old_tasks_retention": "2h",
		})
	})

	log.Printf("Server starting on :%d...", cfg.Server.Port)
	if err := r.Run(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
