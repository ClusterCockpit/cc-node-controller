package cccontrolclient

// TODO currently disfunctional, requires more debugging
// -topology fails and outputs zeroes only

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	lp "github.com/ClusterCockpit/cc-energy-manager/pkg/cc-message"
	cclog "github.com/ClusterCockpit/cc-metric-collector/pkg/ccLogger"
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
	GetControls(hostname string) (CCControlList, error)
	GetTopology(hostname string) (CCControlTopology, error)
	GetControlValue(hostname, control string, device string, deviceID string) (string, error)
	SetControlValue(hostname, control string, device string, deviceID string, value string) error
	Close()
}

type NatsConfig struct {
	Server string
	Port uint16
	InputSubject string
	OutputSubject string
	User string
	Password string
	CredsFile string
	NKeySeedFile string
}

func NewCCControlClient(natsConfig NatsConfig) (CCControlClient, error) {
	n := new(ccControlClient)
	err := n.Init(natsConfig)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func NatsReceive(m *nats.Msg) []lp.CCMessage {
	out, err := lp.FromBytes(m.Data)
	if err != nil {
		return nil
	}
	return out
}

func (c *ccControlClient) Init(natsCfg NatsConfig) error {

	h, err := os.Hostname()
	if err != nil {
		return errors.New("failed to get hostname for CCControlClient")
	}

	c.natsCfg = natsCfg
	c.hostname = h
	c.conn = nil
	return nil
}

func (c *ccControlClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
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

func (c *ccControlClient) GetControls(hostname string) (CCControlList, error) {
	// var wg sync.WaitGroup
	var globerr error = nil
	var outlist CCControlList = CCControlList{}
	if c.conn == nil {
		err := c.connect()
		if err != nil {
			return outlist, err
		}
	}
	tags := map[string]string{
		"hostname": hostname,
		"method":   "GET",
		"type":     "node",
		"type-id":  "0",
	}
	name := "controls"
	out, err := lp.NewGetControl(name, tags, map[string]string{}, time.Now())
	if err != nil {
		return outlist, fmt.Errorf("failed to create control message to %s to get controls", hostname)
	}

	// mysubject := fmt.Sprintf("%s.%s", c.input_subject, hostname)

	// _, err = c.conn.Subscribe(mysubject, func(msg *nats.Msg) {
	// 	wg.Add(1)
	// 	mlist := NatsReceive(msg)
	// 	cclog.ComponentDebug("CCControlClient", mlist)
	// metrics_loop:
	// 	for _, m := range mlist {
	// 		if m.Name() == name {
	// 			if testhost, ok := m.GetTag("hostname"); ok {
	// 				if testhost == hostname {
	// 					if level, ok := m.GetTag("level"); ok {
	// 						if value, ok := m.GetField("value"); ok {
	// 							if level == "INFO" {
	// 								switch x := value.(type) {
	// 								case string:
	// 									globerr = json.Unmarshal([]byte(x), &outlist)
	// 								case []byte:
	// 									globerr = json.Unmarshal(x, &outlist)
	// 								}
	// 							} else {
	// 								cclog.ComponentError("CCControlClient", "Host", hostname, ":", value)
	// 								switch x := value.(type) {
	// 								case string:
	// 									globerr = errors.New(x)
	// 								case []byte:
	// 									globerr = errors.New(string(x))
	// 								}

	// 							}
	// 							break metrics_loop
	// 						}
	// 					}
	// 				}
	// 			}
	// 		}
	// 	}
	// 	wg.Done()
	// })

	cclog.ComponentDebug("CCControlClient", "Publishing to", c.natsCfg.OutputSubject, ":", out.String())
	//c.conn.Publish(c.natsCfg.OutputSubject, []byte(out.ToLineProtocol(map[string]bool{})))
	resp, err := c.conn.Request(c.natsCfg.OutputSubject, []byte(out.ToLineProtocol(map[string]bool{})), time.Second)
	if err != nil {
		return outlist, fmt.Errorf("failed to request to subject %s: %v", c.natsCfg.OutputSubject, err.Error())
	}
	mlist := NatsReceive(resp)
	if len(mlist) == 0 {
		return outlist, fmt.Errorf("failed to receive response to subject %s", c.natsCfg.OutputSubject)
	}
	m := mlist[0]
	if m.Name() == name {
		if testhost, ok := m.GetTag("hostname"); ok && testhost == hostname {
			if level, ok := m.GetTag("level"); ok {
				if value, ok := m.GetField("value"); ok {
					if level == "INFO" {
						switch x := value.(type) {
						case string:
							globerr = json.Unmarshal([]byte(x), &outlist)
						case []byte:
							globerr = json.Unmarshal(x, &outlist)
						}
					} else {
						cclog.ComponentError("CCControlClient", "Host", hostname, ":", value)
						switch x := value.(type) {
						case string:
							globerr = errors.New(x)
						case []byte:
							globerr = errors.New(string(x))
						}

					}
				}
			}
		}
	}
	return outlist, globerr
}

func (c *ccControlClient) GetTopology(hostname string) (CCControlTopology, error) {
	var topo CCControlTopology = CCControlTopology{}
	var globerr error = nil
	if c.conn == nil {
		err := c.connect()
		if err != nil {
			return topo, err
		}
	}
	tags := map[string]string{
		"hostname": hostname,
		"method":   "GET",
		"type":     "node",
		"type-id":  "0",
	}
	name := "topology"
	out, err := lp.NewGetControl(name, tags, map[string]string{}, time.Now())
	if err != nil {
		return topo, fmt.Errorf("failed to create control message to %s to get controls", hostname)
	}

	cclog.ComponentDebug("CCControlClient", "Publishing to", c.natsCfg.OutputSubject, ":", out.String())
	//c.conn.Publish(c.natsCfg.OutputSubject, []byte(out.ToLineProtocol(map[string]bool{})))
	resp, err := c.conn.Request(c.natsCfg.OutputSubject, []byte(out.ToLineProtocol(map[string]bool{})), time.Second)
	if err != nil {
		return topo, fmt.Errorf("failed to request to subject %s: %v", c.natsCfg.OutputSubject, err.Error())
	}
	mlist := NatsReceive(resp)
	if len(mlist) == 0 {
		return topo, fmt.Errorf("failed to receive response to subject %s", c.natsCfg.OutputSubject)
	}
	m := mlist[0]
	if m.Name() != name {
		return topo, fmt.Errorf("unexpected name received: %s (expected: %s)", m.Name(), name)
	}
	if testhost, ok := m.GetTag("hostname"); !ok || testhost != hostname {
		return topo, fmt.Errorf("failed to retrieve hostname or mismatched hostname: %s (expected %s, success %v)", testhost, hostname, ok)
	}
	level, ok := m.GetTag("level")
	if !ok {
		return topo, fmt.Errorf("failed to get level")
	}
	value, ok := m.GetField("value")
	if !ok {
		return topo, fmt.Errorf("failed to get value")
	}
	fmt.Println(m.String())
	if level == "INFO" {
		switch x := value.(type) {
		case string:
			globerr = json.Unmarshal([]byte(x), &topo)
		case []byte:
			globerr = json.Unmarshal(x, &topo)
		}
	} else {
		cclog.ComponentError("CCControlClient", "Host", hostname, ":", value)
		switch x := value.(type) {
		case string:
			globerr = errors.New(x)
		case []byte:
			globerr = errors.New(string(x))
		}
	}
	return topo, globerr
}

func (c *ccControlClient) GetControlValue(hostname, control string, device string, deviceID string) (string, error) {
	// var wg sync.WaitGroup
	var outstring string = ""
	var globerr error = nil
	if c.conn == nil {
		err := c.connect()
		if err != nil {
			return outstring, err
		}
	}
	tags := map[string]string{
		"hostname": hostname,
		"method":   "GET",
		"type":     device,
		"type-id":  deviceID,
	}
	name := control
	out, err := lp.NewGetControl(name, tags, map[string]string{}, time.Now())
	if err != nil {
		return outstring, fmt.Errorf("failed to create control message to %s to get controls", hostname)
	}

	cclog.ComponentDebug("CCControlClient", "Publishing to", c.natsCfg.OutputSubject, ":", out.String())
	//c.conn.Publish(c.natsCfg.OutputSubject, []byte(out.ToLineProtocol(map[string]bool{})))
	resp, err := c.conn.Request(c.natsCfg.OutputSubject, []byte(out.ToLineProtocol(map[string]bool{})), time.Second)
	if err != nil {
		return outstring, fmt.Errorf("failed to request to subject %s: %v", c.natsCfg.OutputSubject, err.Error())
	}
	mlist := NatsReceive(resp)
	if len(mlist) == 0 {
		return outstring, fmt.Errorf("failed to receive response to subject %s", c.natsCfg.OutputSubject)
	}
	m := mlist[0]
	if m.Name() == name {
		if testhost, ok := m.GetTag("hostname"); ok && testhost == hostname {
			if level, ok := m.GetTag("level"); ok {
				if value, ok := m.GetField("value"); ok {
					if level == "INFO" {
						switch x := value.(type) {
						case string:
							outstring = x
						case []byte:
							outstring = string(x)
						default:
							outstring = fmt.Sprintf("%v", x)
						}
					} else {
						cclog.ComponentError("CCControlClient", "Host", hostname, ":", value)
						switch x := value.(type) {
						case string:
							globerr = errors.New(x)
						case []byte:
							globerr = errors.New(string(x))
						}

					}
				}
			}
		}
	}

	return outstring, globerr
}

func (c *ccControlClient) SetControlValue(hostname, control string, device string, deviceID string, value string) error {
	var globerr error = nil
	if c.conn == nil {
		err := c.connect()
		if err != nil {
			return err
		}
	}
	tags := map[string]string{
		"hostname": hostname,
		"method":   "PUT",
		"type":     device,
		"type-id":  deviceID,
	}
	name := control
	out, err := lp.NewPutControl(name, tags, map[string]string{}, value, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create control message to %s to get controls", hostname)
	}

	cclog.ComponentDebug("CCControlClient", "Publishing to", c.natsCfg.OutputSubject, ":", out.String())
	//c.conn.Publish(c.natsCfg.OutputSubject, []byte(out.ToLineProtocol(map[string]bool{})))
	resp, err := c.conn.Request(c.natsCfg.OutputSubject, []byte(out.ToLineProtocol(map[string]bool{})), time.Second)
	if err != nil {
		return fmt.Errorf("failed to request to subject %s: %v", c.natsCfg.OutputSubject, err.Error())
	}
	mlist := NatsReceive(resp)
	if len(mlist) == 0 {
		return fmt.Errorf("failed to receive response to subject %s", c.natsCfg.OutputSubject)
	}
	m := mlist[0]
	if m.Name() == name {
		if testhost, ok := m.GetTag("hostname"); ok && testhost == hostname {
			if level, ok := m.GetTag("level"); ok {
				if value, ok := m.GetField("value"); ok {
					if level == "ERROR" {
						cclog.ComponentError("CCControlClient", "Host", hostname, ":", value)
						switch x := value.(type) {
						case string:
							globerr = errors.New(x)
						case []byte:
							globerr = errors.New(string(x))
						}

					}
				}
			}
		}
	}
	return globerr
}
