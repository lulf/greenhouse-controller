/*
 * Copyright 2020, Ulf Lilleengen
 * License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).
 */
package eventstore

import (
	"context"
	"encoding/json"
	"log"

	"pack.ag/amqp"
)

type EventStore struct {
	url      string
	client   *amqp.Client
	receiver *amqp.Receiver
	sender   *amqp.Sender
}

func NewEventStore(url string) *EventStore {
	return &EventStore{
		url: url,
	}
}

func (store *EventStore) Close() error {
	if store.client != nil {
		store.client.Close()
	}
	return nil
}

func (store *EventStore) Connect(topic string) error {
	client, err := amqp.Dial(store.url)
	if err != nil {
		return err
	}
	store.client = client

	session, err := store.client.NewSession()
	if err != nil {
		return err
	}

	r, err := session.NewReceiver(amqp.LinkSourceAddress(topic))
	if err != nil {
		return err
	}
	store.receiver = r

	s, err := session.NewSender(amqp.LinkTargetAddress(topic))
	if err != nil {
		return err
	}
	store.sender = s
	return nil
}

func (store *EventStore) Receive(ctx context.Context) (*EventContext, error) {

	msg, err := store.receiver.Receive(ctx)
	if err != nil {
		return nil, err
	}

	var event Event
	err = json.Unmarshal(msg.GetData(), &event)
	if err != nil {
		log.Println("Error decoding message:", err)
		return nil, err
	}
	return &EventContext{Event: &event, message: msg}, nil
}

func (store *EventStore) Send(ctx context.Context, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		log.Print("Serializing event:", err)
		return err
	}
	m := amqp.NewMessage(data)
	return store.sender.Send(ctx, m)
}
