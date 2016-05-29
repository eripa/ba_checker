package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testCase struct {
	sites      []site
	maxWidth   int
	totalCount int
}

func getTestSites() []site {
	sites := []site{
		site{
			Base:        "http://test.webdav.org",
			BasicAuth:   []string{"auth-basic"},
			NoBasicAuth: []string{"dav", ""},
		},
		site{
			Base:        "https://httpbin.org/",
			BasicAuth:   []string{"basic-auth/:user/:passwd"},
			NoBasicAuth: []string{"html", ""},
		},
	}
	return sites
}

func TestCheckSuccess(t *testing.T) {
	sites := getTestSites()
	for _, site := range sites {
		for _, ep := range site.endpoints {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if ep.BaShouldBe {
					w.WriteHeader(http.StatusUnauthorized)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			}))
			response, _ := http.Get(ts.URL)
			ep.Success, ep.BaEnabled, ep.Unknown = checkSuccess(response, ep.BaShouldBe)
			if !ep.Success {
				t.Error("No success! Expected success!")
			}
			if ep.BaEnabled != ep.BaShouldBe {
				t.Error("BA unexpected state!")
			}
			ts.Close()
		}
		for _, ep := range site.endpoints {
			response, _ := http.Get(ep.URL)
			ep.Success, ep.BaEnabled, ep.Unknown = checkSuccess(response, ep.BaShouldBe)
			if !ep.Success {
				t.Logf("Tested URL: %s Response BA: %t Expected BA: %t", ep.URL, response.StatusCode == 401, ep.BaShouldBe)
				t.Error("No success! Expected success!")
			}
			if ep.BaEnabled != ep.BaShouldBe {
				t.Error("BA unexpected state!")
			}
		}

	}
}

func TestCheckSuccessFail(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "testing yay")
	}))
	defer ts.Close()
	baShouldBe := true
	response, _ := http.Get(ts.URL)
	success, baEnabled, _ := checkSuccess(response, baShouldBe)
	if success {
		t.Error("Success?! Expected failure!")
	}
	if baEnabled != false {
		t.Error("BA enabled when it shouldn't!")
	}
}

func TestCheckURL(t *testing.T) {
	sites := getTestSites()
	for _, site := range sites {
		for index := range site.endpoints {
			checkURL(&site.endpoints[index])
			if !site.endpoints[index].Success {
				t.Errorf("Expected 'success' to be true, got false")
			}
		}
	}
}

func TestGetMaxWidth(t *testing.T) {
	tc := testCase{
		sites:    getTestSites(),
		maxWidth: 66,
	}
	populateURLConfig(tc.sites)
	got := getMaxWidth(tc.sites)
	if got != tc.maxWidth {
		t.Errorf("Incorrect maxWidth %d, wanted %d", got, tc.maxWidth)
	}
}

func TestNumberOfTotalURL(t *testing.T) {
	tc := testCase{
		sites:      getTestSites(),
		totalCount: 6,
	}
	populateURLConfig(tc.sites)
	got := numberOfTotalURLs(tc.sites)
	if got != tc.totalCount {
		t.Errorf("Incorrect total URL count %d, wanted %d", got, tc.totalCount)
	}
}
