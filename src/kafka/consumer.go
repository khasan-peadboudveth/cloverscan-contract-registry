package kafka

import (
	"context"
	"github.com/clover-network/cloverscan-contract-registry/src/config"
	proto "github.com/clover-network/cloverscan-proto-contract"
	proto2 "github.com/gogo/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/olebedev/emitter"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"time"
)

type Consumer struct {
	config  *config.Config
	emitter *emitter.Emitter
}

func NewConsumer(config *config.Config, emitter *emitter.Emitter) *Consumer {
	return &Consumer{
		config:  config,
		emitter: emitter,
	}
}

func (c *Consumer) Start(handleBlockChanged func(changes []*proto.BlockChanged) error) {
	go c.createBatchConsumerForTopic("v1.blockchainextractor.blockchanged", func(messages []kafka.Message) error {
		entities := make([]*proto.BlockChanged, len(messages))
		for i, message := range messages {
			entity := &proto.BlockChanged{}
			if err := c.parseDelimitedMessage(message, entity); err != nil {
				return err
			}
			entities[i] = entity
		}
		err := handleBlockChanged(entities)
		if err != nil {
			return err
		}
		return nil
	})
	go c.createConsumerForTopic("v1.blockchainextractor.nodechanged", func(message kafka.Message) error {
		entity := &proto.NodeChanged{}
		if err := c.parseDelimitedMessage(message, entity); err != nil {
			return err
		}
		<-c.emitter.Emit("kafkaConsumer/nodesChanged", entity)
		return nil
	})
}

func (c *Consumer) createBatchConsumerForTopic(topic string, handler func([]kafka.Message) error) {
	log.Infof("starting consumer for topic %s", topic)
	reader := kafka.NewReader(kafka.ReaderConfig{
		GroupID: c.config.KafkaGroup,
		Brokers: []string{
			c.config.KafkaAddress,
		},
		StartOffset: kafka.FirstOffset,
		Topic:       topic,
	})
	defer func() {
		if err := reader.Close(); err != nil {
			log.Errorf("failed to close reader for topic %s: %s", topic, err)
		}
	}()
	const MaxBytes = 1024 * 1024 * 10 // 10 MB
	const MaxSize = 200
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		batch := make([]kafka.Message, 0)
		batchSize := 0
		for {
			message, err := reader.FetchMessage(ctx)
			if err == context.DeadlineExceeded {
				break
			}
			if err != nil {
				log.Errorf("unable to read message from kafka for topic %s: %s", topic, err)
				break
			}
			batch = append(batch, message)
			batchSize += len(message.Value)
			if batchSize >= MaxBytes || len(batch) >= MaxSize {
				break
			}
		}
		cancel()
		if len(batch) == 0 {
			continue
		}
		err := handler(batch)
		if err != nil {
			log.Errorf("failed to parse message, trying again in 5 seconds: %s", err)
			time.Sleep(5 * time.Second)
			continue
		}
		ctx = context.Background()
		err = reader.CommitMessages(ctx, batch[len(batch)-1])
		if err != nil {
			log.Errorf("failed to commit messages: %s", err)
			time.Sleep(5 * time.Second)
			continue
		}
	}
}

func (c *Consumer) parseDelimitedMessage(message kafka.Message, result proto2.Message) error {
	buf := proto2.NewBuffer(message.Value)
	bytes, err := buf.DecodeRawBytes(false)
	if err != nil {
		return err
	}
	marshaller := runtime.ProtoMarshaller{}
	return marshaller.Unmarshal(bytes, result)
}

func (c *Consumer) createConsumerForTopic(topic string, handler func(kafka.Message) error) {
	log.Infof("starting consumer for topic %s", topic)
	reader := kafka.NewReader(kafka.ReaderConfig{
		GroupID: c.config.KafkaGroup,
		Brokers: []string{
			c.config.KafkaAddress,
		},
		StartOffset: kafka.FirstOffset,
		Topic:       topic,
	})
	ctx := context.Background()
	for {
		message, err := reader.FetchMessage(ctx)
		if err != nil {
			log.Errorf("unable to read message from kafka for topic %s: %s", topic, err)
			break
		}
		err = handler(message)
		if err != nil {
			log.Errorf("failed to parse message, trying again in 5 seconds: %s", err)
			time.Sleep(5 * time.Second)
			continue
		}
		err = reader.CommitMessages(ctx, message)
		if err != nil {
			log.Errorf("failed to commit messages: %s", err)
			time.Sleep(5 * time.Second)
			continue
		}
	}
	if err := reader.Close(); err != nil {
		log.Errorf("failed to close reader for topic %s: %s", topic, err)
	}
}

func (c *Consumer) Stop() {
}
