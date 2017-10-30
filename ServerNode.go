package cluster

// ServerNode ...
type ServerNode struct {
}

// IsServer ...
func (node *ServerNode) IsServer() bool {
	return true
}
