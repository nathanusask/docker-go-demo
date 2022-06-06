package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"

	"github.com/docker/docker/pkg/archive"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	buildContext, err := archive.TarWithOptions(".", &archive.TarOptions{})
	if err != nil {
		log.Fatal(err)
	}
	resp, err := cli.ImageBuild(ctx, buildContext, types.ImageBuildOptions{
		Tags:           []string{"docker-go-demo:v0.1"},
		SuppressOutput: true, // so that we can obtain only ID or nothing
	})
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	type response struct {
		Stream string `json:"stream"`
	}
	r := &response{}
	if err := json.Unmarshal(bytes, &r); err != nil {
		log.Fatal(err)
	}

	id := strings.Split(strings.Trim(r.Stream, "\n"), ":")[1]
	log.Println("image ID:", id)
	//io.Copy(os.Stdout, resp.Body)

	//reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
	//if err != nil {
	//	log.Fatal(err)
	//}
	//io.Copy(os.Stdout, reader)
	//
	//resp, err := cli.ContainerCreate(ctx, &container.Config{
	//	Image: "alpine",
	//	Cmd:   []string{"echo", "hello world"},
	//}, nil, nil, nil, "")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
	//	log.Fatal(err)
	//}
	//
	//statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	//select {
	//case err := <-errCh:
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//case <-statusCh:
	//}
	//
	//out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	//if err != nil {
	//	log.Fatal(err)
	//}

	//stdcopy.StdCopy(os.Stdout, os.Stderr, out)
}
