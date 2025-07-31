package model

import "time"

type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
	StatusError      TaskStatus = "error"
)

type FileInfo struct {
	URL     string `json:"url"`
	Success bool   `json:"success"`
	Reason  string `json:"reason,omitempty"`
}

type Task struct {
	ID         string     `json:"id"`
	Files      []FileInfo `json:"files"`
	CreatedAt  time.Time  `json:"created_at"`
	Status     TaskStatus `json:"status"`
	ArchiveURL string     `json:"archive_url,omitempty"`
}
