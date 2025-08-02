package taskmgr

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"path/filepath"

	"github.com/google/uuid"
	"github.com/yokitheyo/31_07_25/internal/config"
	"github.com/yokitheyo/31_07_25/internal/model"
	"github.com/yokitheyo/31_07_25/internal/service"
)

const maxActiveTasks = 3
const maxFiles = 3

type TaskManager struct {
	mu          sync.Mutex
	tasks       map[string]*model.Task
	sem         chan struct{}
	config      *config.Config
	activeTasks int
}

func NewTaskManager(cfg *config.Config) *TaskManager {
	return &TaskManager{
		tasks:  make(map[string]*model.Task),
		sem:    make(chan struct{}, maxActiveTasks),
		config: cfg,
	}
}

func (tm *TaskManager) CreateTask() (*model.Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.activeTasks >= maxActiveTasks {
		return nil, ErrTooManyTasks
	}

	id := uuid.New().String()
	task := &model.Task{
		ID:        id,
		CreatedAt: time.Now(),
		Status:    model.StatusPending,
	}
	tm.tasks[id] = task
	return task, nil
}

func (tm *TaskManager) AddFile(taskID, fileURL string) error {
	u, err := url.Parse(fileURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("invalid url: %s", fileURL)
	}

	ext := strings.ToLower(filepath.Ext(u.Path))
	if ext != "" {
		allowed := false
		for _, allowedExt := range tm.config.Files.AllowedExtensions {
			if ext == allowedExt {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file extension not allowed: %s", ext)
		}
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks[taskID]
	if !ok {
		return ErrTaskNotFound
	}

	if task.Status != model.StatusPending {
		return fmt.Errorf("cannot add files to task in status: %s", task.Status)
	}

	if len(task.Files) >= maxFiles {
		return ErrTooManyFiles
	}

	task.Files = append(task.Files, model.FileInfo{URL: fileURL})

	if len(task.Files) == maxFiles {
		task.Status = model.StatusInProgress
		tm.activeTasks++
		go tm.archiveTask(task)
	}

	return nil
}

func (tm *TaskManager) archiveTask(task *model.Task) {
	tm.sem <- struct{}{}
	defer func() {
		<-tm.sem
		tm.mu.Lock()
		tm.activeTasks--
		tm.mu.Unlock()
	}()

	archivePath := filepath.Join("archives", task.ID+".zip")
	urls := make([]string, len(task.Files))
	for i, f := range task.Files {
		urls[i] = f.URL
	}

	failed, err := service.DownloadAndArchive(
		urls,
		tm.config.Files.AllowedContentTypes,
		archivePath,
	)

	tm.mu.Lock()
	defer tm.mu.Unlock()

	if err != nil {
		task.Status = model.StatusError
		return
	}

	failedMap := make(map[string]string)
	for _, fail := range failed {
		if fail != "" {
			parts := strings.SplitN(fail, " (", 2)
			if len(parts) >= 1 {
				failedMap[parts[0]] = fail
			}
		}
	}

	for i, f := range task.Files {
		if reason, failed := failedMap[f.URL]; failed {
			task.Files[i].Success = false
			task.Files[i].Reason = reason
		} else {
			task.Files[i].Success = true
		}
	}

	task.Status = model.StatusDone
	task.ArchiveURL = "/archives/" + task.ID + ".zip"
}

func (tm *TaskManager) GetTask(taskID string) (*model.Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks[taskID]
	if !ok {
		return nil, ErrTaskNotFound
	}

	taskCopy := *task
	taskCopy.Files = make([]model.FileInfo, len(task.Files))
	copy(taskCopy.Files, task.Files)

	return &taskCopy, nil
}

func (tm *TaskManager) CleanupOldTasks(maxAge time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for id, task := range tm.tasks {
		if task.Status == model.StatusDone || task.Status == model.StatusError {
			if task.CreatedAt.Before(cutoff) {
				delete(tm.tasks, id)
			}
		}
	}
}

var (
	ErrTooManyTasks = fmt.Errorf("too many active tasks")
	ErrTooManyFiles = fmt.Errorf("too many files in task")
	ErrTaskNotFound = fmt.Errorf("task not found")
)
