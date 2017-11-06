package client

import (
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/aerogo/packet"
)

// Node ...
type Node struct {
	Stream *packet.Stream
	port   int
	host   string
	close  chan bool
	closed atomic.Value
}

// New ...
func New(port int, host string) *Node {
	node := &Node{
		port:  port,
		host:  host,
		close: make(chan bool),
	}

	node.Stream = packet.NewStream(4096)
	go node.waitClose()
	return node
}

// Connect ...
func (node *Node) Connect() error {
	var conn net.Conn
	var err error

	const maxRetries = 10
	try := 0

	for try < maxRetries {
		fmt.Println("Connecting to", node.host+":"+strconv.Itoa(node.port), "try", try)
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

	conn.(*net.TCPConn).SetNoDelay(true)
	conn.(*net.TCPConn).SetKeepAlive(true)

	node.closed.Store(false)

	node.Stream.SetConnection(conn)

	fmt.Println(node.Address(), "Successfully connected.")
	return nil
}

// waitClose ...
func (node *Node) waitClose() {
	<-node.close
	node.closed.Store(true)

	for len(node.Stream.Incoming) > 0 || len(node.Stream.Outgoing) > 0 {
		time.Sleep(1 * time.Millisecond)
	}

	err := node.Stream.Close()

	if err != nil {
		panic(err)
	}

	close(node.close)
}

// Connection ...
func (node *Node) Connection() net.Conn {
	return node.Stream.Connection()
}

// Broadcast ...
func (node *Node) Broadcast(msg *packet.Packet) {
	node.Stream.Outgoing <- msg
}

// Address ...
func (node *Node) Address() net.Addr {
	return node.Connection().LocalAddr()
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
}

// IsClosed ...
func (node *Node) IsClosed() bool {
	return node.closed.Load().(bool)
}

// IsServer ...
func (node *Node) IsServer() bool {
	return false
}
