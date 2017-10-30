package cluster

// Node ...
type Node interface {
	Close()
	IsClosed() bool
	IsServer() bool
}

// New ...
func New(port int) Node {
	node := &ServerNode{
		port: port,
	}
	node.start()
	return node
}
