/*
 * Copyright 2020, Ulf Lilleengen
 * License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).
 */
package controller

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/lulf/greenhouse-controller/pkg/commandcontrol"
	"github.com/lulf/greenhouse-controller/pkg/eventstore"

	"pack.ag/amqp"
)

type timeController struct {
	store       *eventstore.EventStore
	cc          *commandcontrol.CommandControl
	tenantId    string
	lastWatered map[string]time.Time

	waitPeriod time.Duration
}

func NewTimeController(store *eventstore.EventStore, cc *commandcontrol.CommandControl, waitPeriod time.Duration, tenantId string) Controller {
	return &timeController{
		store:       store,
		cc:          cc,
		tenantId:    tenantId,
		waitPeriod:  waitPeriod,
		lastWatered: make(map[string]time.Time),
	}
}

func (c *timeController) Run(done chan error) {
	go c.checkValues(done)
	log.Println("Starting receive event loop")
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
				log.Println("Error processing event", err)
			}
		}
	}
}

func (c *timeController) handleEvent(event *eventstore.Event) {
	if _, ok := event.Data["soil"]; ok {
		if _, known := c.lastWatered[event.DeviceId]; !known {
			c.lastWatered[event.DeviceId] = time.Time{}
			log.Println("Discovered device with soil sensor", event.DeviceId)
		}
	}
}

func (c *timeController) checkValues(done chan error) {
	log.Println("Starting checkValues loop")
	for {
		now := time.Now().UTC()
		log.Println("Checking device water schedule", now, c.lastWatered)
		for deviceId, lastWatered := range c.lastWatered {
			if lastWatered.Add(c.waitPeriod).Before(now) {
				log.Println("Watering plants", deviceId)
				// Water if any plant is below threshold
				err := c.waterPlants(deviceId, done)
				if err != nil {
					return
				}

				time.Sleep(10 * time.Second)

				// Twice
				err = c.waterPlants(deviceId, done)
				if err != nil {
					return
				}
				c.lastWatered[deviceId] = now
			} else {
				log.Println("Not watering plants", deviceId)
			}
		}
		time.Sleep(120 * time.Second)
	}
}

func (c *timeController) waterPlants(deviceId string, done chan error) error {
	params := make(map[string]interface{})
	params["period"] = 3000
	err := c.cc.Send(context.TODO(), c.tenantId, deviceId, "water", &params)
	if err != nil {
		log.Println("Sending message to device", deviceId, err)
	}
	if err == io.EOF || err == amqp.ErrLinkClosed || err == amqp.ErrSessionClosed {
		log.Println("Send error:", err)
		done <- err
	}
	return err
}
