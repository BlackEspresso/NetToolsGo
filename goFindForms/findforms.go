// findforms.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type HttpTransaction struct {
	Response     http.Response
	DoneTime     int64
	Duration     float64
	RequestBody  string
	ResponseBody string
	Tags         []string
}

func main() {
	files, err := ioutil.ReadDir("./sites")
	checkerr(err)

	links := make(map[string]bool)

	for _, f := range files {
		f, err := os.Open("./sites/" + f.Name())
		if err != nil {
			continue
		}
		defer f.Close()

		dec := json.NewDecoder(f)
		ht := HttpTransaction{}
		dec.Decode(&ht)

		buf := bytes.NewBufferString(ht.ResponseBody)
		doc, err := goquery.NewDocumentFromReader(buf)
		if err != nil {
			continue
		}
		findForms(ht.Response.Request.URL, doc, links)
	}

	for i, _ := range links {
		fmt.Println(i)
	}
}

func findForms(url *url.URL, doc *goquery.Document, links map[string]bool) {

	doc.Find("form").Each(func(i int, s *goquery.Selection) {
		attr, ok := s.Attr("action")
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

func checkerr(err error) {
	if err != nil {
		panic(err)
	}
}
