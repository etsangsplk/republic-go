package network_test

import (
	"fmt"
	"sync"
	"time"

	"github.com/republicprotocol/go-identity"
	"github.com/republicprotocol/go-network"
	"github.com/republicprotocol/go-rpc"
)

type Topology int64

const (
	TopologyFull = 1
	TopologyStar = 2
	TopologyRing = 3
	TopologyLine = 4
)

const (
	DefaultOptionsDebug           = network.DebugOff
	DefaultOptionsAlpha           = 3
	DefaultOptiosnMaxBucketLength = 20
	NodePortBootstrap             = 3000
	NodePortSwarm                 = 4000
)

func GenerateTopology(topology Topology, numberOfNodes int, delegate network.Delegate) ([]*network.Node, map[identity.Address][]*network.Node, error) {
	var err error
	var nodes []*network.Node
	var routingTable map[identity.Address][]*network.Node

	switch topology {
	case TopologyFull:
		nodes, routingTable, err = GenerateFullTopology(numberOfNodes, delegate)
	case TopologyStar:
		nodes, routingTable, err = GenerateStarTopology(numberOfNodes, delegate)
	case TopologyLine:
		nodes, routingTable, err = GenerateLineTopology(numberOfNodes, delegate)
	case TopologyRing:
		nodes, routingTable, err = GenerateRingTopology(numberOfNodes, delegate)
	}
	return nodes, routingTable, err
}

func GenerateNodes(port, numberOfNodes int, delegate network.Delegate) ([]*network.Node, error) {
	nodes := make([]*network.Node, numberOfNodes)
	for i := 0 := range nodes {
		keyPair, err := identity.NewKeyPair()
		if err != nil {
			return nil, err
		}
		multiAddress, err := identity.NewMultiAddressFromString(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/republic/%s", port+i, keyPair.Address()))
		if err != nil {
			return nil, err
		}
		node := network.NewNode(
			delegate,
			network.Options{
				MultiAddress:    multiAddress,
				Debug:           DefaultOptionsDebug,
				Alpha:           DefaultOptionsAlpha,
				MaxBucketLength: DefaultOptiosnMaxBucketLength,
			},
		)
		nodes[i] = node
	}
	return nodes, nil
}

func GenerateFullTopology(numberOfNodes int, delegate network.Delegate) ([]*network.Node, map[identity.Address][]*network.Node, error) {
	nodes, err := GenerateNodes(NodePortBootstrap, numberOfNodes, delegate)
	if err != nil {
		return nil, nil, err
	}
	routingTable := map[identity.Address][]*network.Node{}
	for i, node := range nodes {
		routingTable[node.DHT.Address] = []*network.Node{}
		for j, peer := range nodes {
			if i == j {
				continue
			}
			routingTable[node.DHT.Address] = append(routingTable[node.DHT.Address], peer)
		}
	}
	return nodes, routingTable, nil
}

func generateStarTopology(numberOfNodes int, delegate network.Delegate) ([]*network.Node, map[identity.Address][]*network.Node, error) {
	nodes, err := generateNodes(numberOfNodes, delegate, BOOSTRAP_NODE_PORT)
	if err != nil {
		return nil, nil, err
	}
	topology := map[identity.Address][]*network.Node{}
	for i, node := range nodes {
		topology[node.DHT.Address] = []*network.Node{}
		if i == 0 {
			for j, peer := range nodes {
				if i == j {
					continue
				}
				topology[node.DHT.Address] = append(topology[node.DHT.Address], peer)
			}
		} else {
			topology[node.DHT.Address] = append(topology[node.DHT.Address], nodes[0])
		}
	}
	return nodes, topology, nil
}

func generateLineTopology(numberOfNodes int, delegate network.Delegate) ([]*network.Node, map[identity.Address][]*network.Node, error) {
	nodes, err := generateNodes(numberOfNodes, delegate, BOOSTRAP_NODE_PORT)
	if err != nil {
		return nil, nil, err
	}
	topology := map[identity.Address][]*network.Node{}
	for i, node := range nodes {
		topology[node.DHT.Address] = []*network.Node{}
		if i == 0 {
			topology[node.DHT.Address] = append(topology[node.DHT.Address], nodes[i+1])
		} else if i == len(nodes)-1 {
			topology[node.DHT.Address] = append(topology[node.DHT.Address], nodes[i-1])
		} else {
			topology[node.DHT.Address] = append(topology[node.DHT.Address], nodes[i+1])
			topology[node.DHT.Address] = append(topology[node.DHT.Address], nodes[i-1])
		}
	}
	return nodes, topology, nil
}

func generateRingTopology(numberOfNodes int, delegate network.Delegate) ([]*network.Node, map[identity.Address][]*network.Node, error) {
	nodes, err := generateNodes(numberOfNodes, delegate, BOOSTRAP_NODE_PORT)
	if err != nil {
		return nil, nil, err
	}
	topology := map[identity.Address][]*network.Node{}
	for i, node := range nodes {
		topology[node.DHT.Address] = []*network.Node{}
		if i == 0 {
			topology[node.DHT.Address] = append(topology[node.DHT.Address], nodes[i+1])
			topology[node.DHT.Address] = append(topology[node.DHT.Address], nodes[len(nodes)-1])
		} else if i == len(nodes)-1 {
			topology[node.DHT.Address] = append(topology[node.DHT.Address], nodes[i-1])
			topology[node.DHT.Address] = append(topology[node.DHT.Address], nodes[0])
		} else {
			topology[node.DHT.Address] = append(topology[node.DHT.Address], nodes[i+1])
			topology[node.DHT.Address] = append(topology[node.DHT.Address], nodes[i-1])
		}
	}
	return nodes, topology, nil
}

func ping(nodes []*network.Node, topology map[identity.Address][]*network.Node) error {
	var wg sync.WaitGroup
	wg.Add(len(nodes))
	var muError *sync.Mutex
	var globalError error = nil

	for _, node := range nodes {
		go func(node *network.Node) {
			defer wg.Done()
			peers := topology[node.DHT.Address]
			for _, peer := range peers {
				err := rpc.PingTarget(peer.MultiAddress(), node.MultiAddress(), time.Second)
				if err != nil {
					muError.Lock()
					defer muError.Unlock()
					globalError = err
				}
			}
		}(node)
	}

	wg.Wait()
	return globalError
}

func peers(nodes []*network.Node, topology map[identity.Address][]*network.Node) error {
	var wg sync.WaitGroup
	wg.Add(len(nodes))
	var muError *sync.Mutex
	var globalError error = nil

	for _, node := range nodes {
		go func(node *network.Node) {
			defer wg.Done()
			peers := topology[node.DHT.Address]
			connectedPeers, err := rpc.GetPeersFromTarget(node.MultiAddress(), identity.MultiAddress{}, time.Second)
			if err != nil {
				muError.Lock()
				defer muError.Unlock()
				globalError = err
			}
			for _, peer := range peers {
				connected := false
				for _, connectedPeer := range connectedPeers {
					if peer.MultiAddress().String() == connectedPeer.String() {
						connected = true
					}
				}
				if !connected {
					if err != nil {
						muError.Lock()
						defer muError.Unlock()
						globalError = fmt.Errorf("%s should be connected to %s", node.MultiAddress().String(), peer.MultiAddress().String())
					}
					return
				}
			}
		}(node)
	}

	wg.Wait()
	return globalError
}
