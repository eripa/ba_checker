package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mkideal/cli"
)

const toolVersion = "0.0.2"

var verboseMode = false

type configuration struct {
	Sites []site
}

type site struct {
	Base      string
	Endpoints map[string]bool
}

type argT struct {
	cli.Helper
	Config  string `cli:"c,config" usage:"JSON config file, see config.json-template"`
	Verbose bool   `cli:"v,verbose" usage:"Verbose output"`
	Version bool   `cli:"version" usage:"Check version"`
}

func checkSites(sites []site) error {
	for _, site := range sites {
		if verboseMode {
			log.Printf("Checking site %s\n", site.Base)
		}
		for endpoint, baShouldBe := range site.Endpoints {
			response, err := checkEndpoint(site.Base, endpoint, baShouldBe)
			if err != nil {
				return err
			}
			success, baEnabled := checkSuccess(response, baShouldBe)
			var message string

			if success {
				message = fmt.Sprintf("OK: %s/%s correct. Basic Auth Enabled: %t Should be: %t\n", site.Base, endpoint, baEnabled, baShouldBe)
			} else {
				if response.StatusCode > 401 {
					message = fmt.Sprintf("ERROR: %s/%s unknown. %s\n", site.Base, endpoint, response.Status)
				} else {
					message = fmt.Sprintf("ERROR: %s/%s incorrect. Basic Auth Enabled: %t Should be: %t\n", site.Base, endpoint, baEnabled, baShouldBe)
				}
			}

			if verboseMode {
				log.Printf(message)
			} else {
				fmt.Printf(message)
			}
		}
	}
	return nil
}

func checkSuccess(response *http.Response, baShouldBe bool) (success bool, baEnabled bool) {
	if response.StatusCode == 401 {
		baEnabled = true
	}
	return baEnabled == baShouldBe, response.StatusCode == 401
}

func checkEndpoint(site string, endpoint string, baShouldBe bool) (*http.Response, error) {
	if verboseMode {
		log.Printf("Checking endpoint %v, Basic Auth should be %v\n", endpoint, baShouldBe)
	}
	URL := fmt.Sprintf("%s/%s", site, endpoint)
	resp, err := http.Get(URL)
	return resp, err
}

func main() {
	cli.Run(&argT{}, func(ctx *cli.Context) error {
		argv := ctx.Argv().(*argT)
		if argv.Version {
			ctx.String("ba_checker v%s\n", toolVersion)
			return nil
		}
		if argv.Config == "" {
			return fmt.Errorf("--config <config.json> is required.\n")
		}
		if argv.Verbose {
			verboseMode = true
		}
		if _, err := os.Stat(argv.Config); os.IsNotExist(err) {
			return fmt.Errorf("Error: %s does not exist", argv.Config)
		}
		file, _ := os.Open(argv.Config)
		decoder := json.NewDecoder(file)
		config := configuration{}
		err := decoder.Decode(&config)
		if err != nil {
			fmt.Println("error:", err)
		}
		err = checkSites(config.Sites)
		if err != nil {
			return err
		}
		return nil
	})
}
