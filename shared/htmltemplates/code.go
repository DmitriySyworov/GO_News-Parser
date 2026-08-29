package htmltemplates

import (
	"bytes"
	"html/template"
)

type DataAuthLetter struct {
	SessionUUID string
	Email       string
	Code        string
	ServiceName string
	Year        int
}

const HtmlTemplateAuthString = `
<!DOCTYPE html>
<html>
<head>
	<style>
		body { font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px; }
		.card { background: white; padding: 30px; border-radius: 8px; max-width: 500px; margin: 0 auto; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
		.code { font-size: 32px; font-weight: bold; color: #4F46E5; letter-spacing: 5px; text-align: center; margin: 20px 0; }
		.footer { font-size: 12px; color: #6B7280; text-align: center; margin-top: 20px; }
		.logo { text-align: center; margin-bottom: 20px; }
	</style>
</head>
<body>
	<div class="card">
		<div class="logo" style="font-size: 24px; font-weight: 800; color: #4F46E5; letter-spacing: 1px;">BUDGET PLANNER</div>
		<h2>Welcome to {{.ServiceName}}!</h2>
		<p>You requested a session verification code. Please enter it in the application:</p>
		<div class="code">{{.Code}}</div>
		<p>The code is valid for 5 minutes. If you did not request this code, please ignore this email.</p>
		<div class="footer">© {{.Year}} {{.ServiceName}}. All rights reserved. Session ID: {{.SessionUUID}}</div>
	</div>
</body>
</html>`

var HtmlParseTemplateAuth *template.Template

func CreateHTMLMessageAuth(dataLetter *DataAuthLetter) ([]byte, error) {
	var tmplBytes bytes.Buffer
	if errExecute := HtmlParseTemplateAuth.Execute(&tmplBytes, dataLetter); errExecute != nil {
		return nil, errExecute
	}
	return tmplBytes.Bytes(), nil
}
