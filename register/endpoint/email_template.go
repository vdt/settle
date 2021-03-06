package endpoint

import "text/template"

// EmailData is the data required to execute the email template.
type EmailData struct {
	Env      string
	From     string
	Username string
	Email    string
	Mint     string
	CredsURL string
	Secret   string
}

var emailTemplate *template.Template

func init() {
	emailTemplate = template.New("email")
	emailTemplate.Parse(
		"From: Mint Registration <{{.From}}>\r\n" +
			"To: {{.Email}}\r\n" +
			"Subject: Credentials for {{.Username}}@{{.Mint}}\r\n" +
			"Content-Type: text/plain; charset=UTF-8" +
			"\r\n" +
			"Hi {{.Username}}!\n" +
			"\n" +
			"Please click on the link below to retrieve your credentials for\n" +
			"{{.Mint}}[0]:\n" +
			"\n" +
			"{{.CredsURL}}#?env={{.Env}}&username={{.Username}}&secret={{.Secret}}\n" +
			"\n" +
			"Keep this link safe and secure as this is your only way to retrieve or\n" +
			"roll your credentials in the future.\n" +
			"\n" +
			"-settle\n" +
			"\n" +
			"[0] required to run `settle login`\n",
	)
}
