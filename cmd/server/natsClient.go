package main

import (
	"fmt"
	"os"

	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
	"github.com/nats-io/nats.go"
)

type NatsConnection struct {
	conn       *nats.Conn
	sub        *nats.Subscription
	ch         chan *nats.Msg
	outSubject string
}

type NatsConfig struct {
	Hostname            string `json:"hostname"`
	Port                int    `json:"port"`
	Username            string `json:"username,omitempty"`
	Password            string `json:"password,omitempty"`
	NkeyFile            string `json:"nkey_file,omitempty"`
	InputSubjectPrefix  string `json:"input_subject_prefix,omitempty"`
	InputSubject        string `json:"input_subject,omitempty"`
	OutputSubjectPrefix string `json:"output_subject_prefix,omitempty"`
	OutputSubject       string `json:"output_subject,omitempty"`
	OutstandingMessages int    `json:"outstanding_messages_in_queue,omitempty"`
	subject             string
	outSubject          string
}

func ConnectNats(config NatsConfig) (NatsConnection, error) {
	var uinfo nats.Option = nil
	c := NatsConnection{
		conn:       nil,
		sub:        nil,
		ch:         nil,
		outSubject: config.outSubject,
	}
	if len(config.Username) > 0 && len(config.Password) > 0 {
		uinfo = nats.UserInfo(config.Username, config.Password)
	} else if len(config.NkeyFile) > 0 {
		_, err := os.Stat(config.NkeyFile)
		if err == nil {
			uinfo = nats.UserCredentials(config.NkeyFile)
		} else {
			cclog.ComponentError("NATS", "NKEY file", config.NkeyFile, "does not exist: %v", err.Error())
			return c, err
		}
	}
	uri := fmt.Sprintf("%s:%d", config.Hostname, config.Port)
	cclog.ComponentDebug("NATS", "connecting to", uri)
	conn, err := nats.Connect(uri, uinfo)
	if err != nil {
		return c, err
	}

	ch := make(chan *nats.Msg, config.OutstandingMessages)
	cclog.ComponentDebug("NATS", "subscribing to", config.subject)
	sub, err := conn.ChanSubscribe(config.subject, ch)
	if err != nil {
		return c, err
	}
	c.conn = conn
	c.ch = ch
	c.sub = sub
	return c, nil
}

func PublishNats(conn NatsConnection, event lp.CCMessage) error {
	cclog.ComponentDebug("NATS", "Publish", conn.outSubject, ":", event.ToLineProtocol(nil))
	return conn.conn.Publish(conn.outSubject, []byte(event.ToLineProtocol(nil)))
}

func DisconnectNats(conn NatsConnection) {
	cclog.ComponentDebug("NATS", "disconnecting ...")
	conn.sub.Unsubscribe()
	close(conn.ch)
	conn.conn.Close()
}
