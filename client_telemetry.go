package courier

import (
	"net/url"
	"sort"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/gojekfarm/xtools/generic/slice"
)

// MQTTClientInfo contains information about the internal MQTT client
type MQTTClientInfo struct {
	Addresses     []TCPAddress `json:"addresses"`
	ClientID      string       `json:"client_id"`
	Username      string       `json:"username"`
	ResumeSubs    bool         `json:"resume_subs"`
	CleanSession  bool         `json:"clean_session"`
	AutoReconnect bool         `json:"auto_reconnect"`
	Connected     bool         `json:"connected"`
}

type clientIntoList []MQTTClientInfo

func (c *Client) allClientInfo() clientIntoList {
	if c.options.multiConnectionMode {
		return c.multiClientInfo()
	}

	return c.singleClientInfo()
}

func (c *Client) multiClientInfo() clientIntoList {
	c.clientMu.RLock()
	bCh := make(chan MQTTClientInfo, len(c.mqttClients))
	c.clientMu.RUnlock()

	_ = c.execute(func(cc mqtt.Client) error {
		bCh <- transformClientInfo(cc)

		return nil
	}, execAll)

	close(bCh)

	bl := make(clientIntoList, 0, len(bCh))

	for b := range bCh {
		bl = append(bl, b)
	}

	sort.Slice(bl, func(i, j int) bool { return bl[i].ClientID < bl[j].ClientID })

	return bl
}

func (c *Client) singleClientInfo() clientIntoList {
	var bi MQTTClientInfo

	_ = c.execute(func(cc mqtt.Client) error {
		bi = transformClientInfo(cc)

		return nil
	}, execAll)

	return clientIntoList{bi}
}

func transformClientInfo(cc mqtt.Client) MQTTClientInfo {
	opts := cc.OptionsReader()

	return MQTTClientInfo{
		Addresses: slice.Map(opts.Servers(), func(u *url.URL) TCPAddress {
			i, _ := strconv.Atoi(u.Port())

			return TCPAddress{
				Host: u.Hostname(),
				Port: uint16(i),
			}
		}),
		ClientID:      opts.ClientID(),
		Username:      opts.Username(),
		ResumeSubs:    opts.ResumeSubs(),
		CleanSession:  opts.CleanSession(),
		AutoReconnect: opts.AutoReconnect(),
		Connected:     cc.IsConnected(),
	}
}
