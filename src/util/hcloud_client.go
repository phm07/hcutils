package util

import (
	"context"
	"errors"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"net"
	"os"
	"time"
)

func InitHCloudClient() (*hcloud.Client, error) {
	token := os.Getenv("HCLOUD_TOKEN")
	if token == "" {
		return nil, errors.New("HCLOUD_TOKEN environment variable is not set")
	}
	return hcloud.NewClient(hcloud.WithToken(token)), nil
}

func WaitForAction(ctx context.Context, client *hcloud.Client, action *hcloud.Action) error {
	var err error
	for action.Status == hcloud.ActionStatusRunning {
		action, _, err = client.Action.GetByID(ctx, action.ID)
		if err != nil {
			return err
		}
	}
	if action.Status == hcloud.ActionStatusError {
		return errors.New(action.ErrorMessage)
	}
	return nil
}

func WaitForServerStart(server *hcloud.Server) error {
	start := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if time.Since(start) > time.Minute {
			return errors.New("timeout waiting for server to start")
		}
		conn, _ := net.DialTimeout("tcp", server.PublicNet.IPv4.IP.String()+":22", time.Second)
		if conn != nil {
			_ = conn.Close()
			break
		}
	}
	return nil
}
