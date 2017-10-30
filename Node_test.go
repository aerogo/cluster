package cluster_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aerogo/cluster"
)

func TestNode(t *testing.T) {
	node := cluster.New(3000)

	assert.NotNil(t, node)
	assert.True(t, node.IsServer())
	assert.False(t, node.IsClosed())

	node.Close()

	assert.True(t, node.IsClosed())
}
