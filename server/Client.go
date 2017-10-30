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
	err := client.Stream.Read()

	if err != nil {
		// fmt.Println(err)
	}

	client.Node.deadClients <- client.Connection
}

// write ...
func (client *Client) write() {
	err := client.Stream.Write()

	if err != nil {
		// fmt.Println(err)
	}

	client.Node.deadClients <- client.Connection
}
