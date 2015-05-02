// bfcrawler.go
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
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

func checkerror(e error) {
	if e != nil {
		panic(e)
	}
}

// A data structure to hold a key/value pair.
type Pair struct {
	Key   string
	Value int
}

// A slice of Pairs that implements sort.Interface to sort by Value.
type PairList []Pair

func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value > p[j].Value }

// A function to turn a map into a PairList, then sort and return it.
func sortMapByValue(m map[string]int) PairList {
	p := make(PairList, len(m))
	i := 0
	for k, v := range m {
		p[i] = Pair{k, v}
		i = i + 1
	}
	sort.Sort(p)
	return p
}

func isUpperCase(char rune) bool {
	if char >= 65 && char <= 90 {
		return true
	}
	return false
}

func toValidSubdomainName(t string) string {
	newname := ""
	for i, char := range t {
		str := string(char)
		// numbers
		if char >= 48 && char <= 57 {
			newname += str
		}
		// letters
		if isUpperCase(char) {
			newname += str
		}
		if char >= 97 && char <= 122 {
			newname += str
		}
		if i != 0 && i != len(t)-1 && str == "-" {
			newname += str
		}
		if str == "_" {
			newname += str
		}
		if i != 0 && i != len(t)-1 && str == "." {
			newname += str
		}
	}
	return newname
}

func toValidFileName(t string) string {
	filename := strings.Replace(t, "https", "", -1)
	filename = strings.Replace(filename, "http", "_", -1)
	filename = strings.Replace(filename, ":", "", -1)
	filename = strings.Replace(filename, ".", "_", -1)
	filename = strings.Replace(filename, "/", "", -1)
	return filename
}

func containsString(text string, arr []string) bool {
	for _, v := range arr {
		if v == text {
			return true
		}
	}
	return false
}

func splitTextBy(text string, splits []string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		if containsString(string(r), splits) {
			return true
		}
		return false
	})
}

func splitWordByUpperCase(word string) []string {
	newWord := word
	for {
		found := false
		for i, k := range newWord {
			if i > 0 && isUpperCase(k) && newWord[i-1] != '\n' {
				newWord = strings.Replace(newWord, string(k), "\n"+string(k), -1)
				found = true
				break
			}
		}
		if found == false {
			break
		}
	}

	if newWord == "" {
		return []string{word}
	}
	return strings.Split(newWord, "\n")
}

func findWords(doc *goquery.Document, foundwords map[string]int) {
	text := doc.Find("*").Not("script, style, head, noscript").Text()

	words := splitTextBy(text,
		[]string{",", ".", "\n", "\t", " ", "\r", "?", "!", "(", ")", "{", "}"})

	for _, word := range words {
		word = strings.TrimSpace(word)
		word = toValidSubdomainName(word)
		splitted := splitWordByUpperCase(word)
		for _, spWord := range splitted {
			if spWord == "" {
				continue
			}
			if len(spWord) <= 1 {
				continue
			}
			if len(spWord) >= 25 {
				continue
			}
			i, ok := foundwords[spWord]
			if ok {
				foundwords[spWord] = i + 1
			} else {
				foundwords[spWord] = 1
			}
		}
	}
}

func readFiles(dir string, words map[string]int) {
	files, err := ioutil.ReadDir(dir)
	checkerror(err)

	for _, f := range files {
		f, err := os.Open(dir + "/" + f.Name())
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
		findWords(doc, words)
	}
}

func main() {
	dirStr := flag.String("dir", "", "directory")
	email := flag.String("email", "", "send mail if done")
	flag.Parse()

	if *dirStr == "" {
		log.Fatal("no directory provided.")
	}

	words := make(map[string]int)
	readFiles(*dirStr, words)

	list := sortMapByValue(words)

	msg := ""
	for _, v := range list {
		msg += v.Key + "\n"
	}

	fmt.Print(msg)

	if *email != "" {
		sendmail(*email, "grabwordlist - "+*dirStr, msg)
	}
}

func sendmail(email string, subject string, messageString string) {
	cmd := exec.Command("goSendMail", "-email", email, "-subject", subject)
	cmd.Stdin = strings.NewReader(messageString)
	err := cmd.Run()
	if err != nil {
		fmt.Println(err.Error())
	}
}
