package cccontrolclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
	topo "github.com/ClusterCockpit/cc-node-controller/pkg/ccTopology"
	"github.com/nats-io/nats.go"
)

type CCControlListEntry struct {
	Category    string `json:"category"`
	Name        string `json:"name"`
	DeviceType  string `json:"device_type"`
	Description string `json:"description"`
	Methods     string `json:"methods"`
}

type CCControlList struct {
	Controls []CCControlListEntry `json:"controls"`
}

type CCControlTopology struct {
	HWthreads []topo.HwthreadEntry `json:"hwthreads"`
	CpuInfo   topo.CpuInformation  `json:"cpu_info"`
}

type ccControlClient struct {
	conn           *nats.Conn
	hostname       string
	natsCfg        NatsConfig
}

type CCControlClient interface {
	Init(natsCfg NatsConfig) error
	GetControls(hostname string) (*CCControlList, error)
	GetTopology(hostname string) (*CCControlTopology, error)
	GetControlValue(hostname, control string, device string, deviceID string) (string, error)
	SetControlValue(hostname, control string, device string, deviceID string, value string) error
	Close()
}

type NatsConfig struct {
	Server         string `json:"server"`
	Port           uint16 `json:"port"`
	RequestSubject string `json:"requestSubject"`
	// TODO actually implement ReplySubject. Currently, we use NATS request/reply,
	// which by default uses subject `_INBOX.XXXXXXXXX` as reply subject.
	// However, this is difficult to restrict in terms of permissions.
	//ReplySubject   string `json:"replySubject"`
	User           string `json:"user"`
	Password       string `json:"password"`
	CredsFile      string `json:"credsFile"`
	NKeySeedFile   string `json:"nkeySeedFile"`
}

func NewCCControlClient(natsConfig NatsConfig) (CCControlClient, error) {
	n := new(ccControlClient)
	err := n.Init(natsConfig)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func NatsReceive(m *nats.Msg) ([]lp.CCMessage, error) {
	out, err := lp.FromBytes(m.Data)
	if err != nil {
		return nil, fmt.Errorf("lp.FromBytes failed: %v", err)
	}
	return out, nil
}

func (c *ccControlClient) Init(natsCfg NatsConfig) error {
	h, err := os.Hostname()
	if err != nil {
		return errors.New("failed to get hostname for CCControlClient")
	}

	c.natsCfg = natsCfg
	c.hostname = h
	return c.connect()
}

func (c *ccControlClient) Close() {
	c.conn.Close()
}

func (c *ccControlClient) connect() error {
	addr := nats.DefaultURL
	if len(c.natsCfg.Server) > 0 {
		addr = c.natsCfg.Server
		if c.natsCfg.Port > 0 {
			addr = fmt.Sprintf("nats://%s:%d", addr, c.natsCfg.Port)
		}
	}

	options := make([]nats.Option, 0)
	if len(c.natsCfg.Password) > 0 {
		options = append(options, nats.UserInfo(c.natsCfg.User, c.natsCfg.Password))
	}
	if len(c.natsCfg.CredsFile) > 0 {
		options = append(options, nats.UserCredentials(c.natsCfg.CredsFile))
	}
	if len(c.natsCfg.NKeySeedFile) > 0 {
		r, err := nats.NkeyOptionFromSeed(c.natsCfg.NKeySeedFile)
		if err != nil {
			return fmt.Errorf("Unable to open NKeySeedFile: %w" ,err)
		}
		options = append(options, r)
	}

	conn, err := nats.Connect(addr, options...)
	if err != nil {
		err := fmt.Errorf("failed to establish connection to %s: %v", addr, err.Error())
		cclog.ComponentError("CCControlClient", err.Error())
		return err
	}
	c.conn = conn
	cclog.ComponentDebug("CCControlClient", "Established connection to", addr)
	return nil
}

func (c *ccControlClient) sendRequestAndCheckReply(request lp.CCMessage) (value, level string, err error) {
	resp, err := c.conn.Request(c.natsCfg.RequestSubject, []byte(request.ToLineProtocol(nil)), time.Second)
	if err != nil {
		err = fmt.Errorf("failed to request to subject %s: %w", c.natsCfg.RequestSubject, err)
		return
	}

	replyList, err := NatsReceive(resp)
	if err != nil {
		err = fmt.Errorf("NatsReceive failed: %w", err)
		return
	}

	if len(replyList) == 0 {
		err = fmt.Errorf("Received reply with no CCMessage")
		return
	}

	if len(replyList) > 1 {
		cclog.ComponentError("Received reply with more than one CCMessage:", replyList)
	}

	reply := replyList[0]
	if reply.Name() != request.Name() {
		err = fmt.Errorf("Received reply name '%s' mismatches expected 'controls': %v", reply.Name(), reply)
		return
	}

	requestHostname, _ := request.GetTag("hostname")
	replyHostname, ok := reply.GetTag("hostname")
	if !ok {
		err = fmt.Errorf("Received reply without hostname: %v", reply)
		return
	}
	if replyHostname != requestHostname {
		err = fmt.Errorf("Received reply hostname '%s' mismatches expected '%s': %v", replyHostname, requestHostname, reply)
		return
	}

	level, ok = reply.GetTag("level")
	if !ok {
		err = fmt.Errorf("Received reply without tag 'level': %v", reply)
		return
	}

	if !reply.IsLog() {
		err = fmt.Errorf("Received reply is not of type log: %s", reply)
		return
	}

	value = reply.GetLogValue()
	return // value, level, nil
}

func (c *ccControlClient) GetControls(hostname string) (*CCControlList, error) {
	tags := map[string]string{
		"hostname": hostname,
		"method":   "GET",
		"type":     "node",
		"type-id":  "0",
	}

	request, err := lp.NewGetControl("controls", tags, nil, time.Now())
	if err != nil {
		return nil, fmt.Errorf("Failed to create control message to '%s' to get controls: %w", hostname, err)
	}

	value, level, err := c.sendRequestAndCheckReply(request)
	if err != nil {
		return nil, fmt.Errorf("Request failed: %w", err)
	}

	if level == "INFO" {
		var outlist CCControlList
		err := json.Unmarshal([]byte(value), &outlist)
		return &outlist, err
	} else {
		return nil, fmt.Errorf("Getting controls from host '%s' failed: %s", hostname, value)
	}
}

func (c *ccControlClient) GetTopology(hostname string) (*CCControlTopology, error) {
	tags := map[string]string{
		"hostname": hostname,
		"method":   "GET",
		"type":     "node",
		"type-id":  "0",
	}

	request, err := lp.NewGetControl("topology", tags, nil, time.Now())
	if err != nil {
		return nil, fmt.Errorf("Failed to create control message to '%s' to get controls: %w", hostname, err)
	}

	value, level, err := c.sendRequestAndCheckReply(request)
	if err != nil {
		return nil, fmt.Errorf("Request failed: %w", err)
	}

	if level == "INFO" {
		var topo CCControlTopology
		err = json.Unmarshal([]byte(value), &topo)
		return &topo, err
	} else {
		return nil, fmt.Errorf("Getting topology from host '%s' failed: %s", hostname, value)
	}
}

func (c *ccControlClient) GetControlValue(hostname, control string, device string, deviceID string) (string, error) {
	tags := map[string]string{
		"hostname": hostname,
		"method":   "GET",
		"type":     device,
		"type-id":  deviceID,
	}

	request, err := lp.NewGetControl(control, tags, nil, time.Now())
	if err != nil {
		return "", fmt.Errorf("Failed to create message to '%s' to get controls: %w", hostname, err)
	}

	value, level, err := c.sendRequestAndCheckReply(request)
	if err != nil {
		return "", fmt.Errorf("Request failed: %w", err)
	}

	if level == "INFO" {
		return value, nil
	} else {
		return "", fmt.Errorf("Getting control '%s' from host '%s' failed: %s", control, hostname, value)
	}
}

func (c *ccControlClient) SetControlValue(hostname, control string, device string, deviceID string, value string) error {
	tags := map[string]string{
		"hostname": hostname,
		"method":   "PUT",
		"type":     device,
		"type-id":  deviceID,
	}

	request, err := lp.NewPutControl(control, tags, nil, value, time.Now())
	if err != nil {
		return fmt.Errorf("Failed to create control message to '%s' to set control: %w", hostname, err)
	}

	value, level, err := c.sendRequestAndCheckReply(request)
	if err != nil {
		return fmt.Errorf("Request failed: %w", err)
	}

	if level == "ERROR" {
		return fmt.Errorf("Setting control '%s' from host '%s' to value '%s' failed: %s", control, hostname, value, value)
	}

	return nil
}
