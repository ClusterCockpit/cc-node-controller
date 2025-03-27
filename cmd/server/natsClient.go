package main

import (
	"fmt"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
	"github.com/nats-io/nats.go"
)

type NatsConnection struct {
	conn       *nats.Conn
	sub        *nats.Subscription
	ch         chan *nats.Msg
}

type NatsConfig struct {
	Server              string `json:"server"`
	Port                int    `json:"port"`
	RequestSubject      string `json:"requestSubject"`
	//ReplySubject      string `json:"replySubject"`
	User                string `json:"user"`
	Password            string `json:"password"`
	CredsFile           string `json:"credsFile"`
	NKeySeedFile        string `json:"nkeySeedFile"`
	OutstandingMessages int    `json:"outstandingMessagesInQueue,omitempty"`
}

func ConnectNats(config NatsConfig) (*NatsConnection, error) {
	options := make([]nats.Option, 0)
	if len(config.Password) > 0 {
		options = append(options, nats.UserInfo(config.User, config.Password))
	}
	if len(config.CredsFile) > 0 {
		// TODO do we have to check for file existence here?
		options = append(options, nats.UserCredentials(config.CredsFile))
	}
	if len(config.NKeySeedFile) > 0 {
		r, err := nats.NkeyOptionFromSeed(config.NKeySeedFile)
		if err != nil {
			return nil, fmt.Errorf("Unable to open NKeySeedFile: %w", err)
		}
		options = append(options, r)
	}

	uri := fmt.Sprintf("nats://%s:%d", config.Server, config.Port)
	cclog.ComponentDebug("NATS", "connecting to", uri)
	conn, err := nats.Connect(uri, options...)
	if err != nil {
		return nil, err
	}

	ch := make(chan *nats.Msg, config.OutstandingMessages)
	cclog.ComponentDebug("NATS", "subscribing to", config.RequestSubject)
	sub, err := conn.ChanSubscribe(config.RequestSubject, ch)
	if err != nil {
		return nil, err
	}

	return &NatsConnection{
		conn: conn,
		ch: ch,
		sub: sub,
	}, nil
}

func DisconnectNats(conn *NatsConnection) {
	cclog.ComponentDebug("NATS", "disconnecting ...")
	conn.sub.Unsubscribe()
	close(conn.ch)
	conn.conn.Close()
}
