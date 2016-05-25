# Basic Auth Checker

Tool to verify Basic Auth status of multiple given endpoints, via JSON configuration file.

## Usage


    go get github.com/eripa/ba_checker
    cp config-example.json config.json

    # Edit config.json to your liking
    ba_checker --config config.json


If any check results in `false` in the success column, the tool will exit with 1.

## Example run

                                               URL | Basic Auth |  Wanted BA |    Success | HTTP Status
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

```json
{
  "sites": [
    {
      "base": "http://test.webdav.org",
      "endpoints": {
        "auth-basic": true,
        "dav":        false,
        "":           false
      }
    },
    {
      "base": "https://httpbin.org/",
      "endpoints": {
        "basic-auth/:user/:passwd ": true,
        "html": false,
        "":     false
      }
    }
  ]
}
```

# License

See LICENSE file
