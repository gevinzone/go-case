// Copyright 2025 igevin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQClient interface {
	Close() error
	Send(queueName string, contentType string, body []byte) error
	Receive(queueName string) (<-chan amqp.Delivery, error)
}

type RabbitMQClientImpl struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	qs   map[string]*amqp.Queue
}

func NewRabbitMQClient(conn *amqp.Connection) (RabbitMQClient, error) {
	return &RabbitMQClientImpl{conn: conn, qs: make(map[string]*amqp.Queue)}, nil
}

func (r *RabbitMQClientImpl) getChannel() (*amqp.Channel, error) {
	if r.ch != nil {
		return r.ch, nil
	}
	ch, err := r.conn.Channel()
	if err != nil {
		return nil, err
	}
	r.ch = ch
	return ch, nil
}

func (r *RabbitMQClientImpl) DeclareQueue(queueName string) error {
	ch, err := r.conn.Channel()
	if err != nil {
		return err
	}
	q, err := ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return err
	}
	r.qs[queueName] = &q
	return nil
}

func (r *RabbitMQClientImpl) getQueue(queueName string) (*amqp.Queue, error) {
	_, ok := r.qs[queueName]
	if !ok {
		err := r.DeclareQueue(queueName)
		if err != nil {
			return nil, err
		}
	}
	return r.qs[queueName], nil
}

func (r *RabbitMQClientImpl) Close() error {
	if r.ch == nil {
		return nil
	}
	return r.ch.Close()
}

//func (r *RabbitMQClientImpl) Channel() (*amqp.Channel, error) {
//	return r.conn.Channel()
//}

func (r *RabbitMQClientImpl) Send(queueName string, contentType string, body []byte) error {
	ch, err := r.getChannel()
	if err != nil {
		return err
	}
	defer ch.Close()

	q, err := r.getQueue(queueName)
	if err != nil {
		return err
	}

	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: contentType,
			Body:        body,
		})
	if err != nil {
		return err
	}
	return nil
}

func (r *RabbitMQClientImpl) Receive(queueName string) (<-chan amqp.Delivery, error) {
	ch, err := r.getChannel()
	if err != nil {
		return nil, err
	}

	q, err := r.getQueue(queueName)
	if err != nil {
		return nil, err
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}
