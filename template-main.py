import argparse
import pandas as pd
import datetime
import re

def separate_str_num(s):
    pattern = '(\d+|[A-Za-z]+)'
    return re.findall(pattern, s)

def max_amount_price(group):
    POC = group.loc[group['amount'] == group['amount'].max(), 'price'].mean()
    return pd.Series([POC], ('POC',))

def POC(data, interval='1D'):
    df_all = pd.DataFrame(data)
    df_all['datetime'] = pd.to_datetime(df_all['ts'], unit='ms')
    df = df_all.groupby(pd.Grouper(key='datetime', freq=interval)).apply(max_amount_price).reset_index()
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
    return df[['datetime', 'POC']]


from pymongo import MongoClient

parser = argparse.ArgumentParser(description="Price Open Close")
parser.add_argument("--interval", type=str)

parser.add_argument("--task_id")
parser.add_argument("--host", default="host.internal.docker")
parser.add_argument("--port", type=int, default=27017)
parser.add_argument("--database", default="quant")
parser.add_argument("--collection")

args = parser.parse_args()

mongo_client = MongoClient(host=args.host, port=args.port)

# get data
def get_data(database, collection):
    db = mongo_client[database]
    coll = db[collection]
    pipeline = [
        {
            '$project': {
                '_id': 0,
             }
        }
    ]
    return coll.aggregate(pipeline)


# handle result
def handle_result(result, database, collection):
    assert isinstance(result, pd.DataFrame)
    db = mongo_client[database]
    coll = db[collection]
    coll.insert_many(result.to_dict("records"))

data = get_data(args.database, args.collection)

result = POC(data, interval=args.interval)

print(result)

mongo_client.close()