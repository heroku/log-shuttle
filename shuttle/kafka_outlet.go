package shuttle

import "github.com/Shopify/sarama"
import "strings"

type KafkaOutlet struct {
	inbox    <-chan Batch
	stats    chan<- NamedValue
	drops    *Counter
	lost     *Counter
	lostMark int
	producer *sarama.Producer
	config   ShuttleConfig
}

func NewKafkaOutlet(config ShuttleConfig, drops, lost *Counter, stats chan<- NamedValue, inbox <-chan Batch) *KafkaOutlet {
	producerConfig := sarama.NewProducerConfig()
	producerConfig.Timeout = config.Timeout
	producerConfig.Partitioner = new(sarama.RoundRobinPartitioner)
	producerConfig.Compression = sarama.CompressionGZIP

	if err := producerConfig.Validate(); err != nil {
		panic(err)
	}

	kafkaClient, err := sarama.NewClient(config.ProducerId, strings.Split(config.Brokers, ","), sarama.NewClientConfig())

	if err != nil {
		panic(err)
	}

	producer, err := sarama.NewProducer(kafkaClient, producerConfig)

	if err != nil {
		panic(err)
	}

	return &KafkaOutlet{
		drops:    drops,
		lost:     lost,
		lostMark: int(float64(config.BackBuff) * DEPTH_WATERMARK),
		stats:    stats,
		inbox:    inbox,
		config:   config,
		producer: producer,
	}
}

func (ka *KafkaOutlet) Outlet() {
	for batch := range ka.inbox {
		ka.stats <- NewNamedValue("outlet.inbox.length", float64(len(ka.inbox)))

		ka.publish(batch)
	}
}

func (ka *KafkaOutlet) publish(batch Batch) {

	// TODO: Handle dropped messages

}
