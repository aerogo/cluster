package client

import (
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/aerogo/packet"
)

// Node ...
type Node struct {
	packet.Stream
	port   int
	host   string
	close  chan bool
	closed atomic.Value
}

// New ...
func New(port int, host string) *Node {
	return &Node{
		port:  port,
		host:  host,
		close: make(chan bool),
	}
}

// Start ...
func (node *Node) Start() error {
	var conn net.Conn
	var err error

	const maxRetries = 10
	try := 0

	for try < maxRetries {
		conn, err = net.Dial("tcp", node.host+":"+strconv.Itoa(node.port))

		if err == nil && conn != nil {
			break
		}

		time.Sleep(100 * time.Millisecond)
		try++
	}

	if err != nil {
		return err
	}

	node.closed.Store(false)

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

	node.closed.Store(true)

	for {
		time.Sleep(1 * time.Millisecond)

		if len(node.Incoming) == 0 && len(node.Outgoing) == 0 {
			break
		}
	}

	err := node.Connection.Close()

	if err != nil {
		panic(err)
	}

	node.close <- true
}

// Broadcast ...
func (node *Node) Broadcast(msg *packet.Packet) {
	node.Outgoing <- msg
}

// Address ...
func (node *Node) Address() net.Addr {
	return node.Connection.LocalAddr()
}

// Close ...
func (node *Node) Close() {
	if node.IsClosed() {
		return
	}

	// This will block until the close signal is processed
	node.close <- true

	// Wait for completion signal
	<-node.close

	// Close channel
	close(node.close)
}

// IsClosed ...
func (node *Node) IsClosed() bool {
	return node.closed.Load().(bool)
}

// IsServer ...
func (node *Node) IsServer() bool {
	return false
}
