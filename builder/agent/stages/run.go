package stages

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// Run starts the user-supplied image with the cloned repo mounted read-only at
// /repo and outputDir mounted read-write at /output. It waits for the container
// to exit and enforces timeout as a hard cap. The container is removed on
// return regardless of outcome. The combined stdout+stderr log output is always
// returned (even on error) so callers can surface it.
func Run(image, repoDir, outputDir string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cli, err := client.New(client.FromEnv)
	if err != nil {
		return "", fmt.Errorf("docker client: %w", err)
	}
	defer cli.Close()

	createResp, err := cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{Image: image},
		HostConfig: &container.HostConfig{
			Binds: []string{
				repoDir + ":/repo:ro",
				outputDir + ":/output:rw",
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("container create: %w", err)
	}

	defer cli.ContainerRemove(context.Background(), createResp.ID, client.ContainerRemoveOptions{Force: true}) //nolint:errcheck

	if _, err := cli.ContainerStart(ctx, createResp.ID, client.ContainerStartOptions{}); err != nil {
		return "", fmt.Errorf("container start: %w", err)
	}

	waitResult := cli.ContainerWait(ctx, createResp.ID, client.ContainerWaitOptions{
		Condition: container.WaitConditionNotRunning,
	})

	var waitErr error
	select {
	case err := <-waitResult.Error:
		if ctx.Err() != nil {
			cli.ContainerKill(context.Background(), createResp.ID, client.ContainerKillOptions{Signal: "SIGKILL"}) //nolint:errcheck
			waitErr = fmt.Errorf("container exceeded timeout of %v", timeout)
		} else {
			waitErr = fmt.Errorf("container wait: %w", err)
		}
	case status := <-waitResult.Result:
		if status.StatusCode != 0 {
			waitErr = fmt.Errorf("container exited with code %d", status.StatusCode)
		}
	}

	// Gather logs after the container has stopped (or been killed) so the full
	// output is available.  Use a background context: the timeout only covers
	// the container run, not the log read.
	logs := gatherLogs(cli, createResp.ID)

	return logs, waitErr
}

// gatherLogs reads the combined stdout+stderr from a container.  It never
// fails hard — on any error it returns whatever partial data was read.
func gatherLogs(cli *client.Client, containerID string) string {
	rc, err := cli.ContainerLogs(context.Background(), containerID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return ""
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return string(data) // return whatever was read before the error
	}
	return string(data)
}
