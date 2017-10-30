package server

import (
	"net"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/aerogo/packet"
)

// Node ...
type Node struct {
	listener     net.Listener
	clients      sync.Map
	clientCount  int32
	newClients   chan net.Conn
	deadClients  chan net.Conn
	close        chan bool
	closed       atomic.Value
	port         int
	onConnect    []func(*Client)
	onDisconnect []func(*Client)
}

// New ...
func New(port int) *Node {
	return &Node{
		newClients:  make(chan net.Conn, 32),
		deadClients: make(chan net.Conn, 32),
		close:       make(chan bool),
		port:        port,
	}
}

// Start ...
func (node *Node) Start() error {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(node.port))

	if err != nil {
		return err
	}

	node.closed.Store(false)
	node.listener = listener

	go node.mainLoop()
	go node.acceptConnections()

	return nil
}

// mainLoop ...
func (node *Node) mainLoop() {
	for {
		select {
		case connection := <-node.newClients:
			// Configure connection
			connection.(*net.TCPConn).SetNoDelay(true)
			connection.(*net.TCPConn).SetKeepAlive(true)

			// Create server connection object
			client := &Client{
				Stream: packet.Stream{
					Connection: connection,
					Incoming:   make(chan *packet.Packet),
					Outgoing:   make(chan *packet.Packet),
				},
				Node: node,
			}

			// Add connection to our list
			node.clients.Store(connection, client)
			atomic.AddInt32(&node.clientCount, 1)

			// Start send and receive tasks
			go client.read()
			go client.write()

			for _, callback := range node.onConnect {
				callback(client)
			}

		case connection := <-node.deadClients:
			obj, exists := node.clients.Load(connection)

			if !exists {
				break
			}

			client := obj.(*Client)

			// Close channels
			close(client.Incoming)
			close(client.Outgoing)

			// Close connection
			connection.Close()

			// Remove connection from our list
			node.clients.Delete(connection)
			atomic.AddInt32(&node.clientCount, -1)

			for _, callback := range node.onDisconnect {
				callback(client)
			}

		case <-node.close:
			node.closed.Store(true)

			// Stop the server
			err := node.listener.Close()

			if err != nil {
				panic(err)
			}

			// Exit main loop
			return
		}
	}
}

// acceptConnections ...
func (node *Node) acceptConnections() {
	for {
		conn, err := node.listener.Accept()

		if err != nil {
			if node.IsClosed() {
				return
			}

			panic(err)
		}

		node.newClients <- conn.(*net.TCPConn)
	}
}

// Broadcast ...
func (node *Node) Broadcast(msg *packet.Packet) {
	for client := range node.AllClients() {
		client.(*Client).Outgoing <- msg
	}
}

// AllClients ...
func (node *Node) AllClients() chan interface{} {
	channel := make(chan interface{}, 128)
	go syncMapValues(&node.clients, channel)
	return channel
}

// Close ...
func (node *Node) Close() {
	if node.IsClosed() {
		return
	}

	node.close <- true
	close(node.close)
}

// OnConnect ...
func (node *Node) OnConnect(callback func(*Client)) {
	node.onConnect = append(node.onConnect, callback)
}

// OnDisconnect ...
func (node *Node) OnDisconnect(callback func(*Client)) {
	node.onDisconnect = append(node.onDisconnect, callback)
}

// ClientCount ...
func (node *Node) ClientCount() int {
	return int(atomic.LoadInt32(&node.clientCount))
}

// IsClosed ...
func (node *Node) IsClosed() bool {
	return node.closed.Load().(bool)
}

// IsServer ...
func (node *Node) IsServer() bool {
	return true
}

// syncMapValues iterates over all values in a sync.Map and sends them to the given channel.
func syncMapValues(data *sync.Map, channel chan interface{}) {
	data.Range(func(key, value interface{}) bool {
		channel <- value
		return true
	})

	close(channel)
}
