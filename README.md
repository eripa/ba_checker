# Basic Auth Checker

Tool to verify Basic Auth status of multiple given endpoints, via TOML configuration file.

## Usage


    go get github.com/eripa/ba_checker
    cp config-example.toml config.toml

    # Edit config.toml to your liking
    ba_checker --no-spinner config.toml


If any check results in `false` in the success column, the tool will exit with 1.

## Example run

                                               URL | Basic Auth |  Wanted BA |    Success | HTTP Status
    -----------------------------------------------+------------+------------+------------+---------------------------------
                 http://test.webdav.org/auth-basic |        yes |        yes |       true | 401 Authorization Required
                        http://test.webdav.org/dav |         no |         no |       true | 404 Not Found
                           http://test.webdav.org/ |         no |         no |       true | 200 OK
    -----------------------------------------------+------------+------------+------------+---------------------------------
                                               URL | Basic Auth |  Wanted BA |    Success | HTTP Status
    -----------------------------------------------+------------+------------+------------+---------------------------------
    https://httpbin.org//basic-auth/:user/:passwd  |        yes |        yes |       true | 401 UNAUTHORIZED
                         https://httpbin.org//html |         no |         no |       true | 200 OK
                             https://httpbin.org// |         no |         no |       true | 200 OK
    -----------------------------------------------+------------+------------+------------+---------------------------------


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

MIT license, see LICENSE file for full details
