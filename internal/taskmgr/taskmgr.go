package taskmgr

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"path/filepath"

	"github.com/google/uuid"
	"github.com/yokitheyo/31_07_25/internal/model"
	"github.com/yokitheyo/31_07_25/internal/service"
)

const maxTasks = 3
const maxFiles = 3

type TaskManager struct {
	mu    sync.Mutex
	tasks map[string]*model.Task
	sem   chan struct{} // semaphore for concurrent archiving
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*model.Task),
		sem:   make(chan struct{}, maxTasks),
	}
}

func (tm *TaskManager) CreateTask() (*model.Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if len(tm.tasks) >= maxTasks {
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

func (tm *TaskManager) AddFile(taskID, url string) error {
	// Валидация расширения
	ext := strings.ToLower(filepath.Ext(url))
	if ext != ".pdf" && ext != ".jpeg" {
		return fmt.Errorf("file extension not allowed: %s", ext)
	}
	// Валидация URL
	u, err := urlParse(url)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("invalid url: %s", url)
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	task, ok := tm.tasks[taskID]
	if !ok {
		return ErrTaskNotFound
	}
	if len(task.Files) >= maxFiles {
		return ErrTooManyFiles
	}
	task.Files = append(task.Files, model.FileInfo{URL: url})
	if len(task.Files) == maxFiles {
		task.Status = model.StatusInProgress
		go tm.archiveTask(task)
	}
	return nil
}

func (tm *TaskManager) archiveTask(task *model.Task) {
	tm.sem <- struct{}{}        // acquire slot
	defer func() { <-tm.sem }() // release slot
	archivePath := filepath.Join("archives", task.ID+".zip")
	urls := make([]string, len(task.Files))
	for i, f := range task.Files {
		urls[i] = f.URL
	}
	failed, err := service.DownloadAndArchive(urls, []string{".pdf", ".jpeg"}, archivePath)
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if err != nil {
		task.Status = model.StatusError
		task.ArchiveURL = ""
		return
	}
	// Обновляем статусы файлов
	for i, f := range task.Files {
		found := false
		for _, fail := range failed {
			if fail != "" && (f.URL == fail || failContainsURL(fail, f.URL)) {
				task.Files[i].Success = false
				task.Files[i].Reason = fail
				found = true
				break
			}
		}
		if !found {
			task.Files[i].Success = true
		}
	}
	task.Status = model.StatusDone
	task.ArchiveURL = "/archives/" + task.ID + ".zip"
}

func failContainsURL(fail, url string) bool {
	return len(fail) > 0 && (fail == url || (len(url) > 0 && (fail == url+" (download error)" || fail == url+" (not allowed)")))
}

func (tm *TaskManager) GetTask(taskID string) (*model.Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	task, ok := tm.tasks[taskID]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

func urlParse(raw string) (*url.URL, error) {
	return url.Parse(raw)
}

var (
	ErrTooManyTasks = fmt.Errorf("too many active tasks")
	ErrTooManyFiles = fmt.Errorf("too many files in task")
	ErrTaskNotFound = fmt.Errorf("task not found")
)
