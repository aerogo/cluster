package cluster

// Force interface implementation
var _ Node = (*ClientNode)(nil)

// ClientNode ...
type ClientNode struct {
}

// start ...
func (node *ClientNode) start() error {
	return nil
}

// Close ...
func (node *ClientNode) Close() {
	// ...
}

// IsClosed ...
func (node *ClientNode) IsClosed() bool {
	return false
}

// IsServer ...
func (node *ClientNode) IsServer() bool {
	return false
}
