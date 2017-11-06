package cluster

import (
	"net"

	"github.com/aerogo/cluster/client"
	"github.com/aerogo/cluster/server"
	"github.com/aerogo/packet"
)

// Node ...
type Node interface {
	Broadcast(*packet.Packet)
	Address() net.Addr
	Close()
	IsClosed() bool
	IsServer() bool
}

// Force interface implementations
var (
	_ Node = (*server.Node)(nil)
	_ Node = (*client.Node)(nil)
)

// New ...
func New(port int, hosts ...string) Node {
	serverNode := server.New(port, hosts...)

	if serverNode.Start() == nil {
		return serverNode
	}

	clientNode := client.New(port, "localhost")
	err := clientNode.Connect()

	if err != nil {
		panic(err)
	}

	return clientNode
}
