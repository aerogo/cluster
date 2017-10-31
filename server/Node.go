package server

import (
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aerogo/packet"
)

// Node ...
type Node struct {
	listener          net.Listener
	clients           sync.Map
	clientCount       int32
	newClients        chan net.Conn
	deadClients       chan net.Conn
	close             chan bool
	closed            atomic.Value
	port              int
	onConnect         []func(*Client)
	onDisconnect      []func(*Client)
	onConnectMutex    sync.Mutex
	onDisconnectMutex sync.Mutex
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

			node.onConnectMutex.Lock()

			for _, callback := range node.onConnect {
				callback(client)
			}

			node.onConnectMutex.Unlock()

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

			node.onDisconnectMutex.Lock()

			for _, callback := range node.onDisconnect {
				callback(client)
			}

			node.onDisconnectMutex.Unlock()

		case <-node.close:
			node.closed.Store(true)

			// Stop the server
			err := node.listener.Close()

			if err != nil {
				panic(err)
			}

			time.Sleep(1 * time.Millisecond)

			// Tell the main thread we finished closing
			node.close <- true

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
		client.Outgoing <- msg
	}
}

// AllClients ...
func (node *Node) AllClients() chan *Client {
	channel := make(chan *Client, 128)

	go func() {
		node.clients.Range(func(key, value interface{}) bool {
			channel <- value.(*Client)
			return true
		})

		close(channel)
	}()

	return channel
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

// OnConnect ...
func (node *Node) OnConnect(callback func(*Client)) {
	if callback == nil {
		return
	}

	node.onConnectMutex.Lock()
	node.onConnect = append(node.onConnect, callback)
	node.onConnectMutex.Unlock()
}

// OnDisconnect ...
func (node *Node) OnDisconnect(callback func(*Client)) {
	if callback == nil {
		return
	}

	node.onDisconnectMutex.Lock()
	node.onDisconnect = append(node.onDisconnect, callback)
	node.onDisconnectMutex.Unlock()
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
