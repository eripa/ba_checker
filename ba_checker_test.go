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
			Base: "http://test.webdav.org",
			Endpoints: map[string]bool{
				"auth-basic": true,
				"dav":        false,
				"":           false,
			},
		},
		site{
			Base: "https://httpbin.org/",
			Endpoints: map[string]bool{
				"basic-auth/:user/:passwd ": true,
				"html": false,
				"":     false,
			},
		},
	}
	return sites
}

func TestCheckSuccess(t *testing.T) {
	sites := getTestSites()
	for _, site := range sites {
		for _, baShouldBe := range site.Endpoints {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if baShouldBe {
					w.WriteHeader(http.StatusUnauthorized)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			}))
			response, _ := http.Get(ts.URL)
			success, baEnabled := checkSuccess(response, baShouldBe)
			if !success {
				t.Error("No success! Expected success!")
			}
			if baEnabled != baShouldBe {
				t.Error("BA unexpected state!")
			}
			ts.Close()
		}
		for ep, baShouldBe := range site.Endpoints {
			URL := fmt.Sprintf("%s/%s", site.Base, ep)
			response, _ := http.Get(URL)
			success, baEnabled := checkSuccess(response, baShouldBe)
			if !success {
				t.Logf("Tested URL: %s Response BA: %t Expected BA: %t", URL, response.StatusCode == 401, baShouldBe)
				t.Error("No success! Expected success!")
			}
			if baEnabled != baShouldBe {
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
	success, baEnabled := checkSuccess(response, baShouldBe)
	if success {
		t.Error("Success?! Expected failure!")
	}
	if baEnabled != false {
		t.Error("BA enabled when it shouldn't!")
	}
}

func TestCheckEndpoint(t *testing.T) {
	sites := getTestSites()
	for _, site := range sites {
		for ep, baShouldBe := range site.Endpoints {
			epType := endpoint{
				BaShouldBe: baShouldBe,
				Endpoint:   fmt.Sprintf("%s/%s", site.Base, ep),
			}
			checkEndpoint(&epType)
			if !epType.Success {
				t.Errorf("Expected 'success' to be true, got false")
			}
		}
	}
}

func TestGetMaxWidth(t *testing.T) {
	tc := testCase{
		sites:    getTestSites(),
		maxWidth: 46,
	}
	got := getMaxWidth(tc.sites)
	if got != tc.maxWidth {
		t.Errorf("Incorrect maxWidth %d, wanted %d", got, tc.maxWidth)
	}
}

func TestNumberOfTotalEndpoints(t *testing.T) {
	tc := testCase{
		sites:      getTestSites(),
		maxWidth:   46,
		totalCount: 6,
	}
	got := numberOfTotalEndpoints(tc.sites)
	if got != tc.totalCount {
		t.Errorf("Incorrect total endpoint count %d, wanted %d", got, tc.totalCount)
	}
}
