package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/moby/moby/client"

	sdkclient "github.com/docker/go-sdk/client"

	"github.com/Ender-events/koncord/internal/domain"
	"github.com/Ender-events/koncord/internal/runtime"
	"github.com/moby/moby/api/types/container"
)

const labelFilter = "koncord.enable=true"

// Client implements runtime.ContainerRuntime using the Docker go-sdk.
type Client struct {
	api sdkclient.SDKClient
}

// Ensure interface compliance at compile time.
var _ runtime.ContainerRuntime = (*Client)(nil)

// New creates a new Docker runtime client using the go-sdk.
func New(ctx context.Context) (*Client, error) {
	cli, err := sdkclient.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &Client{api: cli}, nil
}

// Close closes the underlying Docker client.
func (c *Client) Close() error {
	return c.api.Close()
}

// ListContainers returns all containers with the koncord.enable=true label.
func (c *Client) ListContainers(ctx context.Context) ([]domain.ContainerInfo, error) {
	containers, err := c.api.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: make(client.Filters).Add("label", labelFilter),
	})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	result := make([]domain.ContainerInfo, 0, len(containers.Items))
	for _, ctr := range containers.Items {
		result = append(result, toContainerInfo(ctr))
	}
	return result, nil
}

// GetContainer returns a single container by name or ID.
func (c *Client) GetContainer(ctx context.Context, nameOrID string) (*domain.ContainerInfo, error) {
	containers, err := c.ListContainers(ctx)
	if err != nil {
		return nil, err
	}
	for _, ctr := range containers {
		if ctr.Name == nameOrID || ctr.ID == nameOrID || strings.HasPrefix(ctr.ID, nameOrID) {
			return &ctr, nil
		}
	}
	return nil, fmt.Errorf("container %q not found or not labelled koncord.enable=true", nameOrID)
}

// RestartContainer restarts a container by name or ID.
func (c *Client) RestartContainer(ctx context.Context, nameOrID string) error {
	ctr, err := c.GetContainer(ctx, nameOrID)
	if err != nil {
		return err
	}
	timeout := 30 // seconds
	_, err = c.api.ContainerRestart(ctx, ctr.ID, client.ContainerRestartOptions{Timeout: &timeout})
	return err
}

// StreamLogs returns a reader streaming container logs.
func (c *Client) StreamLogs(ctx context.Context, nameOrID string, since time.Time) (io.ReadCloser, error) {
	ctr, err := c.GetContainer(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	return c.api.ContainerLogs(ctx, ctr.ID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Since:      since.Format(time.RFC3339),
	})
}

func toContainerInfo(ctr container.Summary) domain.ContainerInfo {
	name := ""
	if len(ctr.Names) > 0 {
		name = strings.TrimPrefix(ctr.Names[0], "/")
	}
	return domain.ContainerInfo{
		ID:     ctr.ID[:12],
		Name:   name,
		Image:  ctr.Image,
		Status: string(ctr.State),
		State:  ctr.Status,
	}
}
