package kafka

import (
	"context"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
)

func CreateKafkaConnection(kafkaAddress string, topic string) (*kafka.Conn, error) {
	conn, err := kafka.DialLeader(context.Background(), "tcp", kafkaAddress, topic, 0)
	if err != nil {
		return nil, err
	}
	_, err = conn.Brokers()
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func CheckKafkaConnection(kafkaAddress string) error {
	conn, err := kafka.Dial("tcp", kafkaAddress)
	if err != nil {
		return err
	}
	_, err = conn.Brokers()
	if err != nil {
		return err
	}
	return nil
}

func CreateKafkaConnectionOrFatal(kafkaAddress string, topic string) *kafka.Conn {
	conn, err := CreateKafkaConnection(kafkaAddress, topic)
	if err != nil {
		log.Fatalf("unable to connect kafka (%s) topic %s: %s", kafkaAddress, topic, err)
	}
	return conn
}
