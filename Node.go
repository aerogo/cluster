package cluster

import "github.com/aerogo/cluster/server"

// Node ...
type Node interface {
	Close()
	IsClosed() bool
	IsServer() bool
}

// Force interface implementations
var (
	_ Node = (*server.Node)(nil)
	_ Node = (*ClientNode)(nil)
)

// New ...
func New(port int) Node {
	node := server.New(port)
	node.Start()
	return node
}
