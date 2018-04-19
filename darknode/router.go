package darknode

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/republicprotocol/republic-go/logger"

	"github.com/republicprotocol/republic-go/dispatch"
	"google.golang.org/grpc"

	"github.com/republicprotocol/republic-go/identity"
	"github.com/republicprotocol/republic-go/order"
	"github.com/republicprotocol/republic-go/rpc"
)

type Router struct {
	maxConnections int
	address        identity.Address
	multiAddress   identity.MultiAddress

	orderFragmentSplitterCh chan<- order.Fragment
	orderFragmentSplitter   *dispatch.Splitter

	mu               *sync.Mutex
	computeSenders   map[identity.Address]chan *rpc.Computation
	computeReceivers map[identity.Address]*dispatch.Splitter
	computeErrs      map[identity.Address]*dispatch.Splitter
	computeArcs      map[identity.Address]int

	clientPool *rpc.ClientPool
	relayer    rpc.RelayService
	smpcer     rpc.ComputerService
}

func NewRouter(maxConnections int, multiAddress identity.MultiAddress) *Router {
	router := &Router{
		maxConnections: maxConnections,
		address:        multiAddress.Address(),
		multiAddress:   multiAddress,

		orderFragmentSplitterCh: make(chan<- order.Fragment),
		orderFragmentSplitter:   dispatch.NewSplitter(maxConnections),

		mu:               new(sync.Mutex),
		computeSenders:   map[identity.Address]chan *rpc.Computation{},
		computeReceivers: map[identity.Address]*dispatch.Splitter{},
		computeErrs:      map[identity.Address]*dispatch.Splitter{},
		computeArcs:      map[identity.Address]int{},
	}
	router.clientPool = rpc.NewClientPool(multiAddress)
	router.relayer = rpc.NewRelayService(rpc.Options{}, router, logger.StdoutLogger)
	router.smpcer = rpc.NewComputerService()
	return router
}

func (router *Router) Run(done <-chan struct{}, host, port string) <-chan error {
	errs := make(chan error, 1)

	go func() {
		defer close(errs)
		server := grpc.NewServer()
		router.relayer.Register(server)
		router.smpcer.Register(server)
		listener, err := net.Listen("tcp", fmt.Sprintf("%v:%v", host, port))
		if err != nil {
			errs <- err
			return
		}

		if err := server.Serve(listener); err != nil {
			errs <- err
			return
		}
	}()

	return errs
}

func (router *Router) OrderFragments(done <-chan struct{}) (<-chan order.Fragment, <-chan error) {
	orderFragmentReceiver := make(chan order.Fragment)
	errs := make(chan error, 1)

	go func() {
		defer close(orderFragmentReceiver)
		defer close(errs)

		if err := router.orderFragmentSplitter.Subscribe(orderFragmentReceiver); err != nil {
			errs <- err
			return
		}

		<-done
		router.orderFragmentSplitter.Unsubscribe(orderFragmentReceiver)
	}()

	return orderFragmentReceiver, errs
}

func (router *Router) Compute(done <-chan struct{}, addr identity.Address, computationSender <-chan *rpc.Computation) (<-chan *rpc.Computation, <-chan error) {
	computationReceiver := make(chan *rpc.Computation)
	errs := make(chan error)

	go func() {
		defer close(computationReceiver)
		defer close(errs)

		router.mu.Lock()
		if _, ok := router.computeReceivers[addr]; !ok {
			router.setupCompute(addr)
		}
		router.computeArcs[addr]++
		router.mu.Unlock()

		router.computeReceivers[addr].Subscribe(computationReceiver)
		router.computeErrs[addr].Subscribe(errs)

		defer func() {

			router.computeReceivers[addr].Unsubscribe(computationReceiver)
			router.computeErrs[addr].Unsubscribe(errs)

			router.mu.Lock()
			if router.computeArcs[addr]--; router.computeArcs[addr] == 0 {
				router.teardownCompute(addr)
			}
			router.mu.Unlock()
		}()

		for {
			select {
			case <-done:
				return
			case computation, ok := <-computationSender:
				if !ok {
					return
				}
				select {
				case <-done:
				case router.computeSenders[addr] <- computation:
				}
			}
		}
	}()

	return computationReceiver, errs
}

func (router *Router) setupCompute(addr identity.Address) {

	var receiver <-chan *rpc.Computation
	var errs <-chan error

	sender := make(chan *rpc.Computation)
	if bytes.Compare(router.address.ID()[:], addr.ID()[:]) < 0 {
		receiver, errs = router.clientPool.Compute(context.Background(), addr, sender)
		sender <- &rpc.Computation{MultiAddress: rpc.MarshalMultiAddress(&router.multiAddress)}
	} else {
		receiver, errs = router.smpcer.WaitForCompute(addr, sender)
	}

	router.computeSenders[addr] = sender
	router.computeReceivers[addr] = dispatch.NewSplitter(router.maxConnections)
	router.computeReceivers[addr].Split(receiver)
	router.computeErrs[addr] = dispatch.NewSplitter(router.maxConnections)
	router.computeErrs[addr].Split(errs)
}

func (router *Router) teardownCompute(addr identity.Address) {
	close(router.computeSenders[addr])
	delete(router.computeSenders, addr)
	delete(router.computeReceivers, addr)
	delete(router.computeErrs, addr)
}

func (router *Router) OnOpenOrder(from identity.MultiAddress, orderFragment *order.Fragment) {
	router.orderFragmentSplitterCh <- *orderFragment
}
