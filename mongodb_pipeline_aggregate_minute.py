pipeline = [
    {
        '$project': {
            '_id': 0,
            'price': {
                '$toDouble': '$price'
            },
            'ts': {
                '$multiply': [
                    60000, {
                        '$floor': {
                            '$divide': [
                                '$ts', 60000
                            ]
                        }
                    }
                ]
            },
            'buy_amount': {
                '$cond': [
                    {
                        '$eq': [
                            '$direction', 'buy'
                        ]
                    }, {
                        '$toDouble': '$amount'
                    }, 0
                ]
            },
            'sell_amount': {
                '$cond': [
                    {
                        '$eq': [
                            '$direction', 'sell'
                        ]
                    }, {
                        '$toDouble': '$amount'
                    }, 0
                ]
            }
        }
    }, {
        '$project': {
            'price': 1,
            'date': {
                '$toDate': '$ts'
            },
            'buy_amount': 1,
            'sell_amount': 1
        }
    }, {
        '$group': {
            '_id': '$date',
            'price': {
                '$avg': '$price'
            },
            'buy_amount': {
                '$sum': '$buy_amount'
            },
            'sell_amount': {
                '$sum': '$sell_amount'
            }
        }
    }, {
        '$sort': {
            '_id': 1
        }
    }, {
        '$project': {
            'date': '$_id',
            '_id': 0,
            'price': 1,
            'buy_amount': 1,
            'sell_amount': 1
        }
    }
]