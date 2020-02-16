/*
 * Copyright 2019, Ulf Lilleengen
 * License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).
 */
package commandcontrol

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"pack.ag/amqp"
)

type CommandControl struct {
	url        string
	username   string
	password   string
	tlsEnabled bool
	caPem      []byte
	address    string
}

func NewCommandControl(url string, username string, password string, tlsEnabled bool, caPem []byte, address string) *CommandControl {
	return &CommandControl{
		url:        url,
		username:   username,
		password:   password,
		tlsEnabled: tlsEnabled,
		caPem:      caPem,
		address:    address,
	}
}

func (cc *CommandControl) Send(ctx context.Context, tenantId string, deviceId string, command string, params *map[string]interface{}) error {
	var connection net.Conn
	if cc.tlsEnabled {
		certPool := x509.NewCertPool()
		if cc.caPem != nil {
			certPool.AppendCertsFromPEM(cc.caPem)
		}
		config := tls.Config{RootCAs: certPool}
		tlsConn, err := tls.Dial("tcp", cc.address, &config)
		if err != nil {
			return err
		}
		connection = tlsConn
	} else {
		tcpConn, err := net.Dial("tcp", cc.address)
		if err != nil {
			return err
		}
		connection = tcpConn
	}

	opts := []amqp.ConnOption{
		amqp.ConnContainerID("greenhouse-controller"),
	}
	if cc.username != "" && cc.password != "" {
		opts = append(opts, amqp.ConnSASLPlain(cc.username, cc.password))
	}

	amqpClient, err := amqp.New(connection, opts...)
	if err != nil {
		return err
	}

	session, err := amqpClient.NewSession()
	if err != nil {
		return err
	}
	sender, err := session.NewSender(amqp.LinkTargetAddress(cc.address))
	if err != nil {
		return err
	}

	data := make(map[string]interface{}, 1)
	data["command"] = command
	if params != nil {
		for key, value := range *params {
			data[key] = value
		}
	}

	output, err := json.Marshal(data)
	if err != nil {
		return err
	}

	message := amqp.NewMessage(output)
	message.Properties = &amqp.MessageProperties{}
	message.Properties.To = fmt.Sprintf("command/%s/%s", tenantId, deviceId)
	message.Properties.Subject = command
	log.Println("Sending message", message)
	return sender.Send(ctx, message)
}
