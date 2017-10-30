package client

// Node ...
type Node struct {
}

// New ...
func New(serverPort int) *Node {
	return &Node{}
}

// Start ...
func (node *Node) Start() error {
	return nil
}

// Close ...
func (node *Node) Close() {
	// ...
}

// IsClosed ...
func (node *Node) IsClosed() bool {
	return false
}

// IsServer ...
func (node *Node) IsServer() bool {
	return false
}
