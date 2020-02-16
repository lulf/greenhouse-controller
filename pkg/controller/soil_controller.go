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

type soilController struct {
	store                *eventstore.EventStore
	cc                   *commandcontrol.CommandControl
	lowHumidityThreshold float64
	tenantId             string
	lastValue            map[string]float64
	waterPeriod          time.Duration

	waitPeriod time.Duration
}

func NewSoilController(store *eventstore.EventStore, cc *commandcontrol.CommandControl, waitPeriod time.Duration, lowHumidityThreshold float64, tenantId string, waterPeriod time.Duration) Controller {
	return &soilController{
		store:                store,
		cc:                   cc,
		lowHumidityThreshold: lowHumidityThreshold,
		tenantId:             tenantId,
		waitPeriod:           waitPeriod,
		lastValue:            make(map[string]float64),
		waterPeriod:          waterPeriod,
	}
}

func (c *soilController) Run(done chan error) {
	go c.checkValues(done)
	log.Println("Starting receive event loop", c.waitPeriod, c.lowHumidityThreshold, c.waterPeriod)
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

func (c *soilController) handleEvent(event *eventstore.Event) {
	if soil, ok := event.Data["soil"]; ok {
		soilData := soil.(map[string]interface{})
		numSamples := soilData["numSamples"].(float64)
		humidity := soilData["humidity"].([]interface{})
		log.Println("Found soil data", numSamples, humidity)
		if numSamples >= 100 {
			found := false
			var highest float64
			for _, value := range humidity {
				fvalue := value.(float64)
				log.Println("Comparing existing with new", highest, fvalue)
				if fvalue > highest {
					highest = fvalue
					found = true
				}
			}
			if found {
				c.lastValue[event.DeviceId] = highest
				log.Println("Updated last soil value", event.DeviceId, numSamples, highest)
			}
		}
	}
}

func (c *soilController) checkValues(done chan error) {
	log.Println("Starting checkValues loop")
	for {
		for deviceId, value := range c.lastValue {
			if value < c.lowHumidityThreshold {
				// Water if any plant is below threshold
				log.Println("Soil value is below threshold, watering", deviceId, value)
				params := make(map[string]interface{})
				params["period"] = int64(c.waterPeriod / time.Nanosecond)
				err := c.cc.Send(context.TODO(), c.tenantId, deviceId, "water", &params)
				if err != nil {
					log.Println("Sending message to device", deviceId, err)
				}
				if err == io.EOF || err == amqp.ErrLinkClosed || err == amqp.ErrSessionClosed {
					log.Println("Send error:", err)
					done <- err
					break
				}
			} else {
				log.Println("Soil value is above threshold, not watering", deviceId, value)
			}
		}
		time.Sleep(c.waitPeriod * time.Second)
	}
}
