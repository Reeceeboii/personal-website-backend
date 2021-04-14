package email

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"time"
)

// Struct containing data required to send out emails
type Manager struct {
	// GMail SMTP domain
	GmailSMTPDomain string
	// GmailSMTPDomain + port
	GmailSMTPDomainFull string
	// Outbound Email address
	from string
	// Receiving Email address
	to []string
	// smtp.Auth instance
	auth smtp.Auth
}

// Used to send out an email when the main first starts up
func (emailManager *Manager) SendServerStartupEmail(
	serverBootTime time.Time,
	goRunTime string,
	AWSSDKName string,
	AWSSDKVersion string) {
	log.Println("Sending post-boot email")

	// locate and read in the template file
	t, err := template.ParseFiles("./pkg/email/templates/BootTemplate.html")
	if err != nil {
		log.Fatalf("Error parsing template file: %s", err.Error())
	}

	var body bytes.Buffer
	MIME := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("Subject: SERVER START %s \n%s\n\n",
		time.Now().Format(time.RFC822), MIME)))

	// format the required information for the template
	bootInformation := struct {
		ServerBootTime       string
		GoRuntime            string
		AWSSDKName           string
		AWSSDKVersion        string
		GitHubAPIRefreshRate string
		StatusEmailRate      string
	}{
		serverBootTime.Format(time.RFC822),
		goRunTime,
		AWSSDKName,
		AWSSDKVersion,
		fmt.Sprintf("%ss", os.Getenv("GITHUB_REFRESH_SECONDS")),
		fmt.Sprintf("%ss", os.Getenv("STATUS_EMAIL_REFRESH_SECONDS")),
	}

	// create the template with the given information
	err = t.Execute(&body, bootInformation)
	if err != nil {
		log.Fatalf("Error executing startup email template! %s", err.Error())
	}

	err = smtp.SendMail(
		emailManager.GmailSMTPDomainFull,
		emailManager.auth,
		emailManager.from,
		emailManager.to, body.Bytes())
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Println("Boot email sent!")
}

func NewEmailManager() *Manager {
	return &Manager{
		GmailSMTPDomain:     "smtp.gmail.com",
		GmailSMTPDomainFull: "smtp.gmail.com:587",
		from:                os.Getenv("OUTGOING_EMAIL"),
		to:                  []string{os.Getenv("TARGET_EMAIL")},
		auth: smtp.PlainAuth(
			"",
			os.Getenv("OUTGOING_EMAIL"),
			os.Getenv("GMAIL_APP_PASSWORD"),
			"smtp.gmail.com"),
	}
}
