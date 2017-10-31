package server

import (
	"github.com/aerogo/packet"
)

// Client ...
type Client struct {
	packet.Stream
	Node *Node
}

// read ...
func (client *Client) read() {
	err := client.Read()

	if err != nil {
		// fmt.Println(err)
	}

	client.Node.deadConnections <- client.Connection
}

// write ...
func (client *Client) write() {
	err := client.Write()

	if err != nil {
		// fmt.Println(err)
	}

	client.Node.deadConnections <- client.Connection
}
