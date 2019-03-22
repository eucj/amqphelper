package amqphelper

import (
	"errors"
	"fmt"
	"sync"

	"github.com/streadway/amqp"
)

//Config is a configuration object of AMQP standard parameters
type Configuration struct {
	Host                    string
	RoutingKey              string
	ContentType             string
	Exchange                string
	AutoAcknowledgeMessages bool
	Durable                 bool
	DeleteIfUnused          bool
	Exclusive               bool
	NoWait                  bool
	NoLocal                 bool
	arguments               amqp.Table
}

//Queue is the object defined by the Configuration object
type Queue struct {
	*sync.WaitGroup
	Connected     bool
	connection    *amqp.Connection
	channel       *amqp.Channel
	internalQueue *amqp.Queue
	Config        *Configuration
}

type Message struct {
	*amqp.Delivery
}

//GetQueue receives Config object and returns a queue for publishing and consuming
func GetQueue(config *Configuration) (*Queue, error) {
	var wg sync.WaitGroup
	q := Queue{&wg, false, nil, nil, nil, nil}
	err := q.connect(config.Host)
	if err != nil {
		return nil, err
	}
	err = q.openChannel()
	if err != nil {
		return nil, err
	}
	iq, err := q.channel.QueueDeclare(config.RoutingKey, config.Durable, config.DeleteIfUnused, config.Exclusive, config.NoWait, config.arguments)
	if err != nil {
		return nil, err
	}
	q.internalQueue = &iq
	q.Config = config
	return &q, nil
}

//Publish publishes a message to the queue with the initialized
func (q *Queue) Publish(message []byte, mandatory, immediate bool) error {
	if q.channel == nil {
		return fmt.Errorf("Queue has not been initialized")
	}
	return q.channel.Publish(q.Config.Exchange, q.Config.RoutingKey, mandatory, immediate, amqp.Publishing{ContentType: q.Config.ContentType, Body: []byte(message)})
}

// GetConsumer returns a consumer with the specified id
func (q *Queue) GetConsumer(ConsumerID string) (<-chan amqp.Delivery, error) {
	return q.channel.Consume(q.Config.RoutingKey, ConsumerID, q.Config.AutoAcknowledgeMessages, q.Config.Exclusive, q.Config.NoLocal, q.Config.NoWait, q.Config.arguments)
}

//ProcessIncomingMessages initializes a consumer and processes each received message by passing it to the argument function in a separate goroutine. Queue.Wait() should be called next
func (q *Queue) ProcessIncomingMessages(ConsumerID string, f func(m *Message)) error {

	msgs, err := q.GetConsumer(ConsumerID)
	if err != nil {
		return err
	}
	q.Add(1)
	go func() {
		for msg := range msgs {
			f(&Message{&msg})
		}
	}()
	return nil
}

func (q *Queue) connect(host string) error {
	if q.Connected != false {
		return fmt.Errorf("Error: Queue is already connected")
	}
	conn, err := amqp.Dial(host)
	if err != nil {
		return err
	}
	q.connection = conn
	q.Connected = true
	return nil
}

func (q *Queue) openChannel() error {
	if q.connection == nil {
		return errors.New("No connection to queue")
	}
	ch, err := q.connection.Channel()
	if err != nil {
		return err
	}
	q.channel = ch
	return nil
}
