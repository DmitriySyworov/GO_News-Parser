package letter

import (
	notconfig "app/notification/config"
	"context"
	"net/smtp"
	"shared/htmltemplates"
	"shared/loggers"
	"shared/shconstant"
	"shared/shprotos/event"
	"shared/storage"
	"time"

	"github.com/google/uuid"
	"github.com/jordan-wright/email"
	"github.com/sony/gobreaker"
	"google.golang.org/protobuf/proto"
)

type SendLetter struct {
	ApiEmail        string
	ApiPassword     string
	SmtpAddress     string
	SmtpAddressHost string
	Redis           *storage.Redis
	Logger          *loggers.Logger
	CircuitBreaker  *gobreaker.CircuitBreaker
}

type ISendLetter interface {
	SendEmailLetter(eventSend []byte) error
}
type MockSendLetter struct{}

func (ml *MockSendLetter) SendEmailLetter(eventSend []byte) error { return nil }

func NewSendLetter(conf *notconfig.SMTP, shRedis *storage.Redis, logger *loggers.Logger) ISendLetter {
	if conf.ApiEmail == "" || conf.ApiEmail == "example@gmail.com" {
		return &MockSendLetter{}
	}
	circuitBreaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "SMTP-breaker",
		MaxRequests: 3,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Warn("Circuit Breaker [" + name + "] changed condition: " + from.String() + " -> " + to.String())
		},
	})
	return &SendLetter{
		ApiEmail:        conf.ApiEmail,
		ApiPassword:     conf.ApiPassword,
		SmtpAddress:     conf.SmtpAddress,
		SmtpAddressHost: conf.SmtpAddressHost,
		Redis:           shRedis,
		Logger:          logger,
		CircuitBreaker:  circuitBreaker,
	}
}

type NotificationData struct {
	EmailTo []string
}

func (l *SendLetter) SendEmailLetter(eventSendData []byte) error {
	eventSend := &event.NotificationEvent{}

	if errUnmarshal := proto.Unmarshal(eventSendData, eventSend); errUnmarshal != nil {
		l.Logger.Error("failed to unmarshal event send: " + errUnmarshal.Error())
		return errUnmarshal
	}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), shconstant.CtxTimeoutRedis)
	defer cancel()

	eventUUID := eventSend.GetEventUUID()
	emailTo := eventSend.GetEmailTo()
	serviceName := shconstant.ServiceName
	domain := shconstant.Domain
	year := time.Now().Year()

	keyEvent := shconstant.EventKey + "letter:" + eventUUID
	success, errSetNX := l.Redis.SetNX(ctxTimeout, keyEvent, 1, shconstant.EventKafkaTTL).Result()
	if errSetNX != nil {
		l.Logger.Error("failed to set event send letter: " + errSetNX.Error())
		return errSetNX
	}
	if !success {
		l.Logger.Warn("duplicate event pass sent: " + eventUUID)
		return nil
	}

	var text, contentHTML []byte
	var subject string
	switch e := eventSend.Event.(type) {
	case *event.NotificationEvent_AuthCode:
		if e.AuthCode.GetValidTime() < time.Now().Unix() {
			l.Logger.Error("The session lifetime has expired, the event of sending a letter to the mail was missed")
			return nil
		}
		codeUser := e.AuthCode.GetCode()
		htmlMessage, errCreateMessage := htmltemplates.CreateHTMLMessageAuth(&htmltemplates.DataAuthLetter{
			SessionUUID: eventUUID,
			Email:       emailTo,
			Code:        codeUser,
			ServiceName: serviceName,
			Year:        year,
		})
		if errCreateMessage != nil {
			l.Logger.Error("failed to create auth message: " + errCreateMessage.Error())
			return errCreateMessage
		}
		subject = "Your verification code: " + codeUser
		text = []byte("Welcome to " + serviceName + "!\n\n" +
			"Your verification code: " + codeUser + "\n" +
			"Valid for: 5 minutes.\n\n" +
			"---\n" +
			"Session ID: " + eventUUID)
		contentHTML = htmlMessage
	case *event.NotificationEvent_NewDevice:
		device := e.NewDevice.GetDevice()
		clientIP := e.NewDevice.GetClientIp()
		currentTime := e.NewDevice.GetCurrentTime()
		htmlMessage, errCreateMessage := htmltemplates.CreateHTMLMessageNewDevice(&htmltemplates.DataSecurityNotification{
			AlertID:     eventUUID,
			Email:       emailTo,
			Device:      device,
			ClientIP:    clientIP,
			Timestamp:   currentTime,
			ServiceName: serviceName,
			Year:        year,
			Domain:      domain,
		})
		if errCreateMessage != nil {
			l.Logger.Error("failed to create new device message: " + errCreateMessage.Error())
			return errCreateMessage
		}
		subject = "📱 Security Notification: New Device Sign-In Detected"
		text = []byte("Welcome to " + serviceName + "!\n\n" +
			"We noticed a successful login to your account from a device we haven't seen before. Please review the details:\n\n" +
			"Account Email: " + emailTo + "\n" +
			"Device / Agent: " + device + "\n" +
			"IP Address: " + clientIP + "\n" +
			"Time (UTC): " + currentTime + "\n\n" +
			"If this was you, no further action is needed. If you do not recognize this device, someone else may have accessed your account. Please log out of all devices immediately to secure your data:\n" +
			"http://" + domain + "/security/sessions\n\n" +
			"---\n" +
			"Notification ID: " + eventUUID)
		contentHTML = htmlMessage
	case *event.NotificationEvent_SecurityAlert:
		device := e.SecurityAlert.GetDevice()
		clientIP := e.SecurityAlert.GetClientIp()
		currentTime := e.SecurityAlert.GetCurrentTime()
		htmlMessage, errCreateMessage := htmltemplates.CreateHTMLMessageSecurityAlert(&htmltemplates.DataSecurityNotification{
			AlertID:     eventUUID,
			Email:       emailTo,
			Device:      device,
			ClientIP:    clientIP,
			Timestamp:   currentTime,
			ServiceName: serviceName,
			Year:        year,
			Domain:      domain,
		})
		if errCreateMessage != nil {
			l.Logger.Error("failed to create new security alert message: " + errCreateMessage.Error())
			return errCreateMessage
		}
		subject = "📱 Security Notification: New Device Sign-In Detected"
		text = []byte("Welcome to " + serviceName + "!\n\n" +
			"We noticed a successful login to your account from a device we haven't seen before. Please review the details:\n\n" +
			"Account Email: " + emailTo + "\n" +
			"Device / Agent: " + device + "\n" +
			"IP Address: " + clientIP + "\n" +
			"Time (UTC): " + currentTime + "\n\n" +
			"If this was you, no further action is needed. If you do not recognize this device, someone else may have accessed your account. Please log out of all devices immediately to secure your data:\n" +
			"http://" + domain + "/security/sessions\n\n" +
			"---\n" +
			"Notification ID: " + eventUUID)
		contentHTML = htmlMessage
	}
	send := email.NewEmail()
	_, errSendLetter := l.CircuitBreaker.Execute(func() (interface{}, error) {
		send.From = l.ApiEmail
		send.To = []string{emailTo}
		send.Subject = subject
		send.Text = text
		send.HTML = contentHTML
		send.Headers.Set("Message-ID", "<"+uuid.New().String()+"@"+domain+".com>")
		send.Headers.Set("X-Priority", "1")
		send.Headers.Set("X-Mailer", "Go-Notification-Service-v1")
		return nil, send.Send(l.SmtpAddressHost, smtp.PlainAuth("", l.ApiEmail, l.ApiPassword, l.SmtpAddress))
	})
	if errSendLetter != nil {
		l.Logger.Error("failed to send letter to email: " + errSendLetter.Error())
		return errSendLetter
	}
	return nil
}
