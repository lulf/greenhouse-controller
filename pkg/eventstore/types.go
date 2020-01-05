/*
 * Copyright 2020, Ulf Lilleengen
 * License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).
 */
package eventstore

import (
	"pack.ag/amqp"
)

type Event struct {
	DeviceId     string                 `json:"deviceId"`
	CreationTime int64                  `json:"creationTime"`
	Data         map[string]interface{} `json:"data"`
}

type EventContext struct {
	Event   *Event
	message *amqp.Message
}

func (c *EventContext) Accept() error {
	return c.message.Accept()
}

func (c *EventContext) Reject(e *amqp.Error) error {
	return c.message.Reject(e)
}
