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
	Stream  *packet.Stream
	port    int
	host    string
	verbose bool
	close   chan bool
	closed  atomic.Value
}

// New ...
func New(port int, host string) *Node {
	node := &Node{
		port: port,
		host: host,
	}

	node.Stream = packet.NewStream(8192)
	return node
}

// Connect ...
func (node *Node) Connect() error {
	var conn net.Conn
	var err error

	for {
		if node.verbose {
			fmt.Println("[client] Connecting to", node.host+":"+strconv.Itoa(node.port))
		}

		conn, err = net.Dial("tcp", node.host+":"+strconv.Itoa(node.port))

		if err == nil && conn != nil {
			break
		}

		time.Sleep(time.Second)
	}

	if err != nil {
		return err
	}

	conn.(*net.TCPConn).SetNoDelay(true)
	conn.(*net.TCPConn).SetKeepAlive(true)
	conn.(*net.TCPConn).SetLinger(-1)

	node.close = make(chan bool)
	node.closed.Store(false)

	node.Stream.SetConnection(conn)
	go node.waitClose()

	if node.verbose {
		fmt.Println("[client] Successfully connected.", node.Address())
	}

	return nil
}

// waitClose ...
func (node *Node) waitClose() {
	<-node.close
	node.closed.Store(true)

	for len(node.Stream.Incoming) > 0 || len(node.Stream.Outgoing) > 0 {
		time.Sleep(1 * time.Millisecond)
	}

	// This prevents a bug where outgoing packets are not sent by the operating system yet.
	time.Sleep(1 * time.Millisecond)

	// Close connection only, not the stream itself because it's reusable with a different connection.
	node.Stream.Connection().Close()

	close(node.close)
}

// Connection ...
func (node *Node) Connection() net.Conn {
	return node.Stream.Connection()
}

// Broadcast ...
func (node *Node) Broadcast(msg *packet.Packet) {
	select {
	case node.Stream.Outgoing <- msg:
		return

	default:
		go func() {
			node.Stream.Outgoing <- msg
		}()

		return
	}
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
