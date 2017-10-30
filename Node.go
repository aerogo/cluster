package cluster

import (
	"github.com/aerogo/cluster/client"
	"github.com/aerogo/cluster/server"
	"github.com/aerogo/packet"
)

// Node ...
type Node interface {
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

// New ...
func New(port int) Node {
	serverNode := server.New(port)

	if serverNode.Start() == nil {
		return serverNode
	}

	clientNode := client.New(port)
	err := clientNode.Start()

	if err != nil {
		panic(err)
	}

	return clientNode
}
