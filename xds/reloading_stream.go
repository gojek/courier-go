package xds

import (
	"context"
	"sync"
	"time"

	v3discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc"

	"github.com/gojek/courier-go/xds/backoff"
	"github.com/gojek/courier-go/xds/log"
)

type adsStream v3discoverypb.AggregatedDiscoveryService_StreamAggregatedResourcesClient

type reloadingStream struct {
	mu       sync.RWMutex
	reloadWG sync.WaitGroup
	log      log.Logger
	s        adsStream
	ns       func() v3discoverypb.AggregatedDiscoveryServiceClient

	strategy    backoff.Strategy
	onReConnect func(error)
	reloadCh    chan error
	connTimeout time.Duration
}

func (r *reloadingStream) Send(req *v3discoverypb.DiscoveryRequest) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.s.Send(req)
}

func (r *reloadingStream) Recv() (*v3discoverypb.DiscoveryResponse, error) {
	resp, err := r.safeRead()
	if err == nil {
		return resp, nil
	}

	r.reloadWG.Add(1)
	r.reloadCh <- err
	r.reloadWG.Wait()

	return r.safeRead()
}

func (r *reloadingStream) safeRead() (*v3discoverypb.DiscoveryResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.s.Recv()
}

func (r *reloadingStream) createStream(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream, err := r.ns().StreamAggregatedResources(ctx, grpc.WaitForReady(true))
	if err != nil {
		r.log.Error(err, "xds: ReloadingStream could not create stream")

		return err
	}

	r.s = stream

	return nil
}

func (r *reloadingStream) startReloader(ctx context.Context) {
	retries := 0

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-r.reloadCh:
			retries++

			if err := r.createStream(ctx); err != nil {
				go func() {
					select {
					case <-ctx.Done():
						r.reloadWG.Done()

						return
					case <-time.After(r.strategy.Backoff(retries)):
						r.reloadCh <- err
					}
				}()

				continue
			}

			r.reloadWG.Done()

			if r.onReConnect != nil {
				r.onReConnect(err)
			}

			retries = 0
		}
	}
}
