package htmltemplates

import (
	"bytes"
	"html/template"
)

type DataSecurityNotification struct {
	AlertID     string
	Email       string
	Device      string
	ClientIP    string
	Timestamp   string
	ServiceName string
	Domain      string
	Year        int
}

const HtmlNewDeviceTemplateString = `
<!DOCTYPE html>
<html>
<head>
	<style>
		body { font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px; }
		.card { background: white; padding: 30px; border-radius: 8px; max-width: 500px; margin: 0 auto; box-shadow: 0 4px 6px rgba(0,0,0,0.1); border-top: 4px solid #3B82F6; }
		.alert-title { color: #1E3A8A; font-size: 20px; font-weight: bold; margin-top: 10px; }
		.info-table { width: 100%; border-collapse: collapse; margin: 20px 0; font-size: 14px; }
		.info-table td { padding: 10px; border-bottom: 1px solid #E5E7EB; }
		.info-table td.label { color: #6B7280; width: 35%; }
		.info-table td.value { color: #111827; font-weight: 500; }
		.btn-box { text-align: center; margin: 25px 0 15px 0; }
		.btn { background-color: #3B82F6; color: white !important; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block; box-shadow: 0 2px 4px rgba(59,130,246,0.2); }
		.footer { font-size: 12px; color: #6B7280; text-align: center; margin-top: 25px; line-height: 1.5; }
		.logo { text-align: center; margin-bottom: 10px; }
	</style>
</head>
<body>
	<div class="card">
		<div class="logo" style="font-size: 22px; font-weight: 800; color: #1F2937; letter-spacing: 1px;">BUDGET <span style="color: #3B82F6;">PLANNER</span></div>
		
		<div class="alert-title">📱 New Device Sign-In Detected</div>
		
		<p>Hello! We noticed a successful login to your account from a device we haven't seen before. Please review the details below:</p>
		
		<table class="info-table">
			<tr>
				<td class="label">Account Email:</td>
				<td class="value">{{.Email}}</td>
			</tr>
			<tr>
				<td class="label">Device / Agent:</td>
				<td class="value">{{.Device}}</td>
			</tr>
			<tr>
				<td class="label">IP Address:</td>
				<td class="value" style="font-family: monospace; font-size: 14px;">{{.ClientIP}}</td>
			</tr>
			<tr>
				<td class="label">Time (UTC):</td>
				<td class="value">{{.Timestamp}}</td>
			</tr>
		</table>

		<p style="font-size: 14px; color: #4B5563;">If this was you, no further action is needed. <strong>If you do not recognize this device, someone else may have accessed your account. Log out of all devices immediately to secure your data.</strong></p>

		<div class="btn-box">
			<a href="http://{{.Domain}}/security/sessions" class="btn">Review My Sessions</a>
		</div>

		<div class="footer">
			© {{.Year}} {{.ServiceName}}. All rights reserved.<br>
			Notification ID: {{.AlertID}}
		</div>
	</div>
</body>
</html>`

var HtmlParseTemplateNewDevice *template.Template

func CreateHTMLMessageNewDevice(dataLetter *DataSecurityNotification) ([]byte, error) {
	var tmplBytes bytes.Buffer
	if errExecute := HtmlParseTemplateNewDevice.Execute(&tmplBytes, dataLetter); errExecute != nil {
		return nil, errExecute
	}
	return tmplBytes.Bytes(), nil
}

const HtmlSecurityAlertTemplateString = `
<!DOCTYPE html>
<html>
<head>
	<style>
		body { font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px; }
		.card { background: white; padding: 30px; border-radius: 8px; max-width: 500px; margin: 0 auto; box-shadow: 0 4px 6px rgba(0,0,0,0.1); border-top: 4px solid #DC2626; }
		.alert-title { color: #DC2626; font-size: 20px; font-weight: bold; margin-top: 10px; }
		.info-table { width: 100%; border-collapse: collapse; margin: 20px 0; font-size: 14px; }
		.info-table td { padding: 10px; border-bottom: 1px solid #E5E7EB; }
		.info-table td.label { color: #6B7280; width: 35%; }
		.info-table td.value { color: #111827; font-weight: 500; }
		.btn-box { text-align: center; margin: 25px 0 15px 0; }
		.btn { background-color: #DC2626; color: white !important; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block; box-shadow: 0 2px 4px rgba(220,38,38,0.2); }
		.footer { font-size: 12px; color: #6B7280; text-align: center; margin-top: 25px; line-height: 1.5; }
		.logo { text-align: center; margin-bottom: 10px; }
	</style>
</head>
<body>
	<div class="card">
		<div class="logo" style="font-size: 22px; font-weight: 800; color: #1F2937; letter-spacing: 1px;">BUDGET <span style="color: #DC2626;">PLANNER</span></div>
		
		<div class="alert-title">⚠️ CRITICAL: All Sessions Terminated</div>
		
		<p>Hello! We detected a highly suspicious attempt to access your account using a valid token, but from an unrecognized device and network location simultaneously.</p>
		
		<p style="color: #DC2626; font-weight: bold;">To protect your funds and personal budget data, we have instantly terminated all active sessions and locked your account.</p>
		
		<table class="info-table">
			<tr>
				<td class="label">Blocked Location:</td>
				<td class="value" style="color: #DC2626;">Suspicious Remote Request</td>
			</tr>
			<tr>
				<td class="label">Attacker Device:</td>
				<td class="value">{{.Device}}</td>
			</tr>
			<tr>
				<td class="label">Attacker IP:</td>
				<td class="value" style="font-family: monospace; font-size: 14px;">{{.ClientIP}}</td>
			</tr>
			<tr>
				<td class="label">Action Taken:</td>
				<td class="value" style="color: #DC2626; font-weight: bold;">Full Account Lockout</td>
			</tr>
		</table>

		<p style="font-size: 14px; color: #4B5563;">You have been automatically logged out of all devices. To regain access, you must perform a secure password recovery and confirm your email identity.</p>

		<div class="btn-box">
			<a href="http://{{.Domain}}/security/recovery" class="btn">Recover My Account</a>
		</div>

		<div class="footer">
			© {{.Year}} {{.ServiceName}}. All rights reserved.<br>
			Security Incident ID: {{.AlertID}}
		</div>
	</div>
</body>
</html>`

var HtmlParseTemplateSecurityAlert *template.Template

func CreateHTMLMessageSecurityAlert(dataLetter *DataSecurityNotification) ([]byte, error) {
	var tmplBytes bytes.Buffer
	if errExecute := HtmlParseTemplateSecurityAlert.Execute(&tmplBytes, dataLetter); errExecute != nil {
		return nil, errExecute
	}
	return tmplBytes.Bytes(), nil
}
