package cluster

// ClientNode ...
type ClientNode struct {
}

// IsServer ...
func (node *ClientNode) IsServer() bool {
	return false
}
