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

type ParamType struct {
	Name string
	Type string
}

type Factor struct {
	FactorName  string
	Description string
	ParamTypes  []ParamType
}

const PythonMainTemplate = `import argparse
import pandas as pd
from pymongo import MongoClient
mongo_client = MongoClient(host=args.host, port=args.port)

parser = argparse.ArgumentParser(description="{{ .Description }}")
{{ range .ParamTypes }}parser.add_argument("--{{ .Name }}", type={{ .Type }}){{"\n"}}{{ end }}
parser.add_argument("--task_id")
parser.add_argument("--host")
parser.add_argument("--port")
parser.add_argument("--database")
parser.add_argument("--collection")
parser.add_argument("--start", type=int, default=0)
parser.add_argument("--end", type=int, default=-1)

args = parser.parse_args()

# get data
def get_data(database, collection, start, end):
    db = mongo_client[database]
    coll = db[collection]
    filter = {}
    if start < end:
        filter = {"ts": {"$gt": start, "$lt": end}}
    return coll.find(filter)

# handle result
def handle_result(result, database, collection):
    assert isinstance(result, pd.DataFrame)
    db = mongo_client[database]
    coll = db[collection]
    coll.insert_many(result.to_dict("records"))

data = get_data(args.database, args.collection, args.start, args.end)

from {{ .FactorName }} import {{ .FactorName }}

result = {{ .FactorName }}(data, {{ assignParamArg .ParamTypes | join ", "}})

# handle result
output_collection = ".".join([args.task_id, "{{ .FactorName }}"])
handle_result(result, database, output_collection)

mongo_client.close()
`
const MACD = `import pandas as pd
import datetime
import re

def separate_str_num(s):
    pattern = '(\d+|[A-Za-z]+)'
    return re.findall(pattern, s)

def MACD(data, interval='1D', fast=12, slow=26, dea=9):
    df_all = pd.DataFrame(data)
    df = df_all
        .groupby(pd.Grouper(key='datetime', freq=interval))
        .agg(close=pd.NamedAgg(column='price', aggfunc='last'))
        .reset_index()
    duration, interval_type = separate_str_num(interval)
    duration = int(duration)

    if interval_type == 's':
        df['datetime'] += datetime.timedelta(seconds=duration)
    elif interval_type == 'min':
        df['datetime'] += datetime.timedelta(minutes=duration)
    elif interval_type == 'h':
        df['datetime'] += datetime.timedelta(hours=duration)
    elif interval_type == 'W':
        df['datetime'] = df['datetime'].dt.date + datetime.timedelta(days=7)
    elif interval_type == 'SM':
        df['datetime'] = df['datetime'].dt.date + datetime.timedelta(days=15)
    elif interval_type == 'D' or interval_type == 'M':
        df['datetime'] = df['datetime'].dt.date
    exp1 = df['close'].ewm(span=slow, adjust=False).mean()
    exp2 = df['close'].ewm(span=fast, adjust=False).mean()
    df['Diff']=exp1-exp2
    df['DEA'] = df['Diff'].ewm(span=dea, adjust=False).mean()
    df['MACD'] = 2 * (df['Diff']- df['DEA'])
    return df[['datetime','Diff','DEA','MACD']]
`
const DockerfileTemplate = `FROM python:3.10

WORKDIR /app
COPY . .

RUN pip install -r requirements.txt
`

const Requirements = `pymongo
pandas
`

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
	if err := os.Mkdir(macd.FactorName, os.ModePerm); err != nil {
		log.Fatal(err)
	}
	fileMain, err := os.Create(path.Join(macd.FactorName, "main.py"))
	if err != nil {
		log.Fatal(err)
	}
	defer fileMain.Close()
	if err = templ.Execute(fileMain, macd); err != nil {
		log.Fatal(err)
	}

	fileFactor, err := os.Create(path.Join(macd.FactorName, macd.FactorName+".py"))
	if err != nil {
		log.Fatal(err)
	}
	defer fileFactor.Close()
	if _, err = fileFactor.WriteString(MACD); err != nil {
		log.Fatal(err)
	}

	fileRequirements, err := os.Create(path.Join(macd.FactorName, "requirements.txt"))
	if err != nil {
		log.Fatal(err)
	}
	defer fileRequirements.Close()
	if _, err = fileRequirements.WriteString(Requirements); err != nil {
		log.Fatal(err)
	}

	dockerfileTempl, err := template.New("dockerfile").Parse(DockerfileTemplate)
	if err != nil {
		log.Fatal(err)
	}
	dockerfileName := macd.FactorName + "-Dockerfile"
	fileDockerFile, err := os.Create(dockerfileName)
	if err != nil {
		log.Fatal(err)
	}
	defer fileDockerFile.Close()
	if err = dockerfileTempl.Execute(fileDockerFile, macd); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	buildContext, err := archive.TarWithOptions(macd.FactorName, &archive.TarOptions{})
	if err != nil {
		log.Fatal(err)
	}
	resp, err := cli.ImageBuild(ctx, buildContext, types.ImageBuildOptions{
		Tags:           []string{macd.FactorName},
		Dockerfile:     dockerfileName,
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
}
