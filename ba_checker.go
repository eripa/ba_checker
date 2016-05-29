package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/briandowns/spinner"
	"github.com/jawher/mow.cli"
	"github.com/olekukonko/tablewriter"
)

const (
	toolVersion = "v0.8-pre"
)

var (
	lookUpStatusCodeMap = map[int]string{
		0: "OK",
		1: "WARNING",
		2: "CRITICAL",
		3: "UNKNOWN",
	}
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
	Unknown        bool
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

func checkSites(sites []site) {
	amountOfURLs := numberOfTotalURLs(sites)
	endpointChan := make(chan *endpoint, amountOfURLs)
	endpointDone := make(chan bool, amountOfURLs)
	defer close(endpointChan)
	defer close(endpointDone)
	for i := 0; i < 30; i++ {
		go endpointWorker(endpointChan, endpointDone)
	}

	for i := range sites {
		checkSite(&sites[i], endpointChan)
	}
	// Wait for all endpoints to be done
	for i := 0; i < amountOfURLs; i++ {
		<-endpointDone // wait for one task to complete
	}

}

func printSitesTable(sites []site, warningThreshold int, criticalThreshold int) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoFormatHeaders(false)
	table.SetHeader([]string{"URL", "Basic Auth", "Wanted BA", "Success", "HTTP Status"})
	for _, site := range sites {
		sort.Sort(endpointSorter(site.endpoints))
		for _, ep := range site.endpoints {
			baMessage := "no"
			baWantedMessage := "no"
			if ep.BaEnabled {
				baMessage = "yes"
			}
			if ep.Unknown {
				baMessage = "unknown"
			}
			if ep.BaShouldBe {
				baWantedMessage = "yes"
			}
			data := []string{
				ep.URL,
				baMessage,
				baWantedMessage,
				strconv.FormatBool(ep.Success),
				ep.HTTPStatus,
			}
			table.Append(data)
		}
	}
	table.Render()
	fmt.Printf("\nStatus: %s\n", lookUpStatusCodeMap[checkStatus(sites, warningThreshold, criticalThreshold)])
}

func printResults(sites []site, warningThreshold int, criticalThreshold int) {
	printSitesTable(sites, warningThreshold, criticalThreshold)
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

func checkSuccess(response *http.Response, baShouldBe bool) (success bool, baEnabled bool, unknown bool) {
	if response.StatusCode == 401 {
		baEnabled = true
	} else if response.StatusCode > 401 {
		unknown = true
	}
	return baEnabled == baShouldBe, baEnabled, unknown
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
	ep.Success, ep.BaEnabled, ep.Unknown = checkSuccess(response, ep.BaShouldBe)
}

func populateURLConfig(sites []site) {
	for index := range sites {
		for _, baURL := range sites[index].BasicAuth {
			sites[index].endpoints = append(sites[index].endpoints,
				endpoint{
					BaShouldBe: true,
					URL:        fmt.Sprintf("%s/%s", sites[index].Base, baURL),
				})
		}
		for _, URL := range sites[index].NoBasicAuth {
			sites[index].endpoints = append(sites[index].endpoints,
				endpoint{
					BaShouldBe: false,
					URL:        fmt.Sprintf("%s/%s", sites[index].Base, URL),
				})
		}
	}
}

func checkStatus(sites []site, warningThreshold int, criticalThreshold int) (status int) {
	failures := 0
	unknowns := 0
	for _, site := range sites {
		for _, ep := range site.endpoints {
			if !ep.Success {
				failures++
			}
			if ep.Unknown {
				unknowns++
			}
		}
	}
	switch {
	case failures >= criticalThreshold:
		return 2
	case unknowns > 0:
		return 3
	case failures >= warningThreshold:
		return 1
	}
	return
}

func main() {
	app := cli.App("ba_checker", `Check HTTP Basic Auth status

Status can be determined by Exit codes:
 0=Status OK
 1=Above warning threshold
 2=Above critical threshold
 3=Unknown Basic Auth status (4xx or 5xx HTTP codes)`)
	app.Version("v version", toolVersion)
	app.Spec = "[--warning=<number>] [--critical=<number>] [--no-spinner] CONFIGFILE"

	var (
		noSpinner  = app.BoolOpt("no-spinner", false, "Disable spinner animation")
		configFile = app.StringArg("CONFIGFILE", "", "Config file")
		// nagiosOutput      = app.BoolOpt("n nagios", false, "Show more simple Nagios style output")
		warningThreshold  = app.IntOpt("w warning", 1, "Warning threshold")
		criticalThreshold = app.IntOpt("c critical", 2, "Critical threshold")
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
		s := spinner.New(spinner.CharSets[7], 100*time.Millisecond)
		if !*noSpinner {
			s.Prefix = "running tests "
			s.Start()
		}
		populateURLConfig(config.Sites)
		checkSites(config.Sites)

		if !*noSpinner {
			s.Stop()
		}
		printResults(config.Sites, *warningThreshold, *criticalThreshold)
		lookupStatusCode := checkStatus(config.Sites, *warningThreshold, *criticalThreshold)
		if lookupStatusCode > 0 {
			cli.Exit(lookupStatusCode)
		}
	}

	app.Run(os.Args)
}
