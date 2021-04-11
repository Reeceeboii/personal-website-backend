package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"time"
)

// sharedEmailInformation - shared bits of data required to send emails
type sharedEmailInformation struct {
	GmailSMTPDomain     string
	GmailSMTPDomainFull string
	from                string
	to                  []string
	auth                smtp.Auth
}

var emailData = sharedEmailInformation{}

// InitEmailData - sets up information required for server to send outbound emails
func InitEmailData() {
	emailData.GmailSMTPDomain = "smtp.gmail.com"
	emailData.GmailSMTPDomainFull = "smtp.gmail.com:587"
	emailData.from = os.Getenv("OUTGOING_EMAIL")
	emailData.to = []string{os.Getenv("TARGET_EMAIL")}
	emailData.auth = smtp.PlainAuth(
		"",
		os.Getenv("OUTGOING_EMAIL"),
		os.Getenv("GMAIL_APP_PASSWORD"),
		"smtp.gmail.com",
	)
}

// SendServerBootEmail - send some data about the server out when it first starts up
func SendServerBootEmail() {
	log.Println("Sending post-boot email")
	t, err := template.ParseFiles("./src/templates/BootTemplate.html")
	if err != nil {
		log.Fatal(err)
	}

	var body bytes.Buffer
	MIME := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("Subject: SERVER START %s \n%s\n\n", time.Now().Format(time.RFC822), MIME)))

	bootInformation := struct {
		ServerBootTime       string
		GoRuntime            string
		AWSSDKName           string
		AWSSDKVersion        string
		GitHubAPIRefreshRate string
		StatusEmailRate      string
	}{
		StaticInfo.ServerBootTime.Format(time.RFC822),
		StaticInfo.GoRuntime,
		StaticInfo.AWSSDKName,
		StaticInfo.AWSSDKVersion,
		fmt.Sprintf("%ss", os.Getenv("GITHUB_REFRESH_SECONDS")),
		fmt.Sprintf("%ss", os.Getenv("STATUS_EMAIL_REFRESH_SECONDS")),
	}

	t.Execute(&body, bootInformation)

	err = smtp.SendMail(
		emailData.GmailSMTPDomainFull,
		emailData.auth,
		emailData.from,
		emailData.to, body.Bytes())
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Println("Boot email sent!")
}

func EmailJob() {
	tickRate, err := time.ParseDuration(fmt.Sprintf("%ss", os.Getenv("STATUS_EMAIL_REFRESH_SECONDS")))
	if err != nil {
		log.Fatal("Error parsing duration for email job: " + err.Error())
	}

	for range time.Tick(tickRate) {
		log.Println("Sending status email!")

		t, err := template.ParseFiles("./src/templates/StatusTemplate.html")
		if err != nil {
			log.Fatal(err)
		}

		var body bytes.Buffer
		MIME := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
		body.Write([]byte(fmt.Sprintf("Subject: Server update %s \n%s\n\n", time.Now().Format(time.RFC822), MIME)))

		information := struct {
			ServerUptime string
		}{
			time.Now().Sub(StaticInfo.ServerBootTime).String(),
		}

		t.Execute(&body, information)

		err = smtp.SendMail(
			emailData.GmailSMTPDomainFull,
			emailData.auth,
			emailData.from,
			emailData.to, body.Bytes())
		if err != nil {
			log.Fatal(err)
			return
		}
		log.Println("Status email sent!")
	}
}
