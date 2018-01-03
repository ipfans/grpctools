package consul

import (
	"net"
	"os"
	"strconv"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/naming"
)

// Resolver implements the gRPC Resolver interface using a Consul backend.
// Resolver also implements Watcher interface.
type Resolver struct {
	c           *api.Client
	service     string
	tag         string
	logger      grpclog.LoggerV2
	passingOnly bool

	chanQuit    chan struct{}
	chanUpdates chan []*naming.Update
}

// Option for Resolver instance.
type Option func(r *Resolver)

// WithTag to set tag filter for consul.
func WithTag(tag string) Option {
	return func(r *Resolver) {
		r.tag = tag
	}
}

// WithLogger replaced built-in logger to given.
func WithLogger(logger grpclog.LoggerV2) Option {
	return func(r *Resolver) {
		r.logger = logger
	}
}

// NewConsulResolver initializes and returns a new Resolver.
func NewConsulResolver(client *api.Client, service string, opts ...Option) (*Resolver, error) {
	r := &Resolver{
		c:           client,
		service:     service,
		tag:         "",
		logger:      grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr),
		passingOnly: true,
		chanQuit:    make(chan struct{}),
		chanUpdates: make(chan []*naming.Update, 1),
	}

	for _, o := range opts {
		o(r)
	}

	// Retrieve instances immediately
	instances, index, err := r.getInstances(0)
	if err != nil {
		r.logger.Infof("naming/consul: error retrieving instances from Consul: %v\n", err)
	}
	updates := r.makeUpdates(nil, instances)
	if len(updates) > 0 {
		r.chanUpdates <- updates
	}

	// Start updater
	go r.backgroundUpdater(instances, index)

	return r, nil
}

// Resolve also a watcher for target.
func (r *Resolver) Resolve(target string) (naming.Watcher, error) {
	return r, nil
}

// Next blocks until an update or error happens. It may return one or more
// updates. The first call will return the full set of instances available
// as NewResolver will look those up. Subsequent calls to Next() will
// block until the resolver finds any new or removed instance.
//
// An error is returned if and only if the watcher cannot recover.
func (r *Resolver) Next() ([]*naming.Update, error) {
	return <-r.chanUpdates, nil
}

// Close closes the watcher.
func (r *Resolver) Close() {
	select {
	case <-r.chanQuit:
	default:
		close(r.chanQuit)
		close(r.chanUpdates)
	}
}

// backgroundUpdater is a background process started in NewResolver. It takes
// a list of previously resolved instances (in the format of host:port, e.g.
// 192.168.0.1:1234) and the last index returned from Consul.
func (r *Resolver) backgroundUpdater(instances []string, lastIndex uint64) {
	var err error
	var oldInstances = instances
	var newInstances []string

	// TODO Cache the updates for a while, so that we don't overwhelm Consul.
	for {
		select {
		case <-r.chanQuit:
			break
		default:
			newInstances, lastIndex, err = r.getInstances(lastIndex)
			if err != nil {
				r.logger.Infof("naming/consul: error retrieving instances from Consul: %v\n", err)
				continue
			}
			updates := r.makeUpdates(oldInstances, newInstances)
			if len(updates) > 0 {
				r.chanUpdates <- updates
			}
			oldInstances = newInstances
		}
	}
}

// getInstances retrieves the new set of instances registered for the
// service from Consul.
func (r *Resolver) getInstances(lastIndex uint64) ([]string, uint64, error) {
	services, meta, err := r.c.Health().Service(r.service, r.tag, r.passingOnly, &api.QueryOptions{
		WaitIndex: lastIndex,
	})
	if err != nil {
		return nil, lastIndex, err
	}

	var instances []string
	for _, service := range services {
		s := service.Service.Address
		if len(s) == 0 {
			s = service.Node.Address
		}
		addr := net.JoinHostPort(s, strconv.Itoa(service.Service.Port))
		instances = append(instances, addr)
	}
	return instances, meta.LastIndex, nil
}

// makeUpdates calculates the difference between and old and a new set of
// instances and turns it into an array of naming.Updates.
func (r *Resolver) makeUpdates(oldInstances, newInstances []string) []*naming.Update {
	oldAddr := make(map[string]struct{}, len(oldInstances))
	for _, instance := range oldInstances {
		oldAddr[instance] = struct{}{}
	}
	newAddr := make(map[string]struct{}, len(newInstances))
	for _, instance := range newInstances {
		newAddr[instance] = struct{}{}
	}

	var updates []*naming.Update
	for addr := range newAddr {
		if _, ok := oldAddr[addr]; !ok {
			updates = append(updates, &naming.Update{Op: naming.Add, Addr: addr})
		}
	}
	for addr := range oldAddr {
		if _, ok := newAddr[addr]; !ok {
			updates = append(updates, &naming.Update{Op: naming.Delete, Addr: addr})
		}
	}

	return updates
}
