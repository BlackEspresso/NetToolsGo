package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type JsonHttp struct {
	URL        string
	Body       string
	Header     http.Header
	Status     string
	StatusCode int
}

type HttpTransaction struct {
	Request  JsonHttp
	Response JsonHttp
	DoneTime int64
	Duration float64
}

var fileStorageUrl string = ""

func main() {
	urlFlag := flag.String("url", "", "url, e.g. http://www.google.com")
	fileStorage := flag.String("filestore", "http://localhost:8079/file/7363a35f-f411-4751-96ec-2d19b5a22323", "url to filestore")
	delayFlag := flag.Int("delay", 1000, "delay, in milliseconds, default is 1000ms=1sec")
	flag.Parse()

	fileStorageUrl = *fileStorage

	if *urlFlag == "" {
		log.Fatal("no url provided.")
	}

	_, err := url.Parse(*urlFlag)
	checkerror(err)

	links := make(map[string]bool)
	links[*urlFlag] = false // startsite
	fetchSites(links, *delayFlag)
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
		content, err := json.Marshal(ht)
		checkerror(err)

		fileName := strconv.FormatInt(ht.DoneTime, 10) + ".httpt"
		saveCrawl(urlStr, fileName, content)

		// fetch
		buf := bytes.NewBufferString(ht.Response.Body)
		doc, err := goquery.NewDocumentFromReader(buf)
		checkerror(err)
		// parse content
		findLinks(urlStr, doc, links)
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}
}

func TransactionFromResponse(res *http.Response, url string, duration float64) *HttpTransaction {
	ht := HttpTransaction{}

	body, err := ioutil.ReadAll(res.Body)
	if err == nil {
		ht.Response.Body = string(body)
	}
	ht.Request.Header = res.Request.Header
	ht.Request.URL = url

	if res.Request.Body != nil {
		body, err = ioutil.ReadAll(res.Request.Body)
		if err == nil {
			ht.Request.Body = string(body)
		}
	}

	ht.Response.Status = res.Status
	ht.Response.StatusCode = res.StatusCode
	ht.DoneTime = time.Now().Unix()
	ht.Duration = duration

	return &ht
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
	ht := TransactionFromResponse(resp, url, durSec)
	return ht, nil
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

func saveCrawl(crawledUri string, fileName string, content []byte) {
	params := map[string]string{"meta": crawledUri}

	req, err := newfileUploadRequest(fileStorageUrl, params, "upload", fileName, content)
	if err != nil {
		log.Fatal("cant create file store request ", err)
	}

	c := http.Client{}
	c.Timeout = time.Duration(200) * time.Second

	uploadSuccess := false
	for retries := 0; retries < 3; retries++ {
		cresp, err := c.Do(req)
		if err != nil {
			log.Println("file store", err)
			continue
		}
		if cresp.StatusCode != 200 {
			log.Println("file store response ", cresp.StatusCode)
			continue
		}
		uploadSuccess = true
		break
	}

	if !uploadSuccess {
		log.Println("error while saving")
		log.Println(fileName, len(content), fileStorageUrl)
		log.Fatal("exiting")
	}
}

func newfileUploadRequest(uri string, params map[string]string, paramName string, fName string,
	fileContent []byte) (*http.Request, error) {

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, fName)
	if err != nil {
		return nil, err
	}
	part.Write(fileContent)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("POST", uri, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err
}
