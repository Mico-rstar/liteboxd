package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/fslongjin/liteboxd/internal/k8s"
	"github.com/fslongjin/liteboxd/internal/model"
	corev1 "k8s.io/api/core/v1"
)

type SandboxService struct {
	k8sClient *k8s.Client
}

func NewSandboxService(k8sClient *k8s.Client) *SandboxService {
	return &SandboxService{
		k8sClient: k8sClient,
	}
}

func (s *SandboxService) Create(ctx context.Context, req *model.CreateSandboxRequest) (*model.Sandbox, error) {
	id := generateID()

	ttl := req.TTL
	if ttl <= 0 {
		ttl = 3600
	}

	opts := k8s.CreatePodOptions{
		ID:     id,
		Image:  req.Image,
		CPU:    req.CPU,
		Memory: req.Memory,
		TTL:    ttl,
		Env:    req.Env,
	}

	pod, err := s.k8sClient.CreatePod(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create pod: %w", err)
	}

	return s.podToSandbox(pod), nil
}

func (s *SandboxService) Get(ctx context.Context, id string) (*model.Sandbox, error) {
	pod, err := s.k8sClient.GetPod(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}
	return s.podToSandbox(pod), nil
}

func (s *SandboxService) List(ctx context.Context) (*model.SandboxListResponse, error) {
	pods, err := s.k8sClient.ListPods(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	items := make([]model.Sandbox, 0, len(pods.Items))
	for _, pod := range pods.Items {
		items = append(items, *s.podToSandbox(&pod))
	}

	return &model.SandboxListResponse{Items: items}, nil
}

func (s *SandboxService) Delete(ctx context.Context, id string) error {
	return s.k8sClient.DeletePod(ctx, id)
}

func (s *SandboxService) Exec(ctx context.Context, id string, req *model.ExecRequest) (*model.ExecResponse, error) {
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	result, err := s.k8sClient.Exec(execCtx, id, req.Command)
	if err != nil {
		return nil, fmt.Errorf("failed to exec: %w", err)
	}

	return &model.ExecResponse{
		ExitCode: result.ExitCode,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
	}, nil
}

func (s *SandboxService) UploadFile(ctx context.Context, id, path string, content []byte) error {
	return s.k8sClient.UploadFile(ctx, id, path, content)
}

func (s *SandboxService) DownloadFile(ctx context.Context, id, path string) ([]byte, error) {
	return s.k8sClient.DownloadFile(ctx, id, path)
}

func (s *SandboxService) GetLogs(ctx context.Context, id string, tailLines int64) (*model.LogsResponse, error) {
	logs, err := s.k8sClient.GetLogs(ctx, id, tailLines)
	if err != nil {
		logs = ""
	}

	events, err := s.k8sClient.GetPodEvents(ctx, id)
	if err != nil {
		events = nil
	}

	return &model.LogsResponse{
		Logs:   logs,
		Events: events,
	}, nil
}

func (s *SandboxService) StartTTLCleaner(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			s.cleanExpiredSandboxes()
		}
	}()
}

func (s *SandboxService) cleanExpiredSandboxes() {
	ctx := context.Background()
	pods, err := s.k8sClient.ListPods(ctx)
	if err != nil {
		fmt.Printf("TTL cleaner: failed to list pods: %v\n", err)
		return
	}

	for _, pod := range pods.Items {
		if s.isExpired(&pod) {
			sandboxID := pod.Labels[k8s.LabelSandboxID]
			fmt.Printf("TTL cleaner: deleting expired sandbox %s\n", sandboxID)
			if err := s.k8sClient.DeletePod(ctx, sandboxID); err != nil {
				fmt.Printf("TTL cleaner: failed to delete pod %s: %v\n", sandboxID, err)
			}
		}
	}
}

func (s *SandboxService) isExpired(pod *corev1.Pod) bool {
	createdAtStr := pod.Annotations[k8s.AnnotationCreatedAt]
	ttlStr := pod.Annotations[k8s.AnnotationTTL]

	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return false
	}

	ttl, err := strconv.Atoi(ttlStr)
	if err != nil {
		return false
	}

	expiresAt := createdAt.Add(time.Duration(ttl) * time.Second)
	return time.Now().After(expiresAt)
}

func (s *SandboxService) podToSandbox(pod *corev1.Pod) *model.Sandbox {
	sandboxID := pod.Labels[k8s.LabelSandboxID]
	createdAtStr := pod.Annotations[k8s.AnnotationCreatedAt]
	ttlStr := pod.Annotations[k8s.AnnotationTTL]

	createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
	ttl, _ := strconv.Atoi(ttlStr)
	expiresAt := createdAt.Add(time.Duration(ttl) * time.Second)

	var image string
	if len(pod.Spec.Containers) > 0 {
		image = pod.Spec.Containers[0].Image
	}

	var cpu, memory string
	if len(pod.Spec.Containers) > 0 {
		limits := pod.Spec.Containers[0].Resources.Limits
		if cpuQty, ok := limits[corev1.ResourceCPU]; ok {
			cpu = cpuQty.String()
		}
		if memQty, ok := limits[corev1.ResourceMemory]; ok {
			memory = memQty.String()
		}
	}

	return &model.Sandbox{
		ID:        sandboxID,
		Image:     image,
		CPU:       cpu,
		Memory:    memory,
		TTL:       ttl,
		Status:    convertPodPhase(pod.Status.Phase),
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}
}

func convertPodPhase(phase corev1.PodPhase) model.SandboxStatus {
	switch phase {
	case corev1.PodPending:
		return model.SandboxStatusPending
	case corev1.PodRunning:
		return model.SandboxStatusRunning
	case corev1.PodSucceeded:
		return model.SandboxStatusSucceeded
	case corev1.PodFailed:
		return model.SandboxStatusFailed
	default:
		return model.SandboxStatusUnknown
	}
}

func generateID() string {
	id := uuid.New().String()
	return id[:8]
}
