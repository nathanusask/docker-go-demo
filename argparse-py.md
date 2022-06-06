```python
import argparse

parser = argparse.ArgumentParser(description={description})
parser.add_argument("--{input_i}", type={type_i}, help="{description_i}")
...
args = parser.parse_args()

from {filename} import {factor}

factor(args.{arg1}, args.{arg2}, ...)
```