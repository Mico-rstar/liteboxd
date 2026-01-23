package model

import "time"

type SandboxStatus string

const (
	SandboxStatusPending   SandboxStatus = "pending"
	SandboxStatusRunning   SandboxStatus = "running"
	SandboxStatusSucceeded SandboxStatus = "succeeded"
	SandboxStatusFailed    SandboxStatus = "failed"
	SandboxStatusUnknown   SandboxStatus = "unknown"
)

type Sandbox struct {
	ID        string            `json:"id"`
	Image     string            `json:"image"`
	CPU       string            `json:"cpu"`
	Memory    string            `json:"memory"`
	TTL       int               `json:"ttl"`
	Env       map[string]string `json:"env,omitempty"`
	Status    SandboxStatus     `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type CreateSandboxRequest struct {
	Image  string            `json:"image" binding:"required"`
	CPU    string            `json:"cpu"`
	Memory string            `json:"memory"`
	TTL    int               `json:"ttl"`
	Env    map[string]string `json:"env"`
}

type ExecRequest struct {
	Command []string `json:"command" binding:"required"`
	Timeout int      `json:"timeout"`
}

type ExecResponse struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

type SandboxListResponse struct {
	Items []Sandbox `json:"items"`
}

type LogsResponse struct {
	Logs   string   `json:"logs"`
	Events []string `json:"events"`
}
