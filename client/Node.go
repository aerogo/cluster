package client

import (
	"net"
	"strconv"

	"github.com/aerogo/packet"
)

// Node ...
type Node struct {
	packet.Stream
	serverPort int
	close      chan bool
	closed     bool
}

// New ...
func New(serverPort int) *Node {
	return &Node{
		serverPort: serverPort,
		close:      make(chan bool),
	}
}

// Start ...
func (node *Node) Start() error {
	conn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(node.serverPort))

	if err != nil {
		return err
	}

	node.Connection = conn
	node.Incoming = make(chan *packet.Packet)
	node.Outgoing = make(chan *packet.Packet)

	conn.(*net.TCPConn).SetNoDelay(true)
	conn.(*net.TCPConn).SetKeepAlive(true)

	go node.read()
	go node.write()
	go node.waitClose()

	return nil
}

// read ...
func (node *Node) read() {
	err := node.Read()

	if err != nil {
		// fmt.Println(err)
	}
}

// write ...
func (node *Node) write() {
	err := node.Write()

	if err != nil {
		// fmt.Println(err)
	}
}

// waitClose ...
func (node *Node) waitClose() {
	<-node.close

	node.closed = true
	err := node.Connection.Close()

	if err != nil {
		panic(err)
	}
}

// Broadcast ...
func (node *Node) Broadcast(msg *packet.Packet) {
	node.Outgoing <- msg
}

// Close ...
func (node *Node) Close() {
	if node.closed {
		return
	}

	node.close <- true
}

// IsClosed ...
func (node *Node) IsClosed() bool {
	return node.closed
}

// IsServer ...
func (node *Node) IsServer() bool {
	return false
}
