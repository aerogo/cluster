package cluster_test

import (
	"testing"
	"time"

	"github.com/aerogo/cluster"
	"github.com/stretchr/testify/assert"
)

const nodeCount = 5

func TestClusterClose(t *testing.T) {
	nodes := make([]cluster.Node, nodeCount, nodeCount)

	for i := 0; i < nodeCount; i++ {
		nodes[i] = cluster.New(3000)
	}

	time.Sleep(100 * time.Millisecond)

	for i := 0; i < nodeCount; i++ {
		nodes[i].Close()
	}

	time.Sleep(100 * time.Millisecond)

	for i := 0; i < nodeCount; i++ {
		assert.True(t, nodes[i].IsClosed(), "node[%d].IsClosed()", i)
	}
}
