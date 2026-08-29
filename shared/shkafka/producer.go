package shkafka

import (
	"context"
	"net"
	"shared/loggers"
	"time"

	_ "crypto/sha256"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
)

type KafkaProducer struct {
	InstantProducer *kafka.Writer
	BatchProducer   *kafka.Writer
	Logger          *loggers.Logger
}
type ConfigProducer struct {
	Brokers       []string
	KafkaUser     string
	KafkaPassword string
	Topic         string
}

func NewProducer(conf *ConfigProducer, logger *loggers.Logger) (*KafkaProducer, error) {
	mechanism, errScramMechanism := scram.Mechanism(scram.SHA256, conf.KafkaUser, conf.KafkaPassword)
	if errScramMechanism != nil {
		return nil, errScramMechanism
	}

	netDialer := &net.Dialer{
		Timeout:   10 * time.Second,
		DualStack: false,
		Resolver: &net.Resolver{
			PreferGo: true,
		},
	}
	customTransport := &kafka.Transport{
		SASL: mechanism,
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return netDialer.DialContext(ctx, network, addr)
		},
	}

	instantProducer := &kafka.Writer{
		Addr:         kafka.TCP(conf.Brokers...),
		Topic:        conf.Topic,
		Balancer:     &kafka.Hash{},
		MaxAttempts:  5,
		Transport:    customTransport,
		RequiredAcks: kafka.RequireAll,
	}
	batchProducer := &kafka.Writer{
		Addr:         kafka.TCP(conf.Brokers...),
		Topic:        conf.Topic,
		Balancer:     &kafka.Hash{},
		MaxAttempts:  5,
		Transport:    customTransport,
		RequiredAcks: kafka.RequireAll,
		BatchSize:    1000,
		BatchTimeout: time.Millisecond * 20,
		BatchBytes:   5242880,
		Async:        false,
	}
	return &KafkaProducer{
		InstantProducer: instantProducer,
		BatchProducer:   batchProducer,
		Logger:          logger,
	}, nil
}

func (p *KafkaProducer) SendInstantEvent(ctxTimeout context.Context, keyUUID string, event []byte) error {
	msg := kafka.Message{
		Value: event,
	}
	if keyUUID != "" {
		msg.Key = []byte(keyUUID)
	}
	if errSendMessage := p.InstantProducer.WriteMessages(ctxTimeout, msg); errSendMessage != nil {
		return errSendMessage
	}
	return nil
}
func (p *KafkaProducer) SendBatchEvent(ctxTimeout context.Context, batchEvent [][]byte) error {
	sliceMessage := make([]kafka.Message, 0, len(batchEvent))
	for _, event := range batchEvent {
		sliceMessage = append(sliceMessage, kafka.Message{
			Value: event,
		})
	}
	if errSendMessage := p.BatchProducer.WriteMessages(ctxTimeout, sliceMessage...); errSendMessage != nil {
		return errSendMessage
	}
	return nil
}
func (p *KafkaProducer) CloseProducer() {
	if errClose := p.InstantProducer.Close(); errClose != nil {
		p.Logger.Error("failed to close instant producer kafka connection: " + errClose.Error())
	}
	if errClose := p.BatchProducer.Close(); errClose != nil {
		p.Logger.Error("failed to close batch producer kafka connection: " + errClose.Error())
	}
}
