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

type KafkaConsumer struct {
	Consumer *kafka.Reader
	Logger   *loggers.Logger
}
type ConfigConsumer struct {
	Brokers       []string
	KafkaUser     string
	KafkaPassword string
	Topic         string
	GroupID       string
}

func NewConsumer(conf *ConfigConsumer, logger *loggers.Logger) (*KafkaConsumer, error) {
	mechanism, errScramMechanism := scram.Mechanism(scram.SHA256, conf.KafkaUser, conf.KafkaPassword)
	if errScramMechanism != nil {
		return nil, errScramMechanism
	}
	customDialer := &kafka.Dialer{
		Timeout:       10 * time.Second,
		DualStack:     false,
		SASLMechanism: mechanism,
	}
	customDialer.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{
			Timeout:   10 * time.Second,
			DualStack: false,
			Resolver: &net.Resolver{
				PreferGo: true,
			},
		}).DialContext(ctx, network, addr)
	}
	consumer := kafka.NewReader(kafka.ReaderConfig{
		Brokers:           conf.Brokers,
		GroupID:           conf.GroupID,
		Topic:             conf.Topic,
		CommitInterval:    0,
		Dialer:            customDialer,
		SessionTimeout:    60 * time.Second,
		HeartbeatInterval: 6 * time.Second,
		RebalanceTimeout:  30 * time.Second,
	})
	return &KafkaConsumer{
		Consumer: consumer,
		Logger:   logger,
	}, nil
}
func (c *KafkaConsumer) WaitEvent(ctxCancel context.Context, processorFunc func([]byte) error) {
	wait := time.Second * 3
	for {
		msg, errReadMessage := c.Consumer.ReadMessage(ctxCancel)
		if errReadMessage != nil {
			c.Logger.Error("failed to read message consumer kafka: " + errReadMessage.Error())
			wait *= 2
			if wait > time.Second*30 {
				wait = time.Second * 30
			}
			time.Sleep(wait)
			continue
		}
		wait = time.Second * 3
		if errProcess := processorFunc(msg.Value); errProcess != nil {
			c.Logger.Error("failed to process message consumer kafka: " + errProcess.Error())
			continue
		}
		ctxTimeout, cancel := context.WithTimeout(ctxCancel, time.Second*3)
		if errCommit := c.Consumer.CommitMessages(ctxTimeout, msg); errCommit != nil {
			c.Logger.Error("failed to commit message consumer kafka: " + errCommit.Error())
		}
		c.Logger.Info("successful process event")
		cancel()
	}
}
func (c *KafkaConsumer) CloseConsumer() {
	if errClose := c.Consumer.Close(); errClose != nil {
		c.Logger.Error("failed to close consumer kafka connection: " + errClose.Error())
	}
}
