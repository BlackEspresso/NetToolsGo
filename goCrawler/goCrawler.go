// gocrawler
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type HttpTransaction struct {
	Response     http.Response
	DoneTime     int64
	Duration     float64
	RequestBody  string
	ResponseBody string
	Tags         []string
}

func NewFromResponse(res *http.Response, reqBody io.Reader, donetime int64, duration float64) HttpTransaction {
	ht := HttpTransaction{}
	ht.Response = *res

	body, err := ioutil.ReadAll(res.Body)
	if err == nil {
		ht.ResponseBody = string(body)
	}

	if reqBody != nil {
		body, err = ioutil.ReadAll(reqBody)
		if err == nil {
			ht.RequestBody = string(body)
		}
	}

	ht.DoneTime = donetime
	ht.Duration = duration

	return ht
}

func main() {
	urlFlag := flag.String("url", "", "url, e.g. http://www.google.com")
	delayFlag := flag.Int("delay", 1000, "delay, in milliseconds, default is 1000ms=1sec")
	flag.Parse()

	if *urlFlag == "" {
		log.Fatal("no url provided.")
	}

	_, err := url.Parse(*urlFlag)
	checkerror(err)

	links := make(map[string]bool)
	links[*urlFlag] = false // startsite
	fetchSites(links, *delayFlag)
}

func DoGETTransaction(url string) (*HttpTransaction, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "text/html")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/42.0.2311.135 Safari/537.36")

	starttime := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	durSec := time.Now().Sub(starttime).Seconds()
	nowSec := time.Now().Unix()
	ht := NewFromResponse(resp, nil, nowSec, durSec)
	return &ht, nil
}

func fetchSites(links map[string]bool, delayMs int) {
	for {
		urlStr, found := getNextSite(links)
		if !found {
			return // done
		}

		links[urlStr] = true
		fmt.Println("parsing site: " + urlStr)

		ht, err := DoGETTransaction(urlStr)
		if err != nil {
			fmt.Printf("skipping: " + err.Error())
			continue
		}
		if ok, _ := PathExists("./sites/"); !ok {
			os.Mkdir("./sites", 0777)
		}

		f, err := os.Create("./sites/" + strconv.FormatInt(ht.DoneTime, 10) + ".httpt")
		checkerror(err)
		defer f.Close()

		content, err := json.Marshal(ht)
		checkerror(err)
		f.Write(content)

		// fetch
		buf := bytes.NewBufferString(ht.ResponseBody)
		doc, err := goquery.NewDocumentFromReader(buf)
		checkerror(err)
		// parse content
		findLinks(urlStr, doc, links)
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func getNextSite(links map[string]bool) (string, bool) {
	for i, l := range links {
		if l == false {
			return i, true
		}
	}
	return "", false
}

func findLinks(urlStr string, doc *goquery.Document, links map[string]bool) {
	url, err := url.Parse(urlStr)
	checkerror(err)

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		attr, ok := s.Attr("href")
		if !ok {
			return
		}
		refurl, err := url.Parse(attr)
		if err != nil {
			return
		}
		absurl := url.ResolveReference(refurl)

		if !strings.Contains(absurl.Host, url.Host) {
			return
		}

		_, ok = links[absurl.String()]
		if ok {
			return // already exists => return
		}

		links[absurl.String()] = false
	})
}

func checkerror(e error) {
	if e != nil {
		panic(e)
	}
}
