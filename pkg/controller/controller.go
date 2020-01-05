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

type controller struct {
	store               *eventstore.EventStore
	cc                  *commandcontrol.CommandControl
	lowestSoilThreshold float64
	tenantId            string

	waitPeriod time.Duration
}

func NewController(store *eventstore.EventStore, cc *commandcontrol.CommandControl, waitPeriod time.Duration, lowestSoilThreshold float64, tenantId string) *controller {
	return &controller{
		store:               store,
		cc:                  cc,
		lowestSoilThreshold: lowestSoilThreshold,
		tenantId:            tenantId,
		waitPeriod:          waitPeriod,
	}
}

func (c *controller) Run(done chan error) {
	for {
		if eventCtx, err := c.store.Receive(context.TODO()); err == nil {
			err := c.handleEvent(eventCtx.Event)
			if err != nil {
				if err == io.EOF || err == amqp.ErrLinkClosed || err == amqp.ErrSessionClosed {
					log.Println("Receive error:", err)
					done <- err
					break
				} else {
					log.Println("Error processing event", eventCtx.Event, err)
					eventCtx.Reject(nil)
				}
			} else {
				eventCtx.Accept()
			}
		} else {
			log.Println("Receive error:", err)
			done <- err
			break
		}
		time.Sleep(c.waitPeriod)
	}
}

func (c *controller) handleEvent(event *eventstore.Event) error {
	isBelow := false
	if soil, ok := event.Data["soil"]; ok {
		for _, value := range soil.([]interface{}) {
			if value.(float64) < c.lowestSoilThreshold {
				isBelow = true
			}
		}
	}

	// Water if any plant is below threshold
	if isBelow {
		log.Println("Soil value is below threshold, watering")
		err := c.cc.Send(context.TODO(), c.tenantId, event.DeviceId, "water", nil)
		if err != nil {
			log.Println("Sending message to device", event.DeviceId, err)
			return err
		}

	}
	return nil
}
