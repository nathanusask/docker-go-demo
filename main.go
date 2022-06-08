package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/docker/docker/api/types/container"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
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

	imageID, err := BuildFactor(ctx, cli, poc, POC)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("build successful, image ID:", imageID)

	paramArgs := []string{
		"--task_id", "fake_task_id",
		"--host", "47.96.164.104",
		"--port", "27017",
		"--database", "quant",
		"--collection", "swap.eth",
		"--interval", "1min",
	}
	if err := RunFactor(ctx, cli, strings.ToLower(poc.FactorName), paramArgs); err != nil {
		log.Fatal(err)
	}
}

func BuildFactor(ctx context.Context, cli *client.Client, factor Factor, templateStr string) (string, error) {
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
		return "", err
	}

	dirname := strings.ToLower(factor.FactorName)
	if err := os.Mkdir(dirname, os.ModePerm); err != nil {
		return "", err
	}
	defer os.RemoveAll(dirname)
	fileMain, err := os.Create(path.Join(dirname, "main.py"))
	if err != nil {
		return "", err
	}
	defer fileMain.Close()
	if err = templ.Execute(fileMain, factor); err != nil {
		return "", err
	}

	fileFactor, err := os.Create(path.Join(dirname, factor.FactorName+".py"))
	if err != nil {
		return "", err
	}
	defer fileFactor.Close()
	if _, err = fileFactor.WriteString(templateStr); err != nil {
		return "", err
	}

	fileRequirements, err := os.Create(path.Join(dirname, "requirements.txt"))
	if err != nil {
		return "", err
	}
	defer fileRequirements.Close()
	if _, err = fileRequirements.WriteString(Requirements); err != nil {
		return "", err
	}

	fileDockerFile, err := os.Create(path.Join(dirname, "Dockerfile"))
	if err != nil {
		return "", err
	}
	defer fileDockerFile.Close()
	if _, err = fileDockerFile.WriteString(DockerfileTemplate); err != nil {
		return "", err
	}

	buildContext, err := archive.TarWithOptions(dirname, &archive.TarOptions{})
	if err != nil {
		return "", err
	}
	resp, err := cli.ImageBuild(ctx, buildContext, types.ImageBuildOptions{
		Tags:           []string{dirname},
		SuppressOutput: true, // so that we can obtain only ID or nothing
	})
	if err != nil {
		log.Fatal("failed at image build", err)
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	type response struct {
		Stream string `json:"stream"`
	}
	r := &response{}
	if err := json.Unmarshal(bytes, &r); err != nil {
		return "", err
	}

	id := strings.Split(strings.Trim(r.Stream, "\n"), ":")[1]
	log.Println("image ID:", id)
}
