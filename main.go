package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/docker/docker/api/types/mount"

	"github.com/docker/docker/api/types/container"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {

	// example usage
	//macd := Factor{
	//	FactorName:  "MACD",
	//	Description: "MACD",
	//	ParamTypes: []ParamType{
	//		{
	//			Name: "interval",
	//			Type: "str",
	//		},
	//		{
	//			Name: "fast",
	//			Type: "int",
	//		},
	//		{
	//			Name: "slow",
	//			Type: "int",
	//		},
	//		{
	//			Name: "dea",
	//			Type: "int",
	//		},
	//	},
	//}

	poc := Factor{
		FactorName:  "POC",
		FactorCode:  POC,
		Description: "Price Open Close",
		ParamTypes: []ParamType{
			{
				Name: "interval",
				Type: "str",
			},
		},
	}

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	//imageID, err := BuildFactor(ctx, cli, macd, MACD)
	//if err != nil {
	//	log.Fatal(err)
	//}

	err = BuildFactor(poc)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("build successful")

	paramArgs := []string{
		"--task_id", "fake_task_id",
		"--collection", "swap.eth.simplified",
		"--interval", "1min",
	}
	if err := RunFactor(ctx, cli, strings.ToLower(poc.FactorName), paramArgs); err != nil {
		log.Fatal("failed to run", err)
	}
}

func BuildFactor(factor Factor) error {
	assignParamArg := func(pts []ParamType) []string {
		var ret []string
		for _, pt := range pts {
			ret = append(ret, fmt.Sprintf("%s=args.%s", pt.Name, pt.Name))
		}
		return ret
	}

	join := func(sep string, elem []string) string {
		return strings.Join(elem, sep)
	}

	funcs := template.FuncMap{"assignParamArg": assignParamArg, "join": join}
	templ, err := template.New(factor.FactorName).Funcs(funcs).Parse(PythonMainTemplate)
	if err != nil {
		return err
	}

	dirname := strings.ToLower(factor.FactorName)
	if err := os.Mkdir(dirname, os.ModePerm); err != nil {
		return err
	}

	fileMain, err := os.Create(path.Join(dirname, "main.py"))
	if err != nil {
		return err
	}
	defer fileMain.Close()
	if err = templ.Execute(fileMain, factor); err != nil {
		return err
	}

	return nil
}

func RunFactor(ctx context.Context, cli *client.Client, factorNameLowercase string, paramArgs []string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()

	pwd, _ := os.Getwd()
	src := path.Join(pwd, factorNameLowercase, "main.py")
	dst := "/app/main.py"

	body, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "poc", // TODO: change it to a fixed image name
		Cmd:   append([]string{"python", dst}, paramArgs...),
	}, &container.HostConfig{
		AutoRemove: true,
		ExtraHosts: []string{"host.docker.internal:host-gateway"},
		Mounts: []mount.Mount{
			{
				Source: src,
				Target: dst,
			},
		},
	}, nil, nil, factorNameLowercase)
	if err != nil {
		return err
	}

	containerID := body.ID
	log.Println("container ID:", containerID)

	if err := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	bodyChan, errCh := cli.ContainerWait(ctx, body.ID, container.WaitConditionRemoved)
	select {
	case err := <-errCh:
		return err
	case b := <-bodyChan:
		if b.Error != nil {
			log.Println(b.StatusCode, b.Error)
			return errors.New(b.Error.Message)
		}
		log.Println(b.StatusCode)
		return nil
	case <-ctx.Done():
		return errors.New("context timeout")
	}
}
