package courier

import (
	"net/url"
	"sort"
	"strconv"

	mqtt "github.com/gojek/paho.mqtt.golang"
	"github.com/gojekfarm/xtools/generic/slice"
	"github.com/gojekfarm/xtools/generic/xmap"
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
	// Subscriptions contains the topics the client is subscribed to
	// Note: Currently, this field only holds shared subscriptions.
	Subscriptions []string `json:"subscriptions,omitempty"`
}

type infoResponse struct {
	MultiConnMode bool                `json:"multi"`
	PoolMode      bool                `json:"pool"`
	PoolSize      int                 `json:"pool_size,omitempty"`
	Clients       []MQTTClientInfo    `json:"clients,omitempty"`
	Subscriptions map[string]QOSLevel `json:"subscriptions,omitempty"`
}

func (c *Client) infoResponse() *infoResponse {
	subs := c.readSubscriptionMeta()
	ci := c.clientInfo()

	response := &infoResponse{
		MultiConnMode: c.options.multiConnectionMode,
		PoolMode:      c.options.poolEnabled,
		Clients:       ci,
		Subscriptions: subs,
	}

	if c.options.poolEnabled {
		response.PoolSize = c.options.poolSize
	}

	return response
}

func (c *Client) clientInfo() []MQTTClientInfo {
	if c.options.poolEnabled {
		return c.poolClientInfo()
	}

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

	cls := xmap.Values(c.mqttClients)

	if len(cls) == 0 {
		c.clientMu.RUnlock()

		return nil
	}

	bCh := make(chan MQTTClientInfo, len(cls))

	_ = c.execute(
		func(cc mqtt.Client) error { return nil },
		execOptWithState(func(f func(mqtt.Client) error, is *internalState) error {
			is.mu.Lock()
			subs := is.subsCalled.Values()
			is.mu.Unlock()

			bCh <- transformClientInfo(is.client, subs...)

			return f(is.client)
		}),
	)

	c.clientMu.RUnlock()

	close(bCh)

	bl := make([]MQTTClientInfo, 0, len(bCh))

	for b := range bCh {
		bl = append(bl, b)
	}

	sort.Slice(bl, func(i, j int) bool { return bl[i].ClientID < bl[j].ClientID })

	return bl
}

func (c *Client) poolClientInfo() []MQTTClientInfo {
	if len(c.connectionPool) == 0 {
		return nil
	}

	bl := make([]MQTTClientInfo, 0, len(c.connectionPool))

	_ = c.execute(func(cc mqtt.Client) error {
		bi := transformClientInfo(cc)
		bl = append(bl, bi)
		return nil
	}, execAll)

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

func transformClientInfo(cc mqtt.Client, subscribedTopics ...string) MQTTClientInfo {
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
		Connected:     cc.IsConnectionOpen(),
		Subscriptions: subscribedTopics,
	}
}
