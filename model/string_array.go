package model

import (
	"database/sql/driver"
	"fmt"

	"github.com/QuantumNous/new-api/common"
)

// StringArray is stored as JSON in a TEXT column while remaining a JSON array
// on the API and Redis cache boundaries.
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	data, err := common.Marshal(a)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func (a *StringArray) Scan(value any) error {
	if value == nil {
		*a = nil
		return nil
	}

	var data []byte
	switch typed := value.(type) {
	case []byte:
		data = typed
	case string:
		data = []byte(typed)
	default:
		return fmt.Errorf("unsupported StringArray database value %T", value)
	}
	if len(data) == 0 {
		*a = nil
		return nil
	}
	return common.Unmarshal(data, a)
}
