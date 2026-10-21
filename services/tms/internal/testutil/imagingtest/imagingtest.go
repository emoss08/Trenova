package imagingtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/imaging"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

const (
	EndpointEnv  = "TRENOVA_IMAGING_TEST_ENDPOINT"
	ImageEnv     = "TRENOVA_IMAGING_TEST_IMAGE"
	DefaultImage = "trenova-imaging:local"
)

var (
	once     sync.Once
	endpoint string
	startErr error
)

func Endpoint(t testing.TB) string {
	t.Helper()

	if value := strings.TrimSpace(os.Getenv(EndpointEnv)); value != "" {
		return value
	}
	once.Do(func() {
		endpoint, startErr = start(context.Background())
	})
	if startErr != nil {
		t.Fatalf("start the imaging worker: %v", startErr)
	}
	return endpoint
}

func Client(t testing.TB) kernelports.Imaging {
	t.Helper()

	client, err := imaging.Dial(&config.ImagingConfig{
		Endpoint:       "passthrough:///" + Endpoint(t),
		InitialBackoff: 50 * time.Millisecond,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("dial the imaging worker: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func start(ctx context.Context) (string, error) {
	image := strings.TrimSpace(os.Getenv(ImageEnv))
	if image == "" {
		image = DefaultImage
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        image,
			Cmd:          []string{"serve"},
			ExposedPorts: []string{"50051/tcp", "9464/tcp"},
			Env: map[string]string{
				"TRENOVA_IMAGING__ENVIRONMENT":           "development",
				"TRENOVA_IMAGING__POOL__SIZE":            "2",
				"TRENOVA_IMAGING__TELEMETRY__LOG_FORMAT": "text",
			},
			WaitingFor: wait.ForHTTP("/readyz").WithPort("9464/tcp").WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
	})
	if err != nil {
		return "", err
	}
	host, err := container.Host(ctx)
	if err != nil {
		return "", err
	}
	port, err := container.MappedPort(ctx, "50051")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", host, port.Port()), nil
}

func Fixture(t testing.TB, name string) []byte {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for range 12 {
		candidate := filepath.Join(dir, "services", "imaging", "tests", "fixtures", name)
		if data, readErr := os.ReadFile(candidate); readErr == nil {
			return data
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("fixture %s not found above the working directory", name)
	return nil
}
