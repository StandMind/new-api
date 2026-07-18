package main

import (
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/require"
)

func TestMigrationOnlyRequested(t *testing.T) {
	previous := os.Args
	t.Cleanup(func() {
		os.Args = previous
	})

	os.Args = []string{"new-api", "--log-dir", "/tmp/logs"}
	require.False(t, migrationOnlyRequested())

	os.Args = []string{"new-api", "--migrate-only", "--log-dir", "/tmp/logs"}
	require.True(t, migrationOnlyRequested())
}

func TestValidateMigrationNode(t *testing.T) {
	previous := common.IsMasterNode
	t.Cleanup(func() {
		common.IsMasterNode = previous
	})

	common.IsMasterNode = false
	require.ErrorContains(t, validateMigrationNode(), "NODE_TYPE=slave")

	common.IsMasterNode = true
	require.NoError(t, validateMigrationNode())
}
