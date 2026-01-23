package k8s

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	SandboxNamespace    = "liteboxd"
	LabelApp            = "liteboxd"
	LabelSandboxID      = "sandbox-id"
	AnnotationTTL       = "liteboxd/ttl"
	AnnotationCreatedAt = "liteboxd/created-at"
)

type Client struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

func NewClient(kubeconfigPath string) (*Client, error) {
	var config *rest.Config
	var err error

	if kubeconfigPath != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Client{
		clientset: clientset,
		config:    config,
	}, nil
}

func (c *Client) EnsureNamespace(ctx context.Context) error {
	_, err := c.clientset.CoreV1().Namespaces().Get(ctx, SandboxNamespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: SandboxNamespace,
		},
	}
	_, err = c.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	return err
}

type CreatePodOptions struct {
	ID     string
	Image  string
	CPU    string
	Memory string
	TTL    int
	Env    map[string]string
}

func (c *Client) CreatePod(ctx context.Context, opts CreatePodOptions) (*corev1.Pod, error) {
	podName := fmt.Sprintf("sandbox-%s", opts.ID)

	cpuLimit := opts.CPU
	if cpuLimit == "" {
		cpuLimit = "500m"
	}
	memLimit := opts.Memory
	if memLimit == "" {
		memLimit = "512Mi"
	}

	var envVars []corev1.EnvVar
	for k, v := range opts.Env {
		envVars = append(envVars, corev1.EnvVar{Name: k, Value: v})
	}

	var runAsUser int64 = 1000

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: SandboxNamespace,
			Labels: map[string]string{
				"app":          LabelApp,
				LabelSandboxID: opts.ID,
			},
			Annotations: map[string]string{
				AnnotationTTL:       fmt.Sprintf("%d", opts.TTL),
				AnnotationCreatedAt: time.Now().UTC().Format(time.RFC3339),
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Tolerations: []corev1.Toleration{
				{
					Key:      "node.kubernetes.io/disk-pressure",
					Operator: corev1.TolerationOpExists,
				},
				{
					Key:      "node.kubernetes.io/memory-pressure",
					Operator: corev1.TolerationOpExists,
				},
				{
					Key:      "node.kubernetes.io/pid-pressure",
					Operator: corev1.TolerationOpExists,
				},
			},
			SecurityContext: &corev1.PodSecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
			Containers: []corev1.Container{
				{
					Name:    "main",
					Image:   opts.Image,
					Command: []string{"sleep", "infinity"},
					Env:     envVars,
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(cpuLimit),
							corev1.ResourceMemory: resource.MustParse(memLimit),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: boolPtr(false),
						RunAsNonRoot:             boolPtr(true),
						RunAsUser:                &runAsUser,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "workspace",
							MountPath: "/workspace",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "workspace",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	return c.clientset.CoreV1().Pods(SandboxNamespace).Create(ctx, pod, metav1.CreateOptions{})
}

func (c *Client) GetPod(ctx context.Context, sandboxID string) (*corev1.Pod, error) {
	podName := fmt.Sprintf("sandbox-%s", sandboxID)
	return c.clientset.CoreV1().Pods(SandboxNamespace).Get(ctx, podName, metav1.GetOptions{})
}

func (c *Client) ListPods(ctx context.Context) (*corev1.PodList, error) {
	return c.clientset.CoreV1().Pods(SandboxNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", LabelApp),
	})
}

func (c *Client) DeletePod(ctx context.Context, sandboxID string) error {
	podName := fmt.Sprintf("sandbox-%s", sandboxID)
	return c.clientset.CoreV1().Pods(SandboxNamespace).Delete(ctx, podName, metav1.DeleteOptions{})
}

type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func (c *Client) Exec(ctx context.Context, sandboxID string, command []string) (*ExecResult, error) {
	podName := fmt.Sprintf("sandbox-%s", sandboxID)

	req := c.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(SandboxNamespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "main",
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	result := &ExecResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(interface{ ExitStatus() int }); ok {
			result.ExitCode = exitErr.ExitStatus()
		} else {
			result.ExitCode = 1
			result.Stderr = result.Stderr + "\n" + err.Error()
		}
	}

	return result, nil
}

func (c *Client) UploadFile(ctx context.Context, sandboxID string, destPath string, content []byte) error {
	podName := fmt.Sprintf("sandbox-%s", sandboxID)
	dir := filepath.Dir(destPath)
	filename := filepath.Base(destPath)

	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	hdr := &tar.Header{
		Name: filename,
		Mode: 0644,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}
	if _, err := tw.Write(content); err != nil {
		return fmt.Errorf("failed to write tar content: %w", err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	req := c.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(SandboxNamespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "main",
			Command:   []string{"tar", "-xf", "-", "-C", dir},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  &tarBuf,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func (c *Client) DownloadFile(ctx context.Context, sandboxID string, srcPath string) ([]byte, error) {
	podName := fmt.Sprintf("sandbox-%s", sandboxID)
	dir := filepath.Dir(srcPath)
	filename := filepath.Base(srcPath)

	req := c.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(SandboxNamespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "main",
			Command:   []string{"tar", "-cf", "-", "-C", dir, filename},
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w, stderr: %s", err, stderr.String())
	}

	tr := tar.NewReader(&stdout)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar: %w", err)
		}
		if strings.TrimPrefix(hdr.Name, "./") == filename {
			content, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("failed to read file content: %w", err)
			}
			return content, nil
		}
	}

	return nil, fmt.Errorf("file not found in tar archive")
}

func (c *Client) GetLogs(ctx context.Context, sandboxID string, tailLines int64) (string, error) {
	podName := fmt.Sprintf("sandbox-%s", sandboxID)

	opts := &corev1.PodLogOptions{
		Container: "main",
	}
	if tailLines > 0 {
		opts.TailLines = &tailLines
	}

	req := c.clientset.CoreV1().Pods(SandboxNamespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	defer stream.Close()

	logs, err := io.ReadAll(stream)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return string(logs), nil
}

func (c *Client) GetPodEvents(ctx context.Context, sandboxID string) ([]string, error) {
	podName := fmt.Sprintf("sandbox-%s", sandboxID)

	events, err := c.clientset.CoreV1().Events(SandboxNamespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", podName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	var result []string
	for _, event := range events.Items {
		result = append(result, fmt.Sprintf("[%s] %s: %s", event.Type, event.Reason, event.Message))
	}
	return result, nil
}

func boolPtr(b bool) *bool {
	return &b
}
