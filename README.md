# Basic Auth Checker

Tool to verify Basic Auth status of multiple given endpoints, via TOML configuration file.

## Get going


    go get github.com/eripa/ba_checker
    cp config-example.toml config.toml

    # Edit config.toml to your liking
    ba_checker --no-spinner config.toml


If any check results in `false` in the success column, the tool will exit with 1, 2 or 3 depending on the severity (define with --warning/--critical).

### Usage

    Usage: ba_checker [--warning=<number>] [--critical=<number>] [--output=<table|nagios>] [--no-spinner] CONFIGFILE

    Check HTTP Basic Auth status

    Status can be determined by Exit codes:
     0=Status OK
     1=Above warning threshold
     2=Above critical threshold
     3=Unknown Basic Auth status (4xx or 5xx HTTP codes)

    Arguments:
      CONFIGFILE=""   Config file

    Options:
      -v, --version          Show the version and exit
      --no-spinner=false     Disable spinner animation
      -o, --output="table"   Output format, available formats: table, nagios
      -w, --warning=1        Warning threshold
      -c, --critical=2       Critical threshold

## Example runs

### Success run

    ba_checker --no-spinner config-example.toml
                          URL                      | Basic Auth | Wanted BA | Success |        HTTP Status
    +----------------------------------------------+------------+-----------+---------+----------------------------+
      https://httpbin.org/                         | no         | no        | true    | 200 OK
      https://httpbin.org/basic-auth/:user/:passwd | yes        | yes       | true    | 401 UNAUTHORIZED
      https://httpbin.org/html                     | no         | no        | true    | 200 OK
      http://test.webdav.org/                      | no         | no        | true    | 200 OK
      http://test.webdav.org/auth-basic            | yes        | yes       | true    | 401 Authorization Required
      http://test.webdav.org/dav                   | unknown    | no        | true    | 404 Not Found
    +----------------------------------------------+------------+-----------+---------+----------------------------+

    Status: OK

    echo $?
    0

### Critical threshold set to 1

    ba_checker --critical 1 --no-spinner config-example.toml
                          URL                      | Basic Auth | Wanted BA | Success |        HTTP Status
    +----------------------------------------------+------------+-----------+---------+----------------------------+
      https://httpbin.org/                         | no         | no        | true    | 200 OK
      https://httpbin.org/basic-auth/:user/:passwd | yes        | yes       | true    | 401 UNAUTHORIZED
      https://httpbin.org/html                     | no         | no        | true    | 200 OK
      http://test.webdav.org/                      | no         | yes       | false   | 200 OK
      http://test.webdav.org/                      | no         | no        | true    | 200 OK
      http://test.webdav.org/auth-basic            | yes        | yes       | true    | 401 Authorization Required
    +----------------------------------------------+------------+-----------+---------+----------------------------+

    Status: CRITICAL

    echo $?
    2

### Thresholds unset, default Warning threshold is 1. Nagios output format

    ba_checker --no-spinner --output nagios config-example.toml
    BA check: WARNING - OK: 5/6

    echo $?
    1

### Notes

Status `UNKNOWN` is only flagged if Basic Auth was wanted, but a HTTP Code other than 401 was encountered. If Critical threshold is exceeded and there is an unknown, `CRITICAL` will be the end-status. `UNKNOWN` will be the end-status if there is an unknown while Warning threshold is exceeded. `WARNING` will be shown if there is no unknown and failures is below criical threshold.

Priority is 1) critical 2) unknown 3) warning

## Example config file

Same as used in `go test` test cases

```toml
[[site]]
base = "https://httpbin.org"
auth = ["basic-auth/:user/:passwd"]
no_auth = ["html", ""]

[[site]]
base = "http://test.webdav.org"
auth = [
  "auth-basic"
]
no_auth = [
  "dav",
  ""
]
```

# License

BSD 2-Clause License

Copyright (c) 2016, Eric Ripa <eric@ripa.io>

see [LICENSE](LICENSE) file for full details
