package main

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

const (
	containerName    = "ceye-certstream"
	certstreamImage  = "0rickyy0/certstream-server-go:latest"
	containerPort    = "8080"
	hostPort         = "8888"
	websocketURL     = "ws://127.0.0.1:8888/"
	healthCheckURL   = "http://127.0.0.1:8888/example.json"
	startupTimeout   = 120 * time.Second
	healthCheckDelay = 2 * time.Second
)

type DockerManager struct {
	containerID string
}

func NewDockerManager() *DockerManager {
	return &DockerManager{}
}

func (d *DockerManager) IsDockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func (d *DockerManager) IsContainerRunning() bool {
	cmd := exec.Command("docker", "ps", "-q", "-f", fmt.Sprintf("name=%s", containerName))
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

func (d *DockerManager) ContainerExists() bool {
	cmd := exec.Command("docker", "ps", "-aq", "-f", fmt.Sprintf("name=%s", containerName))
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

func (d *DockerManager) PullImage() error {
	logger.Info("pulling certstream-server-go image", "image", certstreamImage)
	cmd := exec.Command("docker", "pull", certstreamImage)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func (d *DockerManager) ImageExists() bool {
	cmd := exec.Command("docker", "images", "-q", certstreamImage)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

func (d *DockerManager) StartContainer() error {
	if d.IsContainerRunning() {
		logger.Info("certstream container already running")
		return nil
	}

	if d.ContainerExists() {
		logger.Info("starting existing certstream container")
		cmd := exec.Command("docker", "start", containerName)
		return cmd.Run()
	}

	if !d.ImageExists() {
		if err := d.PullImage(); err != nil {
			return fmt.Errorf("failed to pull image: %w", err)
		}
	}

	logger.Info("creating certstream container")
	cmd := exec.Command("docker", "run",
		"-d",
		"--name", containerName,
		"-p", fmt.Sprintf("127.0.0.1:%s:%s", hostPort, containerPort),
		"--restart", "unless-stopped",
		certstreamImage,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create container: %w - %s", err, string(output))
	}

	d.containerID = strings.TrimSpace(string(output))
	return nil
}

func (d *DockerManager) StopContainer() error {
	if !d.IsContainerRunning() {
		return nil
	}

	logger.Info("stopping certstream container")
	cmd := exec.Command("docker", "stop", containerName)
	return cmd.Run()
}

func (d *DockerManager) RemoveContainer() error {
	if d.IsContainerRunning() {
		if err := d.StopContainer(); err != nil {
			return err
		}
	}

	if !d.ContainerExists() {
		return nil
	}

	logger.Info("removing certstream container")
	cmd := exec.Command("docker", "rm", containerName)
	return cmd.Run()
}

func (d *DockerManager) GetLogs(lines int) (string, error) {
	cmd := exec.Command("docker", "logs", "--tail", fmt.Sprintf("%d", lines), containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (d *DockerManager) WaitForReady(ctx context.Context) error {
	logger.Info("waiting for certstream server to be ready")

	ticker := time.NewTicker(healthCheckDelay)
	defer ticker.Stop()

	timeout := time.After(startupTimeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			logs, _ := d.GetLogs(20)
			return fmt.Errorf("timeout waiting for certstream server to start. logs:\n%s", logs)
		case <-ticker.C:
			if d.isServerReady() {
				logger.Info("certstream server is ready")
				return nil
			}
		}
	}
}

func (d *DockerManager) isServerReady() bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%s", hostPort), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (d *DockerManager) GetWebSocketURL() string {
	return websocketURL
}

func (d *DockerManager) EnsureRunning(ctx context.Context) error {
	if !d.IsDockerAvailable() {
		return fmt.Errorf("docker is not available. please install docker and ensure it's running")
	}

	if err := d.StartContainer(); err != nil {
		return err
	}

	return d.WaitForReady(ctx)
}
