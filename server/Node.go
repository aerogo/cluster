package server

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aerogo/packet"
)

// Node represents a server node.
type Node struct {
	listener          net.Listener
	clients           sync.Map
	clientCount       int32
	newConnections    chan net.Conn
	deadConnections   chan net.Conn
	close             chan bool
	closed            atomic.Value
	port              int
	onConnect         []func(*packet.Stream)
	onDisconnect      []func(*packet.Stream)
	onConnectMutex    sync.Mutex
	onDisconnectMutex sync.Mutex
	hosts             []string
	localHosts        map[string]bool
	verbose           bool
}

// New creates a new server node.
func New(port int, hosts ...string) *Node {
	// Filter out local hosts
	localHosts := allLocalHosts()
	filteredHosts := []string{}

	for _, host := range hosts {
		_, skip := localHosts[host]

		if skip {
			continue
		}

		filteredHosts = append(filteredHosts, host)
	}

	return &Node{
		newConnections:  make(chan net.Conn, 32),
		deadConnections: make(chan net.Conn, 32),
		close:           make(chan bool),
		port:            port,
		hosts:           filteredHosts,
		localHosts:      localHosts,
	}
}

// Start ...
func (node *Node) Start() error {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(node.port))

	if err != nil {
		return err
	}

	atomic.StoreInt32(&node.clientCount, 0)
	node.closed.Store(false)
	node.listener = listener

	go node.mainLoop()
	go node.acceptConnections()

	// Connect to other hosts
	for _, host := range node.hosts {
		address := host + ":" + strconv.Itoa(node.port)
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)

		if err != nil {
			fmt.Println("Dead node:", address)
			continue
		}

		fmt.Println("Alive node:", address)
		node.newConnections <- conn
	}

	return nil
}

// mainLoop ...
func (node *Node) mainLoop() {
	if node.verbose {
		fmt.Println("[server] main loop started")
	}

	for {
		select {
		case connection := <-node.newConnections:
			if node.verbose {
				fmt.Println("[server] New connection", connection.RemoteAddr())
			}

			// Configure connection
			err := connection.(*net.TCPConn).SetNoDelay(true)

			if err != nil {
				fmt.Printf("Failed setting NoDelay on: %s -> %s\n", connection.LocalAddr().String(), connection.RemoteAddr().String())
			}

			err = connection.(*net.TCPConn).SetKeepAlive(true)

			if err != nil {
				fmt.Printf("Failed setting KeepAlive on: %s -> %s\n", connection.LocalAddr().String(), connection.RemoteAddr().String())
			}

			err = connection.(*net.TCPConn).SetLinger(-1)

			if err != nil {
				fmt.Printf("Failed setting Linger on: %s -> %s\n", connection.LocalAddr().String(), connection.RemoteAddr().String())
			}

			// Create server connection object
			stream := packet.NewStream(8192)

			stream.OnError(func(ioErr packet.IOError) {
				node.deadConnections <- ioErr.Connection
			})

			stream.SetConnection(connection)

			// Add connection to our list
			node.clients.Store(connection, stream)
			atomic.AddInt32(&node.clientCount, 1)

			node.onConnectMutex.Lock()

			for _, callback := range node.onConnect {
				callback(stream)
			}

			node.onConnectMutex.Unlock()

		case connection := <-node.deadConnections:
			if node.verbose {
				fmt.Println("[server] Dead connection", connection.RemoteAddr())
			}

			// Get stream object
			obj, exists := node.clients.Load(connection)

			if !exists {
				break
			}

			stream := obj.(*packet.Stream)
			stream.Close()

			// Remove connection from our list
			node.clients.Delete(connection)
			atomic.AddInt32(&node.clientCount, -1)
			node.onDisconnectMutex.Lock()

			for _, callback := range node.onDisconnect {
				callback(stream)
			}

			node.onDisconnectMutex.Unlock()

		case <-node.close:
			node.closed.Store(true)

			// Stop the server
			if node.verbose {
				fmt.Println("[server] close listener")
			}

			err := node.listener.Close()

			if err != nil {
				fmt.Println(err)
			}

			// This fixes a bug where listener.Close() doesn't close the listener fast enough
			time.Sleep(1 * time.Millisecond)

			// Stop client connections
			node.clients.Range(func(_, client interface{}) bool {
				stream := client.(*packet.Stream)

				for len(stream.Outgoing) > 0 {
					time.Sleep(1 * time.Millisecond)
				}

				// This prevents the send buffer from being discarded
				time.Sleep(1 * time.Millisecond)

				if node.verbose {
					fmt.Println("[server] close client", stream.Connection().RemoteAddr())
				}

				stream.Connection().Close()
				return true
			})

			// Tell the main thread we finished closing
			close(node.close)

			if node.verbose {
				fmt.Println("[server] exit main loop")
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

		// Is the client on our local machine?
		remoteAddr := conn.RemoteAddr().(*net.TCPAddr)
		ip := remoteAddr.IP.String()
		_, ok := node.localHosts[ip]

		// If not, maybe it is in our list of registered hosts.
		if !ok {
			for _, host := range node.hosts {
				if host == ip {
					ok = true
					break
				}
			}
		}

		// Don't allow remote connections from unregistered hosts.
		if !ok {
			conn.Close()
			continue
		}

		node.newConnections <- conn.(*net.TCPConn)
	}
}

// Broadcast sends a packet towards all clients.
func (node *Node) Broadcast(msg *packet.Packet) {
	node.BroadcastFiltered(msg, nil)
}

// BroadcastFiltered sends a packet towards all clients.
func (node *Node) BroadcastFiltered(msg *packet.Packet, filter func(*packet.Stream) bool) {
	for stream := range node.AllClients() {
		// Skip this client if filtered
		if filter != nil && !filter(stream) {
			continue
		}

		// Log message
		if node.verbose {
			fmt.Println("[server] broadcast to", stream.Connection().RemoteAddr())
		}

		// Send the packet
		select {
		case stream.Outgoing <- msg:
			// Sent successfully.
		default:
			// Discard packet.
			// TODO: Find a better solution to deal with this.
		}
	}
}

// Address returns the net.Addr of the server.
func (node *Node) Address() net.Addr {
	return node.listener.Addr()
}

// AllClients returns a channel of all clients' packet streams.
func (node *Node) AllClients() chan *packet.Stream {
	channel := make(chan *packet.Stream, 128)

	go func() {
		node.clients.Range(func(key, value interface{}) bool {
			channel <- value.(*packet.Stream)
			return true
		})

		close(channel)
	}()

	return channel
}

// Close closes the server.
func (node *Node) Close() {
	// This will block until the close signal is processed
	node.close <- true

	// Wait for completion signal
	<-node.close
}

// OnConnect ...
func (node *Node) OnConnect(callback func(*packet.Stream)) {
	if callback == nil {
		return
	}

	node.onConnectMutex.Lock()
	node.onConnect = append(node.onConnect, callback)
	node.onConnectMutex.Unlock()
}

// OnDisconnect ...
func (node *Node) OnDisconnect(callback func(*packet.Stream)) {
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

// IsRemoteAddress tells you whether a given address is a remote address (on another machine).
func (node *Node) IsRemoteAddress(addr net.Addr) bool {
	ip := addressToIP(addr)
	_, isLocal := node.localHosts[ip.String()]
	return !isLocal
}

// addressToIP ...
func addressToIP(addr net.Addr) net.IP {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP
	case *net.IPAddr:
		return v.IP
	case *net.TCPAddr:
		return v.IP
	case *net.UDPAddr:
		return v.IP
	}

	return nil
}

// allLocalHosts ...
func allLocalHosts() map[string]bool {
	hosts := map[string]bool{}
	ifaces, err := net.Interfaces()

	if err != nil {
		return nil
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()

		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ip := addressToIP(addr)
			hosts[ip.String()] = true
		}
	}

	return hosts
}
