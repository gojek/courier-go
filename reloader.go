package courier

import (
	"github.com/gojekfarm/courier-go/xds/bootstrap"
	"github.com/gojekfarm/courier-go/xds/types"
	"github.com/gojekfarm/courier-go/xds/updatehandler"
	"github.com/gojekfarm/courier-go/xds/xdsclient"
	"log"
	"net"
	"strconv"
)

type XdsReloader struct {
	c    *Client
	opts []ClientOption
}

func UseXDSReloader(config *bootstrap.ServerConfig, watchEndpoint string, c *Client, initClientOpts ...ClientOption) error {
	reloader := XdsReloader{
		c:    c,
		opts: initClientOpts,
	}

	_, err := xdsclient.New(config, updatehandler.Config{
		ConnErrCallback: reloader.ConnErrCallback,
		Epw: []types.EndpointWatcher{
			{
				Endpoint: watchEndpoint,
				Callback: reloader.ClientReload,
			},
		},
	})

	return err
}

func (xds *XdsReloader) ClientReload(addresses []string) {
	var (
		host string
		port uint16
		err  error
	)

	//Use the first valid address from the list of addresses
	for _, address := range addresses {
		if host, port, err = getHostPort(address); err == nil {
			break
		}
	}

	if err != nil {
		log.Printf("No valid address obtained: %v", addresses)
		return
	}

	addressOpts := WithTCPAddress(host, port)

	xds.opts = append(xds.opts, addressOpts)

	if err = xds.c.reloadConfig(xds.opts...); err != nil {
		log.Printf("Could not restart client for address: %v error: %v", addresses, err)
	}
}

func (xds *XdsReloader) ConnErrCallback(err error) {
	log.Printf("Connection error from xds client: %v", err)
}

func getHostPort(address string) (string, uint16, error) {
	host, portString, err := net.SplitHostPort(address)
	if err == nil {
		p, err := strconv.ParseUint(portString, 10, 16)
		if err != nil {
			return "", 0, err
		}
		port := uint16(p)

		return host, port, nil
	}

	return "", 0, err
}
