package cluster

import (
	"github.com/aerogo/packet"
)

// ServerConnection ...
type ServerConnection struct {
	packet.Stream
	serverNode *ServerNode
}

// read ...
func (client *ServerConnection) read() {
	err := client.Stream.Read()

	if err != nil {
		// fmt.Println(err)
	}

	client.serverNode.deadConnections <- client.Connection
}

// write ...
func (client *ServerConnection) write() {
	err := client.Stream.Write()

	if err != nil {
		// fmt.Println(err)
	}

	client.serverNode.deadConnections <- client.Connection
}
