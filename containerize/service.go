package containerize

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"

	"github.com/docker/docker/client"
)

const (
	pythonMainFilename = "main.py"
	dstPath            = "/app/main.py"
)

type server struct {
	cli *client.Client
}

func (s server) RunFactor(ctx context.Context, baseImage string, code string, factorNameLowercase string, paramArgs []string) error {
	if err := os.MkdirAll(factorNameLowercase, os.ModePerm); err != nil {
		log.Println("[Error] failed to create dir with error", err.Error())
		return err
	}
	pythonFilepath := path.Join(factorNameLowercase, pythonMainFilename)
	f, err := os.OpenFile(pythonFilepath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		log.Println("[Error] failed to create file with error", err.Error())
		return err
	}
	defer f.Close()
	if _, err = f.WriteString(code); err != nil {
		log.Println("[Error] failed to write code with error", err.Error())
		return err
	}

	pwd, err := os.Getwd()
	if err != nil {
		log.Println("[Error] failed to get the current directory with error", err.Error())
		return err
	}
	src := path.Join(pwd, pythonFilepath)
	body, err := s.cli.ContainerCreate(ctx, &container.Config{
		Cmd:   append([]string{"python", dstPath}, paramArgs...),
		Image: baseImage,
	}, &container.HostConfig{
		AutoRemove: true,
		ExtraHosts: []string{"host.docker.internal:host-gateway"},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: src,
				Target: dstPath,
			},
		},
	}, nil, nil, factorNameLowercase)
	if err != nil {
		log.Println("[Error] failed to create container with error", err.Error())
		return err
	}

	containerID := body.ID
	log.Println("[Info] container ID:", containerID)

	if err = s.cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		log.Println("[Error] failed to start container with error", err.Error())
		return err
	}
	output, err := s.cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{})
	if err != nil {
		log.Println("[Error] failed to get logs for container", containerID, "with error", err.Error())
		return err
	}
	if _, err := io.Copy(os.Stdout, output); err != nil {
		log.Println("[Error] failed to copy container output to stdout with error", err.Error())
		return err
	}

	bodyChan, errCh := s.cli.ContainerWait(ctx, containerID, container.WaitConditionRemoved)
	select {
	case err = <-errCh:
		log.Println("[Error] failed to wait for container to finish with error", err.Error())
		return err
	case b := <-bodyChan:
		if b.Error != nil {
			log.Println("[Error] error occurred", b.Error.Message)
			return errors.New(b.Error.Message)
		}
		log.Println("[Info] container finished and return status", b.StatusCode)
		return nil
	}
}

func New(c *client.Client) Interface {
	return &server{c}
}
