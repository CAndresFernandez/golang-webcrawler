package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var userAgents = []string {
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36 Edg/119.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.3",
}

// function to select a random user agent from the list to avoid getting blacklisted or overloading the site with requests from one user agent
func randomUserAgent() string {
	rand.New(rand.NewSource(time.Now().Unix()))
	randNum := rand.Int() % len(userAgents)
	return userAgents[randNum]
}

// function that will take the http response and convert it to a readable document
func discoverLinks(response *http.Response, baseURL string) []string {
	if response != nil {
		// set a variable containing the readable doc from the response body
		doc, _ := goquery.NewDocumentFromReader(response.Body)
		// build a slice of strings for the urls we find in doc
		foundUrls := []string{}
		if doc != nil {
			// find all links in the html of the doc, then select the href and append it to foundUrls
			doc.Find("a").Each(func(i int, s *goquery.Selection) {
				res, _ := s.Attr("href")
				foundUrls = append(foundUrls, res)
			})
		}
		// return the slice of urls
		return foundUrls

	} else {
		// return an empty string
		return []string{}
	}

	}


// function to make http requests
func getRequest(targetURL string)(*http.Response, error) {
	client := &http.Client{}

	req,_ := http.NewRequest("GET", targetURL, nil)
	// set the user-agent header as a random user agent
	req.Header.Set("User-Agent", randomUserAgent())

	res, err := client.Do(req)
		if err != nil {
			return nil, err
		} else {
			return res, nil
		}
}

// function to check the url and reformat it if necessary with the baseUrl and href
func checkRelative(href string, baseUrl string) string {
	if strings.HasPrefix(href, "/") {
		// return a formatted string
		return fmt.Sprintf("%s%s", baseUrl, href)
	} else {
		return href
	}
}

// function to check the format of urls and parse them, then ensure that they're from the right host
func resolveRelativeLinks(href string, baseUrl string)(bool, string) {
	resultHref := checkRelative(href, baseUrl)
	baseParse, _ := url.Parse(baseUrl)
	resultParse, _ := url.Parse(resultHref)
	if baseParse != nil && resultParse != nil {
		if baseParse.Host == resultParse.Host {
			return true, resultHref
		} else {
			return false, ""
		}
	}
	return false, ""
}

// build a channel of tokens for concurrency, set request limit at 5
var tokens = make(chan struct{}, 5)

// function to control concurrency and crawl pages, taking a targetURL and the baseURL
func Crawl(targetURL string, baseURL string) []string {
	fmt.Println(targetURL)
	// take out a token for the process by sending an empty struct
	tokens <- struct{}{}
	// variable resp contains body of the response received from of getRequest()
	resp, _ := getRequest(targetURL)
	// return the token after the request is made
	<-tokens
	// variable links contains the links which are the result of running the request response through discoverLinks()
	links := discoverLinks(resp, baseURL)
	// build a slice of strings for our links
	foundUrls := []string{}

	// range over the links
	for _,link := range links {
		ok, correctLink := resolveRelativeLinks(link, baseURL)
		if ok {
			if correctLink != "" {
				foundUrls = append(foundUrls, correctLink)
			}
		}
	}
	// ParseHTML(resp) // see below
	return foundUrls
}

// fill this if you want to parse the html from the website
// func ParseHTML(response *http.Response) {
// }

func main() {
	// build a channel that contains a slice of strings
	worklist := make(chan []string)
	// to make sure the loop runs properly, set a variable n and increment it to 1 immediately
	var n int
	n++
	baseDomain := "https://www.theguardian.com/europe"
	// this goroutine starts our worklist by sending the first input to the list: baseDomain
	go func(){worklist <- []string{"https://www.theguardian.com/europe"}}()

	// create our seen map which contains a string (url) and a boolean
	seen := make(map[string]bool)

	for; n>0; n--{
	// set a variable list which contains the worklist so we can range over it
	list := <- worklist
	for _, link := range list {
		// if a link hasn't been seen, set to true, increment, and crawl the page
		if !seen[link] {
			seen[link] = true
			n++
			// pass the url string and the baseDomain string in a goroutine and which will crawl the page
			go func(link string, baseURL string){
				// set a variable which contains the result of the crawl
				foundLinks := Crawl(link, baseDomain)
				// if there are foundLinks after the crawl, add them to the worklist to be crawled
				if foundLinks != nil {
					worklist <- foundLinks
				}
			}(link, baseDomain)
		}
	}
	}
}