package api

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yokitheyo/31_07_25/internal/taskmgr"
)

type APIHandler struct {
	TM *taskmgr.TaskManager
}

type AddFileRequest struct {
	URL string `json:"url" binding:"required"`
}

func RegisterHandlers(r *gin.Engine, tm *taskmgr.TaskManager) {
	h := &APIHandler{TM: tm}

	r.POST("/tasks", h.createTask)
	r.POST("/tasks/:id/files", h.addFile)
	r.GET("/tasks/:id/status", h.getStatus)

	r.GET("/archives/:filename", h.serveArchive)
}

func (h *APIHandler) createTask(c *gin.Context) {
	task, err := h.TM.CreateTask()
	if err != nil {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, task)
}

func (h *APIHandler) addFile(c *gin.Context) {
	taskID := c.Param("id")

	var req AddFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	err := h.TM.AddFile(taskID, req.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *APIHandler) getStatus(c *gin.Context) {
	taskID := c.Param("id")

	task, err := h.TM.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *APIHandler) serveArchive(c *gin.Context) {
	filename := c.Param("filename")

	if !strings.HasSuffix(filename, ".zip") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid archive filename"})
		return
	}

	taskID := strings.TrimSuffix(filename, ".zip")

	_, err := h.TM.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "archive not found"})
		return
	}

	filepath := filepath.Join("archives", filename)
	c.File(filepath)
}
