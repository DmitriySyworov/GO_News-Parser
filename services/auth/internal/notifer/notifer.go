package notifer

import (
	"app/auth-service/internal/common"
	"context"
	"shared/loggers"
	"shared/outbox"
	"shared/shconstant"
	"shared/shkafka"
	"shared/shprotos/event"
	"time"

	"google.golang.org/protobuf/proto"
)

type Notifer struct {
	Logger            *loggers.Logger
	ProduceEmailEvent *shkafka.KafkaProducer
	Outbox            *outbox.Outbox
}

func NewNotifer(producer *shkafka.KafkaProducer, outbox *outbox.Outbox, logger *loggers.Logger) *Notifer {
	return &Notifer{
		Logger:            logger,
		Outbox:            outbox,
		ProduceEmailEvent: producer,
	}
}

type NotificationEvent struct {
	*LetterAuth
	*LetterSecurity
	EmailTo   string
	EventUUID string
}

type LetterAuth struct {
	ValidTime int64
	Code      string
}

type LetterSecurity struct {
	Device      string
	IP          string
	CurrentTime string
}
type NotificationAction string

const (
	SendCodeAction          NotificationAction = "code_action"
	SendNewDeviceAction     NotificationAction = "new_device_action"
	SendSecurityAlertAction NotificationAction = "security_alert_action"

	OutboxSecurityKey = "outbox_security_event"
)

func (n *Notifer) HelperSendEmailEvent(ntEvent *NotificationEvent, action NotificationAction) {
	var dataEvent []byte
	var errMarshal error
	switch action {
	case SendCodeAction:
		eventSendEmail := &event.NotificationEvent{
			Event: &event.NotificationEvent_AuthCode{
				AuthCode: &event.AuthLetterPayload{
					ValidTime: time.Now().Add(common.TTLSessionJWT).Unix(),
					Code:      ntEvent.Code,
				},
			},
			EmailTo:   ntEvent.EmailTo,
			EventUUID: ntEvent.EventUUID,
		}
		dataEvent, errMarshal = proto.Marshal(eventSendEmail)
		if errMarshal != nil {
			n.Logger.Error("failed to marshal proto event email send: " + errMarshal.Error())
			return
		}
		ctxTimeout, cancel := context.WithTimeout(context.Background(), shconstant.CtxTimeoutSendEventKafka)
		defer cancel()
		if errSendEvent := n.ProduceEmailEvent.SendInstantEvent(ctxTimeout, ntEvent.EventUUID, dataEvent); errSendEvent != nil {
			n.Logger.Error("failed to send event email letter: " + errSendEvent.Error())
		}
		return
	case SendNewDeviceAction:
		eventNewDevice := &event.NotificationEvent{
			Event: &event.NotificationEvent_NewDevice{
				NewDevice: &event.SecurityLetterPayload{
					Device:      ntEvent.LetterSecurity.Device,
					ClientIp:    ntEvent.LetterSecurity.IP,
					CurrentTime: time.Now().Format(time.DateTime),
				},
			},
			EmailTo:   ntEvent.EmailTo,
			EventUUID: ntEvent.EventUUID,
		}
		dataEvent, errMarshal = proto.Marshal(eventNewDevice)
		if errMarshal != nil {
			n.Logger.Error("failed to marshal proto event security alert letter: " + errMarshal.Error())
			return
		}
	case SendSecurityAlertAction:
		eventSecurityAlert := &event.NotificationEvent{
			Event: &event.NotificationEvent_SecurityAlert{
				SecurityAlert: &event.SecurityLetterPayload{
					Device:      ntEvent.LetterSecurity.Device,
					ClientIp:    ntEvent.LetterSecurity.IP,
					CurrentTime: time.Now().Format(time.DateTime),
				},
			},
			EmailTo:   ntEvent.EmailTo,
			EventUUID: ntEvent.EventUUID,
		}
		dataEvent, errMarshal = proto.Marshal(eventSecurityAlert)
		if errMarshal != nil {
			n.Logger.Error("failed to marshal proto event security alert letter: " + errMarshal.Error())
			return
		}
	}
	timeout := time.After(time.Minute * 120)
	waitRedisSend := time.Second * 20
	for {
		select {
		case <-timeout:
			n.Logger.Error("failed to send security event in Redis: stop by timeout")
			return
		default:
			if errAddEvent := n.Outbox.AddRedisOutbox(OutboxSecurityKey, dataEvent); errAddEvent != nil {
				n.Logger.Error("failed to send security event in Redis: " + errAddEvent.Error())
				time.Sleep(waitRedisSend)
				if waitRedisSend < time.Minute*30 {
					waitRedisSend += time.Second * 20
				}
				continue
			}
			return
		}
	}
}

func (n *Notifer) SenderSecurityLater(ctxCancel context.Context) {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
	for {
		select {
		case <-ctxCancel.Done():
			n.Logger.Info("graceful shutdown senderSecurityLater")
			return
		case <-ticker.C:
			securityEvents, errGetRemEvents := n.Outbox.RemRangeRedisOutbox(OutboxSecurityKey)
			if errGetRemEvents != nil {
				n.Logger.Error("failed to get-rem outbox security events: " + errGetRemEvents.Error())
				continue
			}
			if len(securityEvents) == 0 {
				continue
			}
			waitKafkaSend := time.Second * 20
			sliceDataEvents := make([][]byte, 0, len(securityEvents))
			for _, e := range securityEvents {
				sliceDataEvents = append(sliceDataEvents, []byte(e))
			}
			for {
				ctxTimeout, cancel := context.WithTimeout(context.Background(), shconstant.CtxTimeoutSendEventKafka)
				if errSendEvent := n.ProduceEmailEvent.SendBatchEvent(ctxTimeout, sliceDataEvents); errSendEvent != nil {
					cancel()
					n.Logger.Error("failed to send batch security event email letter: " + errSendEvent.Error())
					time.Sleep(waitKafkaSend)
					if waitKafkaSend < time.Minute*30 {
						waitKafkaSend += time.Second * 20
					}
					continue
				}
				cancel()
				break
			}

		}

	}
}
