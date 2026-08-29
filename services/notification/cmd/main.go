package main

import (
	notconfig "app/notification/config"
	"app/notification/internal/letter"
	"context"
	"errors"
	"html/template"
	"os"
	"shared/htmltemplates"
	"shared/loggers"
	"shared/shkafka"
	"shared/storage"
)

func init() {
	templateAuth, errParseAuth := template.New("auth").Parse(htmltemplates.HtmlTemplateAuthString)
	if errParseAuth != nil {
		panic(errors.New("failed to init template auth letter: " + errParseAuth.Error()))
	}
	htmltemplates.HtmlParseTemplateAuth = templateAuth
	templateNewDevice, errParseNewDevice := template.New("device").Parse(htmltemplates.HtmlNewDeviceTemplateString)
	if errParseNewDevice != nil {
		panic(errors.New("failed to init template new device letter: " + errParseNewDevice.Error()))
	}
	htmltemplates.HtmlParseTemplateNewDevice = templateNewDevice
	templateSecurityAlert, errParseSecurityAlert := template.New("auth").Parse(htmltemplates.HtmlSecurityAlertTemplateString)
	if errParseSecurityAlert != nil {
		panic(errors.New("failed to init template security alert letter: " + errParseSecurityAlert.Error()))
	}
	htmltemplates.HtmlParseTemplateSecurityAlert = templateSecurityAlert
}
func main() {
	logger := loggers.NewLogger()
	conf := notconfig.NewConfig(logger)
	consumer, errInitConsumer := shkafka.NewConsumer(&shkafka.ConfigConsumer{
		Brokers:       []string{conf.Broker},
		KafkaUser:     conf.KafkaUser,
		KafkaPassword: conf.KafkaPassword,
		Topic:         conf.NotificationTopic,
		GroupID:       conf.NotificationGroupID,
	}, logger)
	if errInitConsumer != nil {
		logger.Error("failed to init consumer kafka: " + errInitConsumer.Error())
		os.Exit(1)
	}
	defer consumer.CloseConsumer()
	sharedRedis := storage.OpenRedis(conf.SharedRedisAddress, conf.SharedRedisPassword, logger)
	defer func() {
		if errClose := sharedRedis.Close(); errClose != nil {
			logger.Error("failed to close shared redis: " + errClose.Error())
		}
	}()
	ctxCancel, cancel := context.WithCancel(context.Background())
	defer cancel()
	sender := letter.NewSendLetter(conf.SMTP, sharedRedis, logger)
	consumer.WaitEvent(ctxCancel, sender.SendEmailLetter)
}
