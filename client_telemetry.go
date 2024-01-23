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

type infoResponse struct {
	MultiConnMode bool                `json:"multi"`
	Clients       []MQTTClientInfo    `json:"clients,omitempty"`
	Subscriptions map[string]QOSLevel `json:"subscriptions,omitempty"`
}

func (c *Client) infoResponse() *infoResponse {
	subs := c.readSubscriptionMeta()
	ci := c.clientInfo()

	return &infoResponse{
		MultiConnMode: c.options.multiConnectionMode,
		Clients:       ci,
		Subscriptions: subs,
	}
}

func (c *Client) clientInfo() []MQTTClientInfo {
	if c.options.multiConnectionMode {
		return c.multiClientInfo()
	}

	return c.singleClientInfo()
}

func (c *Client) readSubscriptionMeta() map[string]QOSLevel {
	c.subMu.RLock()

	subs := make(map[string]QOSLevel, len(c.subscriptions))

	for topic, sub := range c.subscriptions {
		for _, opt := range sub.options {
			switch v := opt.(type) {
			case QOSLevel:
				subs[topic] = v
			}
		}

		// if no QOSLevel Option is provided, default to QOSZero
		if _, ok := subs[topic]; !ok {
			subs[topic] = QOSZero
		}
	}

	c.subMu.RUnlock()

	return subs
}

func (c *Client) multiClientInfo() []MQTTClientInfo {
	c.clientMu.RLock()

	if len(c.mqttClients) == 0 {
		return nil
	}

	bCh := make(chan MQTTClientInfo, len(c.mqttClients))

	_ = c.execute(func(cc mqtt.Client) error {
		bCh <- transformClientInfo(cc)

		return nil
	}, execAll)

	c.clientMu.RUnlock()

	close(bCh)

	bl := make([]MQTTClientInfo, 0, len(bCh))

	for b := range bCh {
		bl = append(bl, b)
	}

	sort.Slice(bl, func(i, j int) bool { return bl[i].ClientID < bl[j].ClientID })

	return bl
}

func (c *Client) singleClientInfo() []MQTTClientInfo {
	c.clientMu.RLock()
	defer c.clientMu.RUnlock()

	if c.mqttClient == nil {
		return nil
	}

	var bi MQTTClientInfo

	_ = c.execute(func(cc mqtt.Client) error {
		bi = transformClientInfo(cc)

		return nil
	}, execAll)

	return []MQTTClientInfo{bi}
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
