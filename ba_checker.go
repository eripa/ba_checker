package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/mkideal/cli"
)

const toolVersion = "0.0.4"

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

func getMaxWidth(sites []site) (width int) {
	var URL string
	for _, site := range sites {
		for endpoint := range site.Endpoints {
			URL = fmt.Sprintf("%s/%s", site.Base, endpoint)
			if len(URL) > width {
				width = len(URL)
			}
		}
	}
	return width
}

func checkSites(sites []site) error {
	maxWidth := getMaxWidth(sites)
	if !verboseMode {
		fmt.Printf("%*s | %*s | %*s | HTTP Status\n%s-+-%s-+-%s-+-%s\n", maxWidth, "URL", 10, "Basic Auth",
			16, "Success", strings.Repeat("-", maxWidth), strings.Repeat("-", 10),
			strings.Repeat("-", 16), strings.Repeat("-", 80-maxWidth-2))
	}
	for _, site := range sites {
		if verboseMode {
			log.Printf("Checking site %s\n", site.Base)
		}
		// channel for synchronizing 'done state', buffer the amount of endpoints
		done := make(chan bool, len(site.Endpoints))
		for endpoint, baShouldBe := range site.Endpoints {
			go checkEndpoint(done, &site, endpoint, baShouldBe, maxWidth)
		}
		// Drain the channel and wait for all goroutines to complete
		for i := 0; i < len(site.Endpoints); i++ {
			<-done // wait for one task to complete
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

func checkEndpoint(done chan bool, site *site, endpoint string, baShouldBe bool, maxWidth int) error {
	response, err := getEndpoint(site.Base, endpoint, baShouldBe)
	if err != nil {
		return err
	}
	success, baEnabled := checkSuccess(response, baShouldBe)
	var message string
	var logMessage string

	if success {
		message = fmt.Sprintf("%*s | %*s | %*t | %s\n", maxWidth, fmt.Sprintf("%s/%s", site.Base, endpoint), 10, "yes", 16, success, response.Status)
		logMessage = fmt.Sprintf("OK: %s/%s correct. Basic Auth Enabled: %t Should be: %t\n", site.Base, endpoint, baEnabled, baShouldBe)
	} else {
		if response.StatusCode > 401 {
			message = fmt.Sprintf("%*s | %*s | %*t | %s\n", maxWidth, fmt.Sprintf("%s/%s", site.Base, endpoint), 10, "unknown", 16, success, response.Status)
			logMessage = fmt.Sprintf("ERROR: %s/%s unknown. %s\n", site.Base, endpoint, response.Status)
		} else {
			message = fmt.Sprintf("%*s | %*s | %*t | %s\n", maxWidth, fmt.Sprintf("%s/%s", site.Base, endpoint), 10, "no", 16, success, response.Status)
			logMessage = fmt.Sprintf("ERROR: %s/%s incorrect. Basic Auth Enabled: %t Should be: %t\n", site.Base, endpoint, baEnabled, baShouldBe)
		}
	}

	if verboseMode {
		log.Printf(logMessage)
	} else {
		fmt.Printf(message)
	}
	done <- true
	return nil
}

func getEndpoint(site string, endpoint string, baShouldBe bool) (*http.Response, error) {
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
