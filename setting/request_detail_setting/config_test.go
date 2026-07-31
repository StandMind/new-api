package request_detail_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOption(t *testing.T) {
	require.NoError(t, ValidateOption(ConfigName+".mode", ModeFailed))
	require.NoError(t, ValidateOption(ConfigName+".retention_days", "7"))
	require.NoError(t, ValidateOption(ConfigName+".max_storage_mb", "5120"))

	assert.Error(t, ValidateOption(ConfigName+".mode", "sometimes"))
	assert.Error(t, ValidateOption(ConfigName+".retention_days", "0"))
	assert.Error(t, ValidateOption(ConfigName+".max_storage_mb", "127"))
}
