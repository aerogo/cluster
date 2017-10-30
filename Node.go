package cluster

// Node ...
type Node interface {
	Close()
	IsClosed() bool
	IsServer() bool
}

// New ...
func New() Node {
	node := &ServerNode{}
	node.start()
	return node
}
