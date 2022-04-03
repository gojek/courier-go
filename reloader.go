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

	connErrCallback func(error)
	reloadCallback  func([]string, error)
}

// UseXDSReloader initializes an xdsClient that watches the specified watchEndpoint and reloads the provided courier client whenever an update is received
// for this endpoint. connErrCallback and reloadCallback are called when connection errors and config reloads are received.
func UseXDSReloader(config *bootstrap.ServerConfig, watchEndpoint string, xdsReloader XdsReloader) (*xdsclient.Client, error) {
	xdsClient, err := xdsclient.New(config, updatehandler.Config{
		ConnErrCallback: xdsReloader.connErr,
		Epw: []types.EndpointWatcher{
			{
				Endpoint: watchEndpoint,
				Callback: xdsReloader.clientReload,
			},
		},
	})

	return xdsClient, err
}

func (xds *XdsReloader) clientReload(addresses []string) {
	var (
		host string
		port uint16
		err  error
	)

	if xds.reloadCallback != nil {
		defer xds.reloadCallback(addresses, err)
	}
	//Use the first valid address from the list of addresses
	for _, address := range addresses {
		if host, port, err = getHostPort(address); err == nil {
			break
		}
	}

	if err != nil {
		return
	}

	addressOpts := WithTCPAddress(host, port)

	xds.opts = append(xds.opts, addressOpts)

	if err = xds.c.reloadConfig(xds.opts...); err != nil {
		log.Printf("Could not restart client for address: %v error: %v", addresses, err)
	}
}

func (xds *XdsReloader) connErr(err error) {
	if xds.connErrCallback != nil {
		xds.connErrCallback(err)
	}
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
