package main

import (
	"fmt"
	"os"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
	"github.com/nats-io/nats.go"
)

type NatsConnection struct {
	conn       *nats.Conn
	sub        *nats.Subscription
	ch         chan *nats.Msg
}

type NatsConfig struct {
	Hostname            string `json:"hostname"`
	Port                int    `json:"port"`
	Username            string `json:"username,omitempty"`
	Password            string `json:"password,omitempty"`
	NkeyFile            string `json:"nkey_file,omitempty"`
	RequestSubject      string `json:"request_subject,omitempty"`
	//ReplySubject      string `json:"reply_subject,omitempty"`
	OutstandingMessages int    `json:"outstanding_messages_in_queue,omitempty"`
}

func ConnectNats(config NatsConfig) (*NatsConnection, error) {
	var uinfo nats.Option = nil
	if len(config.Username) > 0 && len(config.Password) > 0 {
		uinfo = nats.UserInfo(config.Username, config.Password)
	} else if len(config.NkeyFile) > 0 {
		_, err := os.Stat(config.NkeyFile)
		if err == nil {
			uinfo = nats.UserCredentials(config.NkeyFile)
		} else {
			cclog.ComponentError("NATS", "NKEY file", config.NkeyFile, "does not exist: %v", err.Error())
			return nil, err
		}
	}
	uri := fmt.Sprintf("%s:%d", config.Hostname, config.Port)
	cclog.ComponentDebug("NATS", "connecting to", uri)
	conn, err := nats.Connect(uri, uinfo)
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
