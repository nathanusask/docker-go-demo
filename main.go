package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

func main() {
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
	templ, err := template.New("factor").Funcs(funcs).Parse(PythonMainTemplate)
	if err != nil {
		log.Fatal(err)
	}

	// example usage
	macd := Factor{
		FactorName:  "MACD",
		Description: "MACD",
		ParamTypes: []ParamType{
			{
				Name: "interval",
				Type: "str",
			},
			{
				Name: "fast",
				Type: "int",
			},
			{
				Name: "slow",
				Type: "int",
			},
			{
				Name: "dea",
				Type: "int",
			},
		},
	}
	dirname := strings.ToLower(macd.FactorName)
	if err := os.Mkdir(dirname, os.ModePerm); err != nil {
		log.Fatal(err)
	}
	fileMain, err := os.Create(path.Join(dirname, "main.py"))
	if err != nil {
		log.Fatal(err)
	}
	defer fileMain.Close()
	if err = templ.Execute(fileMain, macd); err != nil {
		log.Fatal(err)
	}

	fileFactor, err := os.Create(path.Join(dirname, macd.FactorName+".py"))
	if err != nil {
		log.Fatal(err)
	}
	defer fileFactor.Close()
	if _, err = fileFactor.WriteString(MACD); err != nil {
		log.Fatal(err)
	}

	fileRequirements, err := os.Create(path.Join(dirname, "requirements.txt"))
	if err != nil {
		log.Fatal(err)
	}
	defer fileRequirements.Close()
	if _, err = fileRequirements.WriteString(Requirements); err != nil {
		log.Fatal(err)
	}

	fileDockerFile, err := os.Create(path.Join(dirname, "Dockerfile"))
	if err != nil {
		log.Fatal(err)
	}
	defer fileDockerFile.Close()
	if _, err = fileDockerFile.WriteString(DockerfileTemplate); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	buildContext, err := archive.TarWithOptions(dirname, &archive.TarOptions{})
	if err != nil {
		log.Fatal("failed at TarWithOptions", err)
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
}
