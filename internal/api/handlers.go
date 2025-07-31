package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/yokitheyo/31_07_25/internal/taskmgr"
)

type APIHandler struct {
	TM *taskmgr.TaskManager
}

func RegisterHandlers(mux *http.ServeMux, tm *taskmgr.TaskManager) {
	h := &APIHandler{TM: tm}
	mux.HandleFunc("/tasks", h.handleCreateTask)
	mux.HandleFunc("/tasks/", h.handleTaskSubroutes)
	mux.HandleFunc("/archives/", handleArchiveDownload)
}

func (h *APIHandler) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	task, err := h.TM.CreateTask()
	if err != nil {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *APIHandler) handleTaskSubroutes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/tasks/"), "/")
	if len(parts) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	taskID := parts[0]
	if len(parts) == 2 && parts[1] == "files" {
		h.handleAddFile(w, r, taskID)
		return
	}
	if len(parts) == 2 && parts[1] == "status" {
		h.handleGetStatus(w, r, taskID)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (h *APIHandler) handleAddFile(w http.ResponseWriter, r *http.Request, taskID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err := h.TM.AddFile(taskID, req.URL)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) handleGetStatus(w http.ResponseWriter, r *http.Request, taskID string) {
	task, err := h.TM.GetTask(taskID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func handleArchiveDownload(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Path[len("/archives/"):]
	if file == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	path := "archives/" + file
	w.Header().Set("Content-Type", "application/zip")
	http.ServeFile(w, r, path)
}
