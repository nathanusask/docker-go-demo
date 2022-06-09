```go
type Factor struct {
	FactorName string
	Description string
	ParamTypes []struct{
		Name string
		Type string
    }
}
```

```python
import argparse
{{ .FactorCode }}
from pymongo import MongoClient

parser = argparse.ArgumentParser(description="{{ .Description }}")
{{ range .ParamTypes }}parser.add_argument("--{{ .Name }}", type={{ .Type }}){{"\n"}}{{ end }}
parser.add_argument("--task_id")
parser.add_argument("--host", default="host.docker.internal")
parser.add_argument("--port", type=int, default=27017)
parser.add_argument("--database", default="quant")
parser.add_argument("--collection")
parser.add_argument("--start", type=int, default=0)
parser.add_argument("--end", type=int, default=-1)

args = parser.parse_args()

mongo_client = MongoClient(host=args.host, port=args.port)

# get data
def get_data(database, collection, start, end):
    db = mongo_client[database]
    coll = db[collection]
    pipeline = [{'$project': {'_id': 0}}]
    if start < end:
        filter = {"$match": {"ts": {"$gt": start, "$lt": end}}}
        pipeline.append(filter)
    return coll.aggregate(pipeline)

# handle result
def handle_result(result, database, collection):
    assert isinstance(result, pd.DataFrame)
    db = mongo_client[database]
    coll = db[collection]
    coll.insert_many(result.to_dict("records"))

data = get_data(args.database, args.collection, args.start, args.end)

result = {{ .FactorName }}(data, {{ assignParamArg .ParamTypes | join ", "}})

# handle result
output_collection = ".".join([args.task_id, "{{ .FactorName }}"])
handle_result(result, args.database, output_collection)

mongo_client.close()
```

```python
import pandas as pd
import datetime
import re

def separate_str_num(s):
    pattern = '(\d+|[A-Za-z]+)'
    return re.findall(pattern, s)

def MACD(data, interval='1D', fast=12, slow=26, dea=9):
    df_all = pd.concat(data)
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
```

```dockerfile
FROM python:3.10

WORKDIR /app
COPY . .

RUN pip install -r requirements.txt
```