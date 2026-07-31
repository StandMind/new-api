package model

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	RequestDetailOutcomeSuccess = "success"
	RequestDetailOutcomeFailed  = "failed"
)

type RequestDetail struct {
	ID                int64  `json:"id" gorm:"primaryKey"`
	RequestID         string `json:"request_id" gorm:"type:varchar(64);uniqueIndex:idx_request_details_request_id"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint;index:idx_request_details_created_at"`
	UserID            int    `json:"user_id" gorm:"index:idx_request_details_user_id"`
	Username          string `json:"username" gorm:"type:varchar(128);index:idx_request_details_username;default:''"`
	TokenID           int    `json:"token_id" gorm:"default:0"`
	TokenName         string `json:"token_name" gorm:"type:varchar(128);default:''"`
	ModelName         string `json:"model_name" gorm:"type:varchar(191);index:idx_request_details_model;default:''"`
	Group             string `json:"group" gorm:"type:varchar(64);default:''"`
	Method            string `json:"method" gorm:"type:varchar(16);default:''"`
	Path              string `json:"path" gorm:"type:varchar(512);default:''"`
	IP                string `json:"ip" gorm:"type:varchar(64);default:''"`
	Outcome           string `json:"outcome" gorm:"type:varchar(16);index:idx_request_details_outcome"`
	StatusCode        int    `json:"status_code" gorm:"default:0"`
	ErrorType         string `json:"error_type" gorm:"type:varchar(128);default:''"`
	ErrorCode         string `json:"error_code" gorm:"type:varchar(128);default:''"`
	ErrorMessage      string `json:"error_message" gorm:"type:text"`
	ContentType       string `json:"content_type" gorm:"type:varchar(255);default:''"`
	BodySize          int64  `json:"body_size" gorm:"default:0"`
	BodyOmittedReason string `json:"body_omitted_reason" gorm:"type:varchar(64);default:''"`
	Payload           []byte `json:"-"`
	StorageBytes      int64  `json:"storage_bytes" gorm:"default:0"`
	HasPayload        bool   `json:"has_payload" gorm:"-"`
}

type RequestDetailFilter struct {
	StartTimestamp int64
	EndTimestamp   int64
	Username       string
	ModelName      string
	Outcome        string
	RequestID      string
	StartIndex     int
	Limit          int
}

type RequestDetailStorageStats struct {
	Count        int64 `json:"count"`
	StorageBytes int64 `json:"storage_bytes"`
	OldestAt     int64 `json:"oldest_at"`
	NewestAt     int64 `json:"newest_at"`
}

type RequestDetailCleanupResult struct {
	DeletedCount int64 `json:"deleted_count"`
	FreedBytes   int64 `json:"freed_bytes"`
}

type requestDetailDeleteRow struct {
	ID           int64
	RequestID    string
	StorageBytes int64
}

func CreateRequestDetails(details []*RequestDetail) error {
	if len(details) == 0 {
		return nil
	}
	return LOG_DB.CreateInBatches(details, len(details)).Error
}

func GetRequestDetails(filter RequestDetailFilter) ([]*RequestDetail, int64, error) {
	query := LOG_DB.Model(&RequestDetail{})
	if filter.StartTimestamp > 0 {
		query = query.Where("created_at >= ?", filter.StartTimestamp)
	}
	if filter.EndTimestamp > 0 {
		query = query.Where("created_at <= ?", filter.EndTimestamp)
	}
	if filter.Username != "" {
		query = query.Where("username = ?", filter.Username)
	}
	if filter.ModelName != "" {
		query = query.Where("model_name = ?", filter.ModelName)
	}
	if filter.Outcome != "" {
		query = query.Where("outcome = ?", filter.Outcome)
	}
	if filter.RequestID != "" {
		query = query.Where("request_id = ?", filter.RequestID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	var details []*RequestDetail
	err := query.Select(
		"id", "request_id", "created_at", "user_id", "username", "token_id", "token_name",
		"model_name", "group", "method", "path", "ip", "outcome", "status_code", "error_type",
		"error_code", "error_message", "content_type", "body_size", "body_omitted_reason", "storage_bytes",
	).Order("created_at DESC, request_id DESC").Offset(filter.StartIndex).Limit(limit).Find(&details).Error
	if err != nil {
		return nil, 0, err
	}
	for _, detail := range details {
		detail.HasPayload = detail.StorageBytes > 0
	}
	return details, total, nil
}

func GetRequestDetailByRequestID(requestID string) (*RequestDetail, error) {
	var detail RequestDetail
	err := LOG_DB.Where("request_id = ?", requestID).Order("created_at DESC").First(&detail).Error
	if err != nil {
		return nil, err
	}
	detail.HasPayload = len(detail.Payload) > 0
	return &detail, nil
}

func GetRequestDetailStorageStats(ctx context.Context) (RequestDetailStorageStats, error) {
	var stats RequestDetailStorageStats
	err := LOG_DB.WithContext(ctx).Model(&RequestDetail{}).Select(
		"COUNT(*) AS count, COALESCE(SUM(storage_bytes), 0) AS storage_bytes, " +
			"COALESCE(MIN(created_at), 0) AS oldest_at, COALESCE(MAX(created_at), 0) AS newest_at",
	).Scan(&stats).Error
	return stats, err
}

func CleanupRequestDetails(ctx context.Context, cutoffTimestamp, targetBytes int64, batchSize int) (RequestDetailCleanupResult, error) {
	if batchSize <= 0 {
		batchSize = 500
	}
	result := RequestDetailCleanupResult{}

	if cutoffTimestamp > 0 && !common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		for {
			rows, err := selectRequestDetailDeleteRows(ctx, cutoffTimestamp, "", batchSize)
			if err != nil {
				return result, err
			}
			if len(rows) == 0 {
				break
			}
			deleted, freed, err := deleteRequestDetailRows(ctx, rows)
			if err != nil {
				return result, err
			}
			result.DeletedCount += deleted
			result.FreedBytes += freed
		}
	}

	if targetBytes <= 0 {
		return result, nil
	}
	stats, err := GetRequestDetailStorageStats(ctx)
	if err != nil {
		return result, err
	}
	for stats.StorageBytes > targetBytes {
		rows, selectErr := selectRequestDetailDeleteRows(ctx, 0, RequestDetailOutcomeSuccess, batchSize)
		if selectErr != nil {
			return result, selectErr
		}
		if len(rows) == 0 {
			rows, selectErr = selectRequestDetailDeleteRows(ctx, 0, "", batchSize)
			if selectErr != nil {
				return result, selectErr
			}
		}
		if len(rows) == 0 {
			break
		}
		deleted, freed, deleteErr := deleteRequestDetailRows(ctx, rows)
		if deleteErr != nil {
			return result, deleteErr
		}
		if deleted == 0 {
			break
		}
		result.DeletedCount += deleted
		result.FreedBytes += freed
		stats.StorageBytes -= freed
	}
	return result, nil
}

func selectRequestDetailDeleteRows(ctx context.Context, beforeTimestamp int64, outcome string, limit int) ([]requestDetailDeleteRow, error) {
	query := LOG_DB.WithContext(ctx).Model(&RequestDetail{}).Select("id", "request_id", "storage_bytes")
	if beforeTimestamp > 0 {
		query = query.Where("created_at < ?", beforeTimestamp)
	}
	if outcome != "" {
		query = query.Where("outcome = ?", outcome)
	}
	var rows []requestDetailDeleteRow
	err := query.Order("created_at ASC, request_id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func deleteRequestDetailRows(ctx context.Context, rows []requestDetailDeleteRow) (int64, int64, error) {
	if len(rows) == 0 {
		return 0, 0, nil
	}
	requestIDs := make([]string, 0, len(rows))
	ids := make([]int64, 0, len(rows))
	freedBytes := int64(0)
	for _, row := range rows {
		requestIDs = append(requestIDs, row.RequestID)
		ids = append(ids, row.ID)
		freedBytes += row.StorageBytes
	}

	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		err := LOG_DB.WithContext(ctx).Exec(
			"ALTER TABLE request_details DELETE WHERE request_id IN ? SETTINGS mutations_sync = 1",
			requestIDs,
		).Error
		if err != nil {
			return 0, 0, err
		}
		return int64(len(rows)), freedBytes, nil
	}
	deleteResult := LOG_DB.WithContext(ctx).Where("id IN ?", ids).Delete(&RequestDetail{})
	if deleteResult.Error != nil {
		return 0, 0, deleteResult.Error
	}
	return deleteResult.RowsAffected, freedBytes, nil
}

func requestDetailClickHouseTTLExpression(retentionDays int) string {
	if retentionDays <= 0 {
		return ""
	}
	return fmt.Sprintf("toDateTime(created_at) + INTERVAL %d DAY DELETE", retentionDays)
}

func requestDetailClickHouseCreateTableSQL(retentionDays int) string {
	ttlClause := ""
	if expression := requestDetailClickHouseTTLExpression(retentionDays); expression != "" {
		ttlClause = "\nTTL " + expression
	}
	return fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS request_details (
	id Int64 DEFAULT 0,
	request_id String DEFAULT '',
	created_at Int64 DEFAULT 0,
	user_id Int32 DEFAULT 0,
	username String DEFAULT '',
	token_id Int32 DEFAULT 0,
	token_name String DEFAULT '',
	model_name String DEFAULT '',
	`+"`group`"+` String DEFAULT '',
	method String DEFAULT '',
	path String DEFAULT '',
	ip String DEFAULT '',
	outcome String DEFAULT '',
	status_code Int32 DEFAULT 0,
	error_type String DEFAULT '',
	error_code String DEFAULT '',
	error_message String DEFAULT '',
	content_type String DEFAULT '',
	body_size Int64 DEFAULT 0,
	body_omitted_reason String DEFAULT '',
	payload String DEFAULT '',
	storage_bytes Int64 DEFAULT 0
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(toDateTime(created_at))
ORDER BY (created_at, request_id)%s`, ttlClause)
}

func SyncRequestDetailClickHouseTTL(retentionDays int) error {
	if !common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		return nil
	}
	expression := requestDetailClickHouseTTLExpression(retentionDays)
	if expression != "" {
		return LOG_DB.Exec("ALTER TABLE request_details MODIFY TTL " + expression).Error
	}

	var createTableSQL string
	if err := LOG_DB.Raw("SHOW CREATE TABLE request_details").Scan(&createTableSQL).Error; err != nil {
		return err
	}
	upperSQL := strings.ToUpper(createTableSQL)
	if !strings.Contains(upperSQL, "\nTTL ") && !strings.Contains(upperSQL, " TTL ") {
		return nil
	}
	return LOG_DB.Exec("ALTER TABLE request_details REMOVE TTL").Error
}

func IsRequestDetailNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
