package runtime

import (
	"context"
	"io"
	"time"

	"github.com/Ender-events/koncord/internal/domain"
)

// ContainerRuntime abstracts container management operations.
// Implementations must only expose containers labelled koncord.enable=true.
type ContainerRuntime interface {
	// ListContainers returns all koncord-enabled containers.
	ListContainers(ctx context.Context) ([]domain.ContainerInfo, error)

	// GetContainer returns a single container by name or ID.
	GetContainer(ctx context.Context, nameOrID string) (*domain.ContainerInfo, error)

	// RestartContainer restarts a container by name or ID.
	RestartContainer(ctx context.Context, nameOrID string) error

	// StreamLogs returns a reader that streams container logs from the given
	// time onward. The caller must close the returned reader.
	StreamLogs(ctx context.Context, nameOrID string, since time.Time) (io.ReadCloser, error)
}
