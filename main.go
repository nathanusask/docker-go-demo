package main

import (
	"context"
	"io"
	"log"
	"os"

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

	//bytes, err := ioutil.ReadAll(resp.Body)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//idResp := &types.IDResponse{}
	//if err := json.Unmarshal(bytes, &idResp); err != nil {
	//	log.Fatal(err)
	//}
	//log.Println("image ID:", idResp.ID)
	io.Copy(os.Stdout, resp.Body)

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
