package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/jawher/mow.cli"
)

const toolVersion = "v0.7-pre"

var (
	anyLookUpfailed bool
)

type configuration struct {
	Sites []site
}

type site struct {
	Base            string
	Endpoints       map[string]bool
	EndpointsResult []*endpoint
}

type endpoint struct {
	BaShouldBe     bool
	Endpoint       string
	MaxWidth       int
	BaEnabled      bool
	Success        bool
	HTTPStatus     string
	HTTPStatusCode int
}

type endpointSorter []*endpoint

func (a endpointSorter) Len() int           { return len(a) }
func (a endpointSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a endpointSorter) Less(i, j int) bool { return a[i].Endpoint < a[j].Endpoint }

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

func numberOfTotalEndpoints(sites []site) (count int) {
	for _, site := range sites {
		count += len(site.Endpoints)
	}
	return count
}

func checkSites(sites []site, nospinner bool) {
	amountOfEndpoints := numberOfTotalEndpoints(sites)
	endpointChan := make(chan *endpoint, amountOfEndpoints)
	endpointDone := make(chan bool, amountOfEndpoints)
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
	for i := 0; i < amountOfEndpoints; i++ {
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
	sort.Sort(endpointSorter(site.EndpointsResult))
	for _, ep := range site.EndpointsResult {
		baMessage := "no"
		if ep.BaEnabled {
			baMessage = "yes"
		}
		baWantedMessage := "no"
		if ep.BaShouldBe {
			baWantedMessage = "yes"
		}
		if ep.Success {
			fmt.Printf("%*s | %*s | %*s | %*t | %s\n", maxWidth, ep.Endpoint, 10, baMessage, 10, baWantedMessage, 10, ep.Success, ep.HTTPStatus)
		} else {
			if ep.HTTPStatusCode > 401 {
				fmt.Printf("%*s | %*s | %*s | %*t | %s\n", maxWidth, ep.Endpoint, 10, baMessage, 10, baWantedMessage, 10, ep.Success, ep.HTTPStatus)
			} else {
				fmt.Printf("%*s | %*s | %*s | %*t | %s\n", maxWidth, ep.Endpoint, 10, baMessage, 10, baWantedMessage, 10, ep.Success, ep.HTTPStatus)
			}
			anyLookUpfailed = true
		}
	}
	fmt.Printf("%s-+-%s-+-%s-+-%s-+-%s\n", strings.Repeat("-", maxWidth), strings.Repeat("-", 10),
		strings.Repeat("-", 10), strings.Repeat("-", 10), strings.Repeat("-", 90-maxWidth-2))
}

func endpointWorker(endpointChan <-chan *endpoint, endpointDone chan bool) {
	for ep := range endpointChan {
		checkEndpoint(ep)
		endpointDone <- true
	}
}

func checkSite(site *site, endpointChan chan *endpoint) {
	for ep, baShouldBe := range site.Endpoints {
		epType := endpoint{
			BaShouldBe: baShouldBe,
			Endpoint:   fmt.Sprintf("%s/%s", site.Base, ep),
		}

		site.EndpointsResult = append(site.EndpointsResult, &epType)
		endpointChan <- &epType
	}
}

func checkSuccess(response *http.Response, baShouldBe bool) (success bool, baEnabled bool) {
	if response.StatusCode == 401 {
		baEnabled = true
	}
	return baEnabled == baShouldBe, baEnabled
}

func checkEndpoint(ep *endpoint) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", ep.Endpoint, nil)
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

func main() {
	app := cli.App("ba_checker", "Check HTTP Basic Auth status")
	app.Version("v version", toolVersion)
	app.Spec = "[--no-spinner] CONFIGFILE"

	var (
		noSpinner  = app.BoolOpt("n no-spinner", false, "Disable spinner animation")
		configFile = app.StringArg("CONFIGFILE", "", "Config file")
	)

	app.Action = func() {
		if _, err := os.Stat(*configFile); os.IsNotExist(err) {
			fmt.Printf("Error: Given config file %s does not exist, exiting..\n", *configFile)
			cli.Exit(1)
		}
		file, _ := os.Open(*configFile)
		decoder := json.NewDecoder(file)
		config := configuration{}
		err := decoder.Decode(&config)
		if err != nil {
			fmt.Println("error:", err)
		}
		checkSites(config.Sites, *noSpinner)
		if anyLookUpfailed {
			cli.Exit(1)
		}
	}

	app.Run(os.Args)
}
