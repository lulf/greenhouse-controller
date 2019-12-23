/*
 * Copyright 2019, Ulf Lilleengen
 * License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).
 */
package main

func main() {
	var eventstoreAddr string
	var controlAddr string
	var tenantId string
	var password string
	var tlsEnabled bool
	var cafile string

	flag.StringVar(&eventstoreAddr, "a", "amqp://127.0.0.1:5672", "Address of AMQP event store")
	flag.StringVar(&controlAddr, "e", "messaging.bosch-iot-hub.com:5671", "Address of Eclipse Hono Command and Control endpoint")
	flag.StringVar(&tenantId, "t", "", "Tenant ID for Bosch IoT Hub")
	flag.StringVar(&password, "p", "", "Password for AMQP event source")
	flag.BoolVar(&tlsEnabled, "s", false, "Enable TLS")
	flag.StringVar(&cafile, "c", "", "Certificate CA file")

	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Printf("    -e example.com:5672 [-t tenant_id] [-p password] [-s] [-c cafile] [-a 127.0.0.1:5672] \n")
		flag.PrintDefaults()
	}
	flag.Parse()
}
