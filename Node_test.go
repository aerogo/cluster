package cluster_test

import (
	"sync"
	"testing"
	"time"

	"github.com/aerogo/cluster"
	"github.com/aerogo/cluster/client"
	"github.com/aerogo/cluster/server"
	"github.com/aerogo/packet"
	"github.com/stretchr/testify/assert"
)

const nodeCount = 5

func TestClusterClose(t *testing.T) {
	nodes := make([]cluster.Node, nodeCount)

	for i := 0; i < nodeCount; i++ {
		nodes[i] = cluster.New(3000)
	}

	time.Sleep(100 * time.Millisecond)

	for i := 0; i < nodeCount; i++ {
		assert.False(t, nodes[i].IsClosed(), "node[%d].IsClosed()", i)
		nodes[i].Close()
		assert.True(t, nodes[i].IsClosed(), "node[%d].IsClosed()", i)
	}
}

func TestClusterBroadcast(t *testing.T) {
	nodes := make([]cluster.Node, nodeCount)
	wg := sync.WaitGroup{}
	message := "hello"

	for i := 0; i < nodeCount; i++ {
		nodes[i] = cluster.New(3000)

		if nodes[i].IsServer() {
			continue
		}

		wg.Add(1)

		go func(node *client.Node) {
			for msg := range node.Stream.Incoming {
				switch msg.Type {
				case 0:
					assert.Equal(t, message, string(msg.Data))
					wg.Done()
					return

				default:
					panic("Unknown message")
				}
			}
		}(nodes[i].(*client.Node))
	}

	// Wait for clients to connect
	for nodes[0].(*server.Node).ClientCount() < nodeCount-1 {
		time.Sleep(10 * time.Millisecond)
	}

	nodes[0].Broadcast(packet.New(0, []byte(message)))
	wg.Wait()

	for i := 0; i < nodeCount; i++ {
		nodes[i].Close()
	}
}
