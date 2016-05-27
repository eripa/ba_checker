package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/briandowns/spinner"
	"github.com/jawher/mow.cli"
)

const toolVersion = "v0.7"

var (
	anyLookUpfailed bool
)

type configuration struct {
	Sites []site `toml:"site"`
}
type site struct {
	Base        string   `toml:"base"`
	BasicAuth   []string `toml:"auth"`
	NoBasicAuth []string `toml:"no_auth"`
	endpoints   []endpoint
}

type endpoint struct {
	BaShouldBe     bool
	URL            string
	BaEnabled      bool
	Success        bool
	HTTPStatus     string
	HTTPStatusCode int
}

type endpointSorter []endpoint

func (a endpointSorter) Len() int           { return len(a) }
func (a endpointSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a endpointSorter) Less(i, j int) bool { return a[i].URL < a[j].URL }

func getMaxWidth(sites []site) (width int) {
	var URL string
	for _, site := range sites {
		for _, ep := range site.endpoints {
			URL = fmt.Sprintf("%s/%s", site.Base, ep.URL)
			if len(URL) > width {
				width = len(URL)
			}
		}
	}
	return width
}

func numberOfTotalURLs(sites []site) (count int) {
	for _, site := range sites {
		count += len(site.endpoints)
	}
	return count
}

func checkSites(sites []site, nospinner bool) {
	amountOfURLs := numberOfTotalURLs(sites)
	endpointChan := make(chan *endpoint, amountOfURLs)
	endpointDone := make(chan bool, amountOfURLs)
	defer close(endpointChan)
	defer close(endpointDone)
	for i := 0; i < 30; i++ {
		go endpointWorker(endpointChan, endpointDone)
	}

	maxWidth := getMaxWidth(sites)
	s := spinner.New(spinner.CharSets[7], 100*time.Millisecond)
	if !nospinner {
		s.Prefix = "running tests"
		s.Start()
	}

	for i := range sites {
		checkSite(&sites[i], endpointChan)
	}
	// Wait for all endpoints to be done
	for i := 0; i < amountOfURLs; i++ {
		<-endpointDone // wait for one task to complete
	}
	if !nospinner {
		s.Stop()
	}
	for _, site := range sites {
		printResults(site, maxWidth)
	}
}

func printResults(site site, maxWidth int) {
	fmt.Printf("%*s | %*s | %*s | %*s | HTTP Status\n%s-+-%s-+-%s-+-%s-+-%s\n", maxWidth, "URL", 10, "Basic Auth",
		10, "Wanted BA", 10, "Success", strings.Repeat("-", maxWidth), strings.Repeat("-", 10),
		strings.Repeat("-", 10), strings.Repeat("-", 10), strings.Repeat("-", 90-maxWidth-2))
	sort.Sort(endpointSorter(site.endpoints))
	for _, ep := range site.endpoints {
		baMessage := "no"
		if ep.BaEnabled {
			baMessage = "yes"
		}
		baWantedMessage := "no"
		if ep.BaShouldBe {
			baWantedMessage = "yes"
		}
		if ep.Success {
			fmt.Printf("%*s | %*s | %*s | %*t | %s\n", maxWidth, ep.URL, 10, baMessage, 10, baWantedMessage, 10, ep.Success, ep.HTTPStatus)
		} else {
			if ep.HTTPStatusCode > 401 {
				fmt.Printf("%*s | %*s | %*s | %*t | %s\n", maxWidth, ep.URL, 10, baMessage, 10, baWantedMessage, 10, ep.Success, ep.HTTPStatus)
			} else {
				fmt.Printf("%*s | %*s | %*s | %*t | %s\n", maxWidth, ep.URL, 10, baMessage, 10, baWantedMessage, 10, ep.Success, ep.HTTPStatus)
			}
			anyLookUpfailed = true
		}
	}
	fmt.Printf("%s-+-%s-+-%s-+-%s-+-%s\n", strings.Repeat("-", maxWidth), strings.Repeat("-", 10),
		strings.Repeat("-", 10), strings.Repeat("-", 10), strings.Repeat("-", 90-maxWidth-2))
}

func endpointWorker(endpointChan <-chan *endpoint, endpointDone chan bool) {
	for ep := range endpointChan {
		checkURL(ep)
		endpointDone <- true
	}
}

func checkSite(site *site, endpointChan chan *endpoint) {
	for index := range site.endpoints {
		endpointChan <- &site.endpoints[index]
	}
}

func checkSuccess(response *http.Response, baShouldBe bool) (success bool, baEnabled bool) {
	if response.StatusCode == 401 {
		baEnabled = true
	}
	return baEnabled == baShouldBe, baEnabled
}

func checkURL(ep *endpoint) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", ep.URL, nil)
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", fmt.Sprintf("ba_checker %s", toolVersion))
	response, err := client.Do(req)

	if err != nil {
		ep.Success = false
		ep.BaEnabled = false
	}
	ep.HTTPStatusCode = response.StatusCode
	ep.HTTPStatus = response.Status
	ep.Success, ep.BaEnabled = checkSuccess(response, ep.BaShouldBe)
}

func populateURLConfig(config *configuration) {
	for index := range config.Sites {
		for _, baURL := range config.Sites[index].BasicAuth {
			config.Sites[index].endpoints = append(config.Sites[index].endpoints,
				endpoint{
					BaShouldBe: true,
					URL:        fmt.Sprintf("%s/%s", config.Sites[index].Base, baURL),
				})
		}
		for _, URL := range config.Sites[index].NoBasicAuth {
			config.Sites[index].endpoints = append(config.Sites[index].endpoints,
				endpoint{
					BaShouldBe: false,
					URL:        fmt.Sprintf("%s/%s", config.Sites[index].Base, URL),
				})
		}
	}
}

func main() {
	app := cli.App("ba_checker", "Check HTTP Basic Auth status")
	app.Version("v version", toolVersion)
	app.Spec = "[--no-spinner] CONFIGFILE"

	var (
		noSpinner  = app.BoolOpt("n no-spinner", false, "Disable spinner animation")
		configFile = app.StringArg("CONFIGFILE", "", "Config file")
	)

	app.Action = func() {
		var config configuration
		if _, err := os.Stat(*configFile); os.IsNotExist(err) {
			fmt.Printf("Error: Given config file %s does not exist, exiting..\n", *configFile)
			cli.Exit(1)
		}
		if _, err := toml.DecodeFile(*configFile, &config); err != nil {
			fmt.Println("Error:", err)
			cli.Exit(1)
		}
		populateURLConfig(&config)
		checkSites(config.Sites, *noSpinner)
		if anyLookUpfailed {
			cli.Exit(1)
		}
	}

	app.Run(os.Args)
}
