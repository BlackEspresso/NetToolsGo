// bfcrawler.go
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"reflect"
	"strings"
)

func checkerror(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	hostString := flag.String("host", "", "host, e.g. google.com")
	emailString := flag.String("email", "", "email e.g. m@googlemail.com")
	flag.Parse()

	if *hostString == "" {
		log.Fatal("no --host flag provided.")
	}

	files, err := ioutil.ReadDir("./")
	checkerror(err)

	lines := make(map[string]bool)

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".wl") {
			continue
		}
		cfile, err := ioutil.ReadFile(file.Name())
		checkerror(err)
		content := string(cfile)
		for _, l := range strings.Split(content, "\n") {
			l = strings.TrimSpace(l)
			lines[l] = false
		}
	}

	if len(lines) == 0 {
		log.Fatal("no subdomains found in wordlists *.wl")
	}

	// lookup domains + aliases
	subdomains := LookUpIPs(GetKeys(lines), *hostString)
	aliases := LookUpAliases(GetKeys(subdomains), *hostString)

	// filter aliases for new domains
	domainsInSearch := FilterDomains(GetValues(aliases), *hostString)
	subdomainsInSearch := GetSubDomains(domainsInSearch, *hostString)
	newSubDomains := FindNewItems(GetKeys(subdomains), subdomainsInSearch)

	// lookup new domains
	subdomains2 := LookUpIPs(newSubDomains, *hostString)

	// append newdomains
	for k, v := range subdomains2 {
		subdomains[k] = v
	}

	msg := FormatSubdomains(subdomains, aliases, *hostString)

	fmt.Printf("found %d subdomains\n", len(subdomains))
	fmt.Printf(msg)

	if *emailString != "" {
		sendmail(*emailString, "goQueryDns - "+*hostString, msg)
	}
}

func FilterDomains(domains []string, filterDomain string) []string {
	founds := []string{}
	for _, domain := range domains {
		if strings.HasSuffix(domain, "."+filterDomain) {
			founds = append(founds, domain)
		} else if strings.HasSuffix(domain, "."+filterDomain+".") {
			founds = append(founds, domain)
		}
	}
	return founds
}

func GetSubDomains(domains []string, filterDomain string) []string {
	founds := []string{}
	for _, domain := range domains {
		splits := strings.Split(domain, "."+filterDomain)
		if len(splits) > 0 {
			founds = append(founds, splits[0])
		}
	}
	return founds
}

func GetValues(amap map[string][]string) []string {
	values := make([]string, 0, len(amap))
	for _, strs := range amap {
		for _, v := range strs {
			values = append(values, v)
		}

	}
	return values
}

func GetKeys(amap interface{}) []string {
	vt := reflect.ValueOf(amap)
	if vt.Kind() != reflect.Map {
		return []string{}
	}

	keys := make([]string, 0, vt.Len())
	for _, k := range vt.MapKeys() {
		keys = append(keys, k.String())
	}
	return keys
}

func Contains(list []string, searchValue string) bool {
	for _, v := range list {
		if v == searchValue {
			return true
		}
	}
	return false
}

func FindNewItems(list1 []string, list2 []string) []string {
	newItems := []string{}
	for _, v := range list2 {
		if !Contains(list1, v) {
			newItems = append(newItems, v)
		}
	}
	return newItems
}

func FormatSubdomains(subdomains map[string][]net.IP,
	aliases map[string][]string, host string) string {

	msg := ""
	for subdomain, ips := range subdomains {
		lookupName := GetLookUpName(subdomain, host)
		msg += fmt.Sprintln(lookupName)

		for n := range ips {
			line := fmt.Sprintf("\t%s\n", ips[n].String())
			msg += line
		}

		alias, ok := aliases[subdomain]
		if ok {
			for _, al := range alias {
				line := fmt.Sprintf("\t%s\n", al)
				msg += line
			}
		}
	}
	return msg
}

func LookUpAliases(subdomains []string, host string) map[string][]string {
	lookups := make(map[string][]string)

	for _, subdomain := range subdomains {
		lookupName := GetLookUpName(subdomain, host)
		alias := LookUpAlias(lookupName)
		if len(alias) > 0 {
			lookups[subdomain] = alias
		}

	}

	return lookups
}

func GetLookUpName(subdomain string, host string) string {
	return subdomain + "." + host
}

func sendmail(email string, subject string, messageString string) {
	cmd := exec.Command("goSendMail", "-email", email, "-subject", subject)
	cmd.Stdin = strings.NewReader(messageString)
	err := cmd.Run()
	if err != nil {
		fmt.Println(err.Error())
	}
}

func LookUpIPs(subdomains []string, host string) map[string][]net.IP {
	lookups := make(map[string][]net.IP)

	for _, subdomain := range subdomains {
		lookupName := GetLookUpName(subdomain, host)
		ips, err := net.LookupIP(lookupName)

		if err != nil || len(ips) == 0 {
			continue
		}

		lookups[subdomain] = ips
	}
	return lookups
}

func LookUpAlias(name string) []string {
	var rets = []string{}
	var maxResolveTries = 15

	for {
		maxResolveTries -= 1
		if maxResolveTries <= 0 {
			return rets
		}
		k, err := net.LookupCNAME(name)
		if err != nil {
			continue
		}
		if k == name {
			return rets
		}
		rets = append(rets, k)
		name = k

	}
	return rets
}
