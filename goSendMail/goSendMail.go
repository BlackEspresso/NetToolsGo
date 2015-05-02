// senmail.go
package main

import (
	"flag"
	"fmt"
	"gopkg.in/gomail.v1"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

type Configuration struct {
	Smtp struct {
		Username   string
		Password   string
		Servername string
		Port       int
	}
}

func main() {
	emailString := flag.String("email", "", "email e.g. m@googlemail.com")
	subjectString := flag.String("subject", "", "subject")
	verbose := flag.Bool("verbose", false, "more info what iam doing.")
	flag.Parse()

	if *emailString == "" {
		fmt.Println("no email found")
		return
	}

	config := Configuration{}

	configraw, err := ioutil.ReadFile("sendmail_config.yaml")
	checkerr(err)
	err = yaml.Unmarshal(configraw, &config)
	checkerr(err)

	bytes, err := ioutil.ReadAll(os.Stdin)
	checkerr(err)

	input := string(bytes)

	if *verbose {
		fmt.Println(config)
	}

	sendmail(*emailString, *subjectString, "message from <b>sendmail.go</b>", input, config)
}

func checkerr(err error) {
	if err != nil {
		panic(err)
	}
}

func sendmail(email string, subject string, messageString string, txtAttachment string, config Configuration) {
	msg := gomail.NewMessage()
	msg.SetHeader("From", config.Smtp.Username)
	msg.SetHeader("To", email)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", messageString)
	f := gomail.CreateFile("attached.txt", []byte(txtAttachment))
	msg.Attach(f)

	mailer := gomail.NewMailer(config.Smtp.Servername, config.Smtp.Username, config.Smtp.Password, config.Smtp.Port)
	err := mailer.Send(msg)
	checkerr(err)
}
