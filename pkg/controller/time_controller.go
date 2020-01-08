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
	store        *eventstore.EventStore
	cc           *commandcontrol.CommandControl
	tenantId     string
	knownDevices map[string]bool

	waitPeriod time.Duration
}

func NewTimeController(store *eventstore.EventStore, cc *commandcontrol.CommandControl, waitPeriod time.Duration, tenantId string) Controller {
	return &timeController{
		store:        store,
		cc:           cc,
		tenantId:     tenantId,
		waitPeriod:   waitPeriod,
		knownDevices: make(map[string]bool),
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
		c.knownDevices[event.DeviceId] = true
		log.Println("Discovered device with soil sensor", event.DeviceId)
	}
}

func (c *timeController) checkValues(done chan error) {
	log.Println("Starting checkValues loop")
	for {
		time.Sleep(c.waitPeriod)
		for deviceId, _ := range c.knownDevices {
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
		}
	}
}

func (c *timeController) waterPlants(deviceId string, done chan error) error {
	params := make(map[string]interface{})
	params["period"] = 4000
	err := c.cc.Send(context.TODO(), c.tenantId, deviceId, "water", &params)
	if err != nil {
		log.Println("Sending message to device", deviceId, err)
	}
	if err == io.EOF || err == amqp.ErrLinkClosed || err == amqp.ErrSessionClosed {
		log.Println("Send error:", err)
		done <- err
	} else {
		log.Println("Error sending command", err)
	}
	return err
}
