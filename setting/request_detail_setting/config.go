package request_detail_setting

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/QuantumNous/new-api/setting/config"
)

const (
	ConfigName = "request_detail_setting"

	ModeAll    = "all"
	ModeFailed = "failed"
	ModeNone   = "none"

	DefaultRetentionDays = 7
	DefaultMaxStorageMB  = 5 * 1024
	MaxBodyBytes         = 64 * 1024
	MaxResponseBodyBytes = 1024 * 1024
)

type RequestDetailSetting struct {
	Mode          string `json:"mode"`
	RetentionDays int    `json:"retention_days"`
	MaxStorageMB  int    `json:"max_storage_mb"`
}

var (
	requestDetailSetting = RequestDetailSetting{
		Mode:          ModeFailed,
		RetentionDays: DefaultRetentionDays,
		MaxStorageMB:  DefaultMaxStorageMB,
	}
	requestDetailSettingSnapshot atomic.Value
)

func init() {
	config.GlobalConfig.Register(ConfigName, &requestDetailSetting)
	UpdateAndSync()
}

func GetSetting() RequestDetailSetting {
	return requestDetailSettingSnapshot.Load().(RequestDetailSetting)
}

func UpdateAndSync() {
	requestDetailSettingSnapshot.Store(requestDetailSetting)
}

func ValidateOption(key, value string) error {
	switch key {
	case ConfigName + ".mode":
		if value != ModeAll && value != ModeFailed && value != ModeNone {
			return fmt.Errorf("request detail mode must be all, failed, or none")
		}
	case ConfigName + ".retention_days":
		days, err := strconv.Atoi(value)
		if err != nil || days < 1 || days > 3650 {
			return fmt.Errorf("request detail retention days must be between 1 and 3650")
		}
	case ConfigName + ".max_storage_mb":
		maxStorageMB, err := strconv.Atoi(value)
		if err != nil || maxStorageMB < 128 || maxStorageMB > 10*1024*1024 {
			return fmt.Errorf("request detail storage limit must be between 128 MB and 10 TB")
		}
	default:
		return fmt.Errorf("unknown request detail setting: %s", key)
	}
	return nil
}
