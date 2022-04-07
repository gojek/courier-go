package xdsclient
//
//import (
//	"context"
//	"errors"
//	"fmt"
//	"github.com/gojekfarm/courier-go/xds"
//	"github.com/gojekfarm/courier-go/xds/backoff"
//	"github.com/gojekfarm/courier-go/xds/updatehandler"
//	"google.golang.org/grpc/credentials/insecure"
//	"log"
//	"sync"
//	"time"
//
//	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
//	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
//	"github.com/golang/protobuf/proto"
//	"google.golang.org/grpc"
//	"google.golang.org/grpc/keepalive"
//
//	"github.com/gojekfarm/courier-go/xds/bootstrap"
//)
//
//// ToDo: Make this implement resolver. xds:///<customer-broker>
//
//type Client struct {
//	config *bootstrap.ServerConfig
//
//	cc     *grpc.ClientConn
//	client xds.Client
//
//	backoff func(int) time.Duration
//
//	mu sync.Mutex
//
//	stopRunGoroutine context.CancelFunc
//
//	updateHandler updatehandler.UpdateHandler
//
//	//ToDo: should have a single endpoint
//	endpoints map[string]struct{}
//	// version contains the version that was acked (the version in the ack
//	// request that was sent on wire).
//	version string
//	// nonce contains the nonce from the most recent received response.
//	nonce string
//}
//
//// New creates a new client with the provided config and starts a subscription for
//// resources defined in endpoints.
//func New(config *bootstrap.ServerConfig, updateHandlerCfg updatehandler.Config) (xdsClient *Client, retErr error) {
//	switch {
//	case config == nil:
//		return nil, errors.New("xds: no xds_server provided")
//	case config.ServerURI == "":
//		return nil, errors.New("xds: no xds_server name provided in options")
//	case config.NodeProto == nil:
//		return nil, errors.New("xds: no node_proto provided in options")
//	}
//
//	ret := &Client{
//		config:        config,
//		backoff:       backoff.DefaultExponential.Backoff,
//		endpoints:     endpointWatcherSliceToMap(updateHandlerCfg.Epw),
//		updateHandler: updatehandler.New(updateHandlerCfg),
//	}
//
//	defer func() {
//		if retErr != nil {
//			ret.Close()
//		}
//	}()
//
//	ctx, cancel := context.WithCancel(context.Background())
//	ret.stopRunGoroutine = cancel
//
//	dopts := []grpc.DialOption{
//		grpc.WithTransportCredentials(insecure.NewCredentials()),
//		grpc.WithKeepaliveParams(keepalive.ClientParameters{
//			Time:    5 * time.Minute,
//			Timeout: 20 * time.Second,
//		}),
//	}
//	cc, err := grpc.Dial(config.ServerURI, dopts...)
//	if err != nil {
//		// An error from a non-blocking dial indicates something serious.
//		return nil, fmt.Errorf("xds: failed to dial control plane {%s}: %v", config.ServerURI, err)
//	}
//	ret.cc = cc
//
//	if ret.client, err = xds.NewClient(ctx, config.NodeProto.(*v3corepb.Node), cc); err != nil {
//		if err != nil {
//			ret.updateHandler.NewConnectionError(err)
//			return nil, fmt.Errorf("xds: failed to create EDS stream {%s}: %v", config.ServerURI, err)
//		}
//	}
//
//	//Send subscriptions
//	if err = ret.startStream(ctx); err != nil {
//		return nil, fmt.Errorf("xds: failed to send initial EDS subscription {%s}: %v", config.ServerURI, err)
//	}
//
//	go ret.run(ctx)
//
//	return ret, nil
//}
//
//func (c *Client) Close() error {
//	// Note that Close needs to check for nils even if some of them are always
//	// set in the constructor. This is because the constructor defers Close() in
//	// error cases, and the fields might not be set when the error happens.
//	if c.stopRunGoroutine != nil {
//		c.stopRunGoroutine()
//	}
//	if c.cc != nil {
//		return c.cc.Close()
//	}
//	return nil
//}
//
//// run starts an ADS stream (and backs off exponentially, if the previous
//// stream failed without receiving a single reply) and runs the sender and
//// receiver routines to send and receive data from the stream respectively.
//func (c *Client) run(ctx context.Context) {
//	retries := 0
//	streamRunning := true
//
//	for {
//		select {
//		case <-ctx.Done():
//			return
//		default:
//		}
//
//		if retries != 0 {
//			timer := time.NewTimer(c.backoff(retries))
//			select {
//			case <-timer.C:
//			case <-ctx.Done():
//				if !timer.Stop() {
//					<-timer.C
//				}
//				return
//			}
//		}
//
//		retries++
//
//		//restart if xds stream if not already running
//		if !streamRunning {
//			if err := c.startStream(ctx); err != nil {
//				//ToDo: Send metrics here and logger here
//				log.Printf("xds: Error recovering from broken stream: %v", err)
//				streamRunning = false
//				continue
//			}
//			streamRunning = true
//		}
//
//		if c.recv() {
//			//On successful return from recv, reset retries to 0
//			retries = 0
//		}
//	}
//}
//
//// sends xDS requests to the management server and is used to create subscription,
//// unsubscription, nack and ack requests.
//func (c *Client) send(errMsg string) {
//	target := mapToSlice(c.endpoints)
//	if err := c.client.SendRequest(target, c.version, c.nonce, errMsg); err != nil {
//		log.Printf("xds: EDS send failed: %v", err)
//	}
//}
//
//// startStream sends out xDS requests for registered endpoints when recovering
//// from a broken stream.
//func (c *Client) startStream(ctx context.Context) error {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//
//	if err := c.client.RestartEDSClient(ctx, c.cc); err != nil {
//		c.updateHandler.NewConnectionError(err)
//		log.Printf("xds: ADS stream creation failed: %v", err)
//		return err
//	}
//
//	// Reset the ack versions when the stream restarts.
//	c.version, c.nonce = "", ""
//
//	if err := c.client.SendRequest(mapToSlice(c.endpoints), "", "", ""); err != nil {
//		log.Printf("ADS request failed: %v", err)
//		return err
//	}
//
//	return nil
//}
//
//// recv receives xDS responses on the provided ADS stream and branches out to
//// message specific handlers.
////Returns when:
////error is received when any connection error occurs
//func (c *Client) recv() bool {
//	success := false
//	for {
//		resp, err := c.client.Receive()
//		if err != nil {
//			c.updateHandler.NewConnectionError(err)
//			log.Printf("ADS stream is closed with error: %v", err)
//			return success
//		}
//
//		version, nonce, err := c.handleResponse(resp)
//		if err != nil {
//			log.Printf("Sending NACK for version: %v, nonce: %v, reason: %v", version, nonce, err)
//			c.send(err.Error())
//			continue
//		}
//
//		c.version, c.nonce = version, nonce
//		c.send("")
//		log.Printf("Sending ACK for version: %v, nonce: %v", version, nonce)
//		success = true
//	}
//}
//
////handleResponse calls update handlers and based on the error returned ack/nack is called by the caller
//func (c *Client) handleResponse(resp proto.Message) (string, string, error) {
//	resources, version, nonce, err := c.client.ParseResponse(resp)
//	if err != nil {
//		return version, nonce, err
//	}
//
//	addressMap := make(map[string][]string)
//
//	for _, r := range resources {
//		cla := &v3endpointpb.ClusterLoadAssignment{}
//		if err = proto.Unmarshal(r.GetValue(), cla); err != nil {
//			return "", "", fmt.Errorf("failed to unmarshal resource: %v", err)
//		}
//		log.Printf("Resource with name: %v, type: %T, contains: %v", cla.GetClusterName(), cla, cla)
//		//Return map of cluster names and endpoints
//		addressMap[cla.GetClusterName()] = []string{}
//		for _, localityLbEndpoints := range cla.Endpoints {
//			addressMap[cla.GetClusterName()] = append(addressMap[cla.GetClusterName()], parseEndpoints(localityLbEndpoints.LbEndpoints)...)
//		}
//	}
//
//	c.updateHandler.NewEndpoints(addressMap)
//
//	return version, nonce, err
//}
