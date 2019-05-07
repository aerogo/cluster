package cluster

import (
	"net"

	"github.com/aerogo/cluster/client"
	"github.com/aerogo/cluster/server"
	"github.com/aerogo/packet"
)

// Node is a general-purpose node in the cluster. It can act either as a server or as a client.
type Node interface {
	Address() net.Addr
	Broadcast(*packet.Packet)
	Close()
	IsClosed() bool
	IsServer() bool
}

// Force interface implementations
var (
	_ Node = (*server.Node)(nil)
	_ Node = (*client.Node)(nil)
)

// New creates a new node in the cluster.
func New(port int, hosts ...string) Node {
	// Try to bind the port to start as a server
	serverNode := server.New(port, hosts...)

	if serverNode.Start() == nil {
		return serverNode
	}

	// If the port binding failed, this node will be a client
	clientNode := client.New(port, "localhost")
	err := clientNode.Connect()

	if err != nil {
		panic(err)
	}

	return clientNode
}
