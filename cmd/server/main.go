package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/yokitheyo/31_07_25/internal/api"
	"github.com/yokitheyo/31_07_25/internal/config"
	"github.com/yokitheyo/31_07_25/internal/taskmgr"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	cleanupArchives() // очистка при старте
	go func() {
		for {
			time.Sleep(time.Hour)
			cleanupArchives()
		}
	}()

	tm := taskmgr.NewTaskManager()
	mux := http.NewServeMux()
	api.RegisterHandlers(mux, tm)

	log.Printf("Server starting on :%d...", cfg.Server.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.Port), logRequest(mux))
}

func logRequest(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		h.ServeHTTP(w, r)
	})
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
