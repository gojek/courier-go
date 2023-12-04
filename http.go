package courier

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/gojekfarm/xtools/generic/slice"
)

type brokerList []broker

type broker struct {
	Addresses     []TCPAddress `json:"addresses"`
	ClientID      string       `json:"client_id"`
	Username      string       `json:"username"`
	ResumeSubs    bool         `json:"resume_subs"`
	CleanSession  bool         `json:"clean_session"`
	AutoReconnect bool         `json:"auto_reconnect"`
	Connected     bool         `json:"connected"`
}

// TelemetryHandler returns a http.Handler that exposes the connected brokers information
func (c *Client) TelemetryHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !c.options.multiConnectionMode {
			var bi broker

			err := c.execute(func(cc mqtt.Client) error {
				bi = brokerInfo(cc)

				return nil
			}, execAll)

			writeResponse(w, brokerList{bi}, err, false)

			return
		}

		var bl brokerList
		bCh := make(chan broker, len(c.mqttClients))

		err := c.execute(func(cc mqtt.Client) error {
			bCh <- brokerInfo(cc)

			return nil
		}, execAll)

		close(bCh)

		for b := range bCh {
			bl = append(bl, b)
		}

		sort.Slice(bl, func(i, j int) bool { return bl[i].ClientID < bl[j].ClientID })

		writeResponse(w, bl, err, true)
	})
}

func writeResponse(w http.ResponseWriter, list brokerList, err error, multi bool) {
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, `{"error": "%s"}`, err.Error())

		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"multi":   multi,
		"brokers": list,
	})
}

func brokerInfo(cc mqtt.Client) broker {
	opts := cc.OptionsReader()

	return broker{
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
