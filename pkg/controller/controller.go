/*
 * Copyright 2020, Ulf Lilleengen
 * License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).
 */
package controller

import (
	"context"
	"io"
	"log"
	"math"
	"time"

	"github.com/lulf/greenhouse-controller/pkg/commandcontrol"
	"github.com/lulf/greenhouse-controller/pkg/eventstore"

	"pack.ag/amqp"
)

type controller struct {
	store               *eventstore.EventStore
	cc                  *commandcontrol.CommandControl
	lowestSoilThreshold float64
	tenantId            string
	lastValue           map[string]float64

	waitPeriod time.Duration
}

func NewController(store *eventstore.EventStore, cc *commandcontrol.CommandControl, waitPeriod time.Duration, lowestSoilThreshold float64, tenantId string) *controller {
	return &controller{
		store:               store,
		cc:                  cc,
		lowestSoilThreshold: lowestSoilThreshold,
		tenantId:            tenantId,
		waitPeriod:          waitPeriod,
		lastValue:           make(map[string]float64),
	}
}

func (c *controller) Run(done chan error) {
	go c.checkValues(done)
	for {
		if eventCtx, err := c.store.Receive(context.TODO()); err == nil {
			c.handleEvent(eventCtx.Event)
			eventCtx.Accept()
		} else {
			if err == io.EOF || err == amqp.ErrLinkClosed || err == amqp.ErrSessionClosed {
				log.Println("Receive error:", err)
				done <- err
				break
			} else {
				log.Println("Error processing event", eventCtx.Event, err)
				eventCtx.Reject(nil)
			}
		}
	}
}

func (c *controller) handleEvent(event *eventstore.Event) {
	if soil, ok := event.Data["soil"]; ok {
		smallest := math.MaxFloat64
		found := false
		for _, value := range soil.([]interface{}) {
			fvalue := value.(float64)
			if fvalue < smallest {
				smallest = fvalue
				found = true
			}
		}
		if found {
			c.lastValue[event.DeviceId] = smallest
		}
	}
}

func (c *controller) checkValues(done chan error) {
	for {
		for deviceId, value := range c.lastValue {
			if value < c.lowestSoilThreshold {
				// Water if any plant is below threshold
				log.Println("Soil value is below threshold, watering", deviceId, value)
				err := c.cc.Send(context.TODO(), c.tenantId, deviceId, "water", nil)
				if err != nil {
					log.Println("Sending message to device", deviceId, err)
				}
				if err == io.EOF || err == amqp.ErrLinkClosed || err == amqp.ErrSessionClosed {
					log.Println("Send error:", err)
					done <- err
					break
				} else {
					log.Println("Error sending command", err)
				}
			} else {
				log.Println("Soil value is above threshold, not watering", deviceId, value)
			}
		}
		time.Sleep(c.waitPeriod)
	}
}
