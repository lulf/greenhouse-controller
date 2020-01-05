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
	"net"

	"pack.ag/amqp"
)

type CommandControl struct {
	address    string
	username   string
	password   string
	tlsEnabled bool
	caPem      []byte
	tcpConn    net.Conn
	amqpClient *amqp.Client
	sender     *amqp.Sender
}

func NewCommandControl(address string, username string, password string, tlsEnabled bool, caPem []byte) *CommandControl {
	return &CommandControl{
		address:    address,
		username:   username,
		password:   password,
		tlsEnabled: tlsEnabled,
		caPem:      caPem,
		tcpConn:    nil,
		amqpClient: nil,
		sender:     nil,
	}
}

func (cc *CommandControl) Close() error {
	if cc.amqpClient != nil {
		cc.amqpClient.Close()
	}

	if cc.tcpConn != nil {
		cc.tcpConn.Close()
	}

	return nil
}

func (cc *CommandControl) Connect(target string) error {
	if cc.tcpConn == nil {
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
			cc.tcpConn = tlsConn
		} else {
			tcpConn, err := net.Dial("tcp", cc.address)
			if err != nil {
				return err
			}
			cc.tcpConn = tcpConn
		}
	}

	if cc.amqpClient == nil {
		opts := []amqp.ConnOption{
			amqp.ConnContainerID("event-sink"),
		}
		if cc.username != "" && cc.password != "" {
			opts = append(opts, amqp.ConnSASLPlain(cc.username, cc.password))
		}
		amqpClient, err := amqp.New(cc.tcpConn, opts...)
		if err != nil {
			return err
		}
		cc.amqpClient = amqpClient
	}

	session, err := cc.amqpClient.NewSession()
	if err != nil {
		return err
	}
	s, err := session.NewSender(amqp.LinkTargetAddress(target))
	if err != nil {
		return err
	}

	cc.sender = s
	return nil
}

func (cc *CommandControl) Send(ctx context.Context, tenantId string, deviceId string, command string, params *map[string]string) error {
	data := make(map[string]string, 1)
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
	return cc.sender.Send(ctx, message)
}
