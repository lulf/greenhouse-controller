/*
 * Copyright 2019, Ulf Lilleengen
 * License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).
 */
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/lulf/greenhouse-controller/pkg/commandcontrol"
	"github.com/lulf/greenhouse-controller/pkg/controller"
	"github.com/lulf/greenhouse-controller/pkg/eventstore"
)

func main() {
	var eventstoreAddr string
	var controlAddr string
	var tenantId string
	var password string
	var tlsEnabled bool
	var cafile string

	var waitPeriod int64
	var waterPeriod int64
	var lowHumidityThreshold float64

	flag.StringVar(&eventstoreAddr, "a", "amqp://127.0.0.1:5672", "Address of AMQP event store")
	flag.StringVar(&controlAddr, "e", "messaging.bosch-iot-hub.com:5671", "Address of Eclipse Hono Command and Control endpoint")
	flag.StringVar(&tenantId, "t", "", "Tenant ID for Bosch IoT Hub")
	flag.StringVar(&password, "p", "", "Password for Bosch IoT Hub")
	flag.BoolVar(&tlsEnabled, "s", false, "Enable TLS")
	flag.StringVar(&cafile, "c", "", "Certificate CA file")
	flag.Int64Var(&waitPeriod, "w", 3600, "Wait period between watering checks")
	flag.Float64Var(&lowHumidityThreshold, "h", 0.0, "Lowest soil value before watering")
	flag.Int64Var(&waterPeriod, "r", 6000, "Pump water period")

	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Printf("    -e example.com:5672 [-t tenant_id] [-p password] [-s] [-c cafile] [-a 127.0.0.1:5672] \n")
		flag.PrintDefaults()
	}
	flag.Parse()

	var err error
	var ca []byte
	if cafile != "" {
		ca, err = ioutil.ReadFile(cafile)
		if err != nil {
			log.Fatal("Reading CA file:", err)
		}
	}

	username := fmt.Sprintf("messaging@%s", tenantId)
	cc := commandcontrol.NewCommandControl(controlAddr, username, password, tlsEnabled, ca, fmt.Sprintf("command/%s", tenantId))

	store := eventstore.NewEventStore(eventstoreAddr)
	err = store.Connect("events")
	if err != nil {
		log.Fatal("Connecting to event store:", err)
	}
	defer store.Close()

	controller := controller.NewSoilController(store, cc, time.Duration(waitPeriod), lowHumidityThreshold, tenantId, time.Duration(waterPeriod))

	done := make(chan error)
	go controller.Run(done)

	// Exit if any of our processes complete
	for {
		err := <-done
		if err != nil {
			log.Println("Finished with error", err)
			os.Exit(1)
		} else {
			log.Println("Finished without error")
			os.Exit(0)
		}
	}
}
