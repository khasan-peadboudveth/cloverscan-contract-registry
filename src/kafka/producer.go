package kafka

import (
	"context"
	"fmt"
	"github.com/clover-network/cloverscan-contract-registry/src/config"
	protocontract "github.com/clover-network/cloverscan-proto-contract"
	"github.com/golang/protobuf/proto"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Producer struct {
	config      *config.Config
	producers   map[string]*kafka.Writer
	connections map[string]*kafka.Conn
	mu          sync.Mutex
}

func NewProducer(config *config.Config) *Producer {
	return &Producer{
		config:      config,
		producers:   make(map[string]*kafka.Writer),
		connections: make(map[string]*kafka.Conn),
	}
}

func (producer *Producer) Start() {

}

func (producer *Producer) createProducerForTopic(topic string) {
	producer.mu.Lock()
	defer producer.mu.Unlock()
	conn := CreateKafkaConnectionOrFatal(producer.config.KafkaAddress, topic)
	writer := &kafka.Writer{
		Addr:         kafka.TCP(producer.config.KafkaAddress),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		MaxAttempts:  10,
		ErrorLogger:  log.New(),
		WriteTimeout: 20 * time.Second,
		ReadTimeout:  20 * time.Second,
		BatchBytes:   104857600,
	}
	producer.producers[topic] = writer
	producer.connections[topic] = conn
}

func (producer *Producer) Stop() {
	for _, conn := range producer.producers {
		_ = conn.Close()
	}
	for _, conn := range producer.connections {
		_ = conn.Close()
	}
}

func (producer *Producer) WriteMessageSilently(topic string, message proto.Message) {
	err := producer.WriteMessage(topic, message)
	if err != nil {
		log.Errorf("failed to write message to topic %s: %s", topic, err)
	}
}

func (producer *Producer) WriteMessage(topic string, messages ...proto.Message) error {
	writer, ok := producer.producers[topic]
	if !ok {
		producer.createProducerForTopic(topic)
	}
	writer, ok = producer.producers[topic]
	if !ok {
		return fmt.Errorf("there is no kafka producer for such topic: %s", topic)
	}
	encodedKafkaMessages := make([]kafka.Message, len(messages))
	for i, message := range messages {
		buf := proto.NewBuffer([]byte{})
		if err := buf.EncodeMessage(message); err != nil {
			return err
		}
		encodedKafkaMessages[i] = kafka.Message{Value: buf.Bytes()}
	}
	err := writer.WriteMessages(context.Background(), encodedKafkaMessages...)
	return err
}

func (producer *Producer) WriteBlock(messages ...*protocontract.BlockChanged) error {
	protoMessages := make([]proto.Message, len(messages))
	for i, message := range messages {
		protoMessages[i] = message
	}
	return producer.WriteMessage("v1.blockchainextractor.blockchanged", protoMessages...)
}
