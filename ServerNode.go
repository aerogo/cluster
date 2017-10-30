package cluster

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/aerogo/packet"
)

// Force interface implementation
var _ Node = (*ServerNode)(nil)

// ServerNode ...
type ServerNode struct {
	listener        net.Listener
	connections     sync.Map
	connectionCount int32
	newConnections  chan net.Conn
	deadConnections chan net.Conn
	close           chan bool
	closed          bool
}

// start ...
func (node *ServerNode) start() error {
	node.newConnections = make(chan net.Conn, 32)
	node.deadConnections = make(chan net.Conn, 32)
	node.close = make(chan bool)

	listener, err := net.Listen("tcp", ":3000")

	if err != nil {
		return err
	}

	node.listener = listener

	go node.mainLoop()
	go node.acceptConnections()

	return nil
}

// mainLoop ...
func (node *ServerNode) mainLoop() {
	for {
		select {
		case connection := <-node.newConnections:
			connection.(*net.TCPConn).SetNoDelay(true)
			connection.(*net.TCPConn).SetKeepAlive(true)

			client := &ServerConnection{
				serverNode: node,
				Stream: packet.Stream{
					Connection: connection,
					Incoming:   make(chan *packet.Packet),
					Outgoing:   make(chan *packet.Packet),
				},
			}

			node.connections.Store(connection, client)
			atomic.AddInt32(&node.connectionCount, 1)

			go client.read()
			go client.write()

		case connection := <-node.deadConnections:
			obj, exists := node.connections.Load(connection)

			if !exists {
				break
			}

			client := obj.(*ServerConnection)
			close(client.Incoming)
			close(client.Outgoing)
			connection.Close()
			node.connections.Delete(connection)
			atomic.AddInt32(&node.connectionCount, -1)

		case <-node.close:
			node.closed = true
			err := node.listener.Close()

			if err != nil {
				panic(err)
			}

			return
		}
	}
}

// acceptConnections ...
func (node *ServerNode) acceptConnections() {
	for {
		conn, err := node.listener.Accept()

		if err != nil {
			if node.closed {
				return
			}

			panic(err)
		}

		node.newConnections <- conn.(*net.TCPConn)
	}
}

// Close ...
func (node *ServerNode) Close() {
	// ...
}

// IsClosed ...
func (node *ServerNode) IsClosed() bool {
	return node.closed
}

// IsServer ...
func (node *ServerNode) IsServer() bool {
	return true
}
