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
		for endpoint, shouldBe := range site.Endpoints {
			baEnabled, err := checkEndpoint(site.Base, endpoint, shouldBe)
			errorMessage := fmt.Sprintf("ERROR: %s/%s incorrect. Basic Auth Enabled: %t Should be: %t\n", site.Base, endpoint, baEnabled, shouldBe)
			if err != nil {
				errorMessage = fmt.Sprintf("ERROR: %s/%s unknown. got HTTP Status Code above 400\n", site.Base, endpoint)
			}
			if verboseMode {
				log.Printf("Basic Auth enabled: %t\n", baEnabled)
			}
			success := baEnabled == shouldBe
			if !success {
				if verboseMode {
					log.Printf(errorMessage)
				} else {
					fmt.Printf(errorMessage)
				}
			}
		}
	}
	return nil
}

func checkEndpoint(site string, endpoint string, shouldBe bool) (bool, error) {
	if verboseMode {
		log.Printf("Checking endpoint %v, Basic Auth should be %v\n", endpoint, shouldBe)
	}
	URL := fmt.Sprintf("%s/%s", site, endpoint)
	resp, err := http.Get(URL)
	if resp.StatusCode >= 400 {
		err = fmt.Errorf("Unexpected HTTP Status Code, got: %d", resp.StatusCode)
		return false, err
	}
	baActive := false
	if resp.Header.Get("Www-Authenticate") != "" {
		baActive = true
	}
	return baActive, err
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
		checkSites(config.Sites)
		return nil
	})
}
