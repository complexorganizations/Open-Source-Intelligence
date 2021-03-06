package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	strip "github.com/grokify/html-strip-tags-go"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	errorMessage     = "message"
	errorStatusCode  = "status_code"
	errorResponseURL = "response_url"
	userAgent        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36"
	projectSherlock  = "configs/project-sherlock.json"
)

type socialNetwork struct {
	URL       string `json:"url,omitempty"`
	ErrorType string `json:"errorType,omitempty"`
	ErrorMsg  string `json:"errorMsg,omitempty"`
	ErrorURL  string `json:"errorUrl,omitempty"`
	NoPeriod  string `json:"noPeriod,omitempty"`
}

type socialNetworks map[string]*socialNetwork

var client = &http.Client{
	Timeout: time.Second * 30,
}

var username string

func init() {
	// Parse FLags
	tempUsername := flag.String("username", "", "checking services with given username")
	flag.Parse()
	username = *tempUsername
}

func successLine(name, message string) {
	fmt.Printf("\033[37;1m[\033[92;1m+\033[37;1m]\033[92;1m %s:\033[0m %s\n", name, message)
}

func errorLine(name, message string) {
	fmt.Printf("\033[37;1m[\033[91;1m-\033[37;1m]\033[92;1m %s:\033[93;1m %s\033[0m\n", name, message)
}

func isAvailable(s *socialNetwork, res *http.Response) bool {
	if s.ErrorType == errorMessage {
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			return true
		}
		if strings.Contains(strip.StripTags(string(bodyBytes)), s.ErrorMsg) {
			return true
		}
	} else if s.ErrorType == errorStatusCode {
		if res.StatusCode != 200 {
			return true
		}
	} else if s.ErrorType == errorResponseURL {
		if strings.Contains(res.Request.URL.String(), s.ErrorURL) {
			return true
		}
	}

	return false
}

func makeRequest(wg *sync.WaitGroup, username, name string, s *socialNetwork) {
	defer wg.Done()

	if s.NoPeriod == "True" && strings.Contains(username, ".") {
		errorLine(name, "User Name Not Allowed!")
		return
	}

	s.URL = strings.Replace(string(s.URL), "{}", username, 1)

	req, err := http.NewRequest("GET", s.URL, nil)
	if err != nil {
		errorLine(name, fmt.Sprintf("can't create request: %v", err))
		return
	}

	req.Header.Set("User-Agent", userAgent)

	res, err := client.Do(req)
	if err != nil {
		errorLine(name, fmt.Sprintf("request failed: %v", err))
		return
	}
	defer res.Body.Close()

	if isAvailable(s, res) {
		errorLine(name, fmt.Sprintf("Not Found (%s)", s.URL))
	} else {
		successLine(name, fmt.Sprintf("Found (%s)", res.Request.URL.String()))
	}
}

func sherlock(username string) {
	data, err := os.ReadFile(projectSherlock)
	// get all social networks
	socialNetworks := socialNetworks{}
	err = json.Unmarshal(data, &socialNetworks)
	if err != nil {
		errorLine("JSON", "Failed to parse JSON-Data")
		os.Exit(1)
	}

	// start checking ...
	var wg sync.WaitGroup
	wg.Add(len(socialNetworks))
	for name, socialNetwork := range socialNetworks {
		go makeRequest(&wg, username, name, socialNetwork)
	}
	wg.Wait()
}

func main() {
	if username == "" {
		// Read Username, if flags is empty
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\033[37;1mUsername:\033[0m ")
		username, _ = reader.ReadString('\n')
	}

	username = strings.ToLower(strings.Replace(strings.Trim(username, " \r\n"), " ", "", -1))

	sherlock(username)
}
