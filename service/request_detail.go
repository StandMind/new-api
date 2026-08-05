package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/request_detail_setting"

	"github.com/gin-gonic/gin"
)

const (
	requestDetailBatchSize              = 100
	requestDetailFlushInterval          = 250 * time.Millisecond
	requestDetailMaintenanceInterval    = time.Minute
	requestDetailRetentionCheckInterval = time.Hour
	requestDetailStorageRefreshInterval = 5 * time.Minute
	requestDetailFailureQueueSize       = 512
	requestDetailSuccessQueueSize       = 2048
	requestDetailCleanupBatchSize       = 500
	requestDetailMaxErrorBytes          = 8 * 1024
	requestDetailMaxDecodedPayloadBytes = 4 * request_detail_setting.MaxResponseBodyBytes
	requestDetailStorageOverheadBytes   = 512
	requestDetailMinFreeDiskBytes       = 2 << 30
	requestDetailFailureContextKey      = "request_detail_failure"
	requestDetailResponseContextKey     = "request_detail_response"
)

type RequestDetailFailure struct {
	StatusCode int
	ErrorType  string
	ErrorCode  string
	Message    string
}

type RequestDetailPayload struct {
	Headers  map[string]string             `json:"headers,omitempty"`
	Query    map[string]interface{}        `json:"query,omitempty"`
	Body     interface{}                   `json:"body,omitempty"`
	Routing  *RoutingDiagnosticSnapshot    `json:"routing,omitempty"`
	Response *RequestDetailResponsePayload `json:"response,omitempty"`
}

type RequestDetailResponsePayload struct {
	StatusCode    int         `json:"status_code"`
	ContentType   string      `json:"content_type,omitempty"`
	BodySize      int64       `json:"body_size"`
	Body          interface{} `json:"body,omitempty"`
	Truncated     bool        `json:"truncated,omitempty"`
	OmittedReason string      `json:"omitted_reason,omitempty"`
}

type RequestDetailResponseSnapshot struct {
	StatusCode    int
	ContentType   string
	BodySize      int64
	Body          []byte
	Truncated     bool
	OmittedReason string
}

type RequestDetailRuntimeStats struct {
	Written        uint64 `json:"written"`
	DroppedSuccess uint64 `json:"dropped_success"`
	DroppedFailure uint64 `json:"dropped_failure"`
	QueueSuccess   int    `json:"queue_success"`
	QueueFailure   int    `json:"queue_failure"`
	DiskSuspended  bool   `json:"disk_suspended"`
}

type queuedRequestDetail struct {
	detail   *model.RequestDetail
	body     []byte
	query    url.Values
	headers  map[string]string
	routing  *RoutingDiagnosticSnapshot
	response *RequestDetailResponseSnapshot
	failed   bool
}

type requestDetailWriter struct {
	failureQueue chan *queuedRequestDetail
	successQueue chan *queuedRequestDetail
	stop         chan struct{}
	done         chan struct{}
	stopOnce     sync.Once

	written        atomic.Uint64
	droppedSuccess atomic.Uint64
	droppedFailure atomic.Uint64
	diskSuspended  atomic.Bool

	storageBytes      int64
	lastStatsRefresh  time.Time
	lastRetentionRun  time.Time
	lastClickHouseTTL int
	lastDiskCheck     time.Time
	lastDiskLow       bool
}

var activeRequestDetailWriter atomic.Pointer[requestDetailWriter]

func SetRequestDetailFailure(c *gin.Context, failure *RequestDetailFailure) {
	if c == nil || failure == nil {
		return
	}
	c.Set(requestDetailFailureContextKey, failure)
}

func SetRequestDetailResponse(c *gin.Context, response *RequestDetailResponseSnapshot) {
	if c == nil || response == nil {
		return
	}
	c.Set(requestDetailResponseContextKey, response)
}

func CaptureRequestDetailFromContext(c *gin.Context) {
	if c == nil {
		return
	}
	failure, _ := c.Get(requestDetailFailureContextKey)
	typedFailure, _ := failure.(*RequestDetailFailure)
	CaptureRequestDetail(c, typedFailure)
}

func StartRequestDetailWriter() {
	if activeRequestDetailWriter.Load() != nil {
		return
	}
	writer := &requestDetailWriter{
		failureQueue: make(chan *queuedRequestDetail, requestDetailFailureQueueSize),
		successQueue: make(chan *queuedRequestDetail, requestDetailSuccessQueueSize),
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
	}
	if !activeRequestDetailWriter.CompareAndSwap(nil, writer) {
		return
	}
	go writer.run()
}

func StopRequestDetailWriter(ctx context.Context) {
	writer := activeRequestDetailWriter.Swap(nil)
	if writer == nil {
		return
	}
	writer.stopOnce.Do(func() { close(writer.stop) })
	select {
	case <-writer.done:
	case <-ctx.Done():
	}
}

func GetRequestDetailRuntimeStats() RequestDetailRuntimeStats {
	writer := activeRequestDetailWriter.Load()
	if writer == nil {
		return RequestDetailRuntimeStats{}
	}
	return RequestDetailRuntimeStats{
		Written:        writer.written.Load(),
		DroppedSuccess: writer.droppedSuccess.Load(),
		DroppedFailure: writer.droppedFailure.Load(),
		QueueSuccess:   len(writer.successQueue),
		QueueFailure:   len(writer.failureQueue),
		DiskSuspended:  writer.diskSuspended.Load(),
	}
}

// CaptureRequestDetail snapshots a completed logical request. It intentionally
// does no JSON parsing or compression on the request goroutine.
func CaptureRequestDetail(c *gin.Context, failure *RequestDetailFailure) {
	if c == nil {
		return
	}
	settings := request_detail_setting.GetSetting()
	statusCode := c.Writer.Status()
	failed := failure != nil || statusCode >= http.StatusBadRequest
	if !shouldCaptureRequestDetail(settings.Mode, failed) {
		return
	}
	if failure != nil && failure.StatusCode > 0 {
		statusCode = failure.StatusCode
	}
	if statusCode <= 0 {
		statusCode = http.StatusOK
	}

	requestID := c.GetString(common.RequestIdKey)
	if requestID == "" {
		requestID = common.NewRequestId()
	}
	group := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
	if group == "" {
		group = c.GetString("group")
	}
	outcome := model.RequestDetailOutcomeSuccess
	if failed {
		outcome = model.RequestDetailOutcomeFailed
	}

	detail := &model.RequestDetail{
		RequestID:  requestID,
		CreatedAt:  common.GetTimestamp(),
		UserID:     c.GetInt("id"),
		Username:   c.GetString("username"),
		TokenID:    c.GetInt("token_id"),
		TokenName:  c.GetString("token_name"),
		ModelName:  c.GetString("original_model"),
		Group:      group,
		Outcome:    outcome,
		StatusCode: statusCode,
	}
	if detail.ModelName == "" {
		detail.ModelName = common.GetContextKeyString(c, constant.ContextKeyOriginalModel)
	}
	if failure != nil {
		detail.ErrorType = boundedRequestDetailString(failure.ErrorType, 128)
		detail.ErrorCode = boundedRequestDetailString(failure.ErrorCode, 128)
		detail.ErrorMessage = boundedRequestDetailString(common.MaskSensitiveInfo(failure.Message), requestDetailMaxErrorBytes)
	}

	queued := &queuedRequestDetail{detail: detail, failed: failed}
	if response, exists := c.Get(requestDetailResponseContextKey); exists {
		queued.response, _ = response.(*RequestDetailResponseSnapshot)
	}
	if c.Request != nil {
		detail.Method = boundedRequestDetailString(c.Request.Method, 16)
		detail.ContentType = boundedRequestDetailString(c.Request.Header.Get("Content-Type"), 255)
		queued.headers = requestDetailHeaderSnapshot(c.Request.Header)
		if c.Request.URL != nil {
			detail.Path = boundedRequestDetailString(c.Request.URL.Path, 512)
			queued.query = cloneURLValues(c.Request.URL.Query())
		}
	}
	detail.IP = boundedRequestDetailString(c.ClientIP(), 64)
	queued.routing = requestDetailRoutingSnapshot(c, group, failed)

	if storageValue, ok := c.Get(common.KeyBodyStorage); ok {
		if storage, storageOK := storageValue.(common.BodyStorage); storageOK && storage != nil {
			detail.BodySize = storage.Size()
			if detail.BodySize > request_detail_setting.MaxBodyBytes {
				detail.BodyOmittedReason = "too_large"
			} else if detail.BodySize > 0 && isRequestDetailBodyTypeSupported(detail.ContentType) {
				body, err := storage.Bytes()
				if err == nil {
					queued.body = append([]byte(nil), body...)
				} else {
					detail.BodyOmittedReason = "read_failed"
				}
			} else if detail.BodySize > 0 {
				detail.BodyOmittedReason = "unsupported_content_type"
			}
		}
	}

	writer := activeRequestDetailWriter.Load()
	if writer == nil {
		return
	}
	writer.enqueue(queued)
}

func shouldCaptureRequestDetail(mode string, failed bool) bool {
	switch mode {
	case request_detail_setting.ModeAll:
		return true
	case request_detail_setting.ModeFailed:
		return failed
	default:
		return false
	}
}

func DecodeRequestDetailPayload(compressed []byte) (RequestDetailPayload, error) {
	if len(compressed) == 0 {
		return RequestDetailPayload{}, nil
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return RequestDetailPayload{}, err
	}
	decompressed, readErr := io.ReadAll(io.LimitReader(reader, requestDetailMaxDecodedPayloadBytes+1))
	closeErr := reader.Close()
	if readErr != nil {
		return RequestDetailPayload{}, readErr
	}
	if closeErr != nil {
		return RequestDetailPayload{}, closeErr
	}
	if len(decompressed) > requestDetailMaxDecodedPayloadBytes {
		return RequestDetailPayload{}, fmt.Errorf("request detail payload exceeds decoded size limit")
	}
	var payload RequestDetailPayload
	if err := common.Unmarshal(decompressed, &payload); err != nil {
		return RequestDetailPayload{}, err
	}
	return payload, nil
}

func (w *requestDetailWriter) enqueue(detail *queuedRequestDetail) {
	if detail.failed {
		select {
		case w.failureQueue <- detail:
		default:
			w.droppedFailure.Add(1)
		}
		return
	}
	select {
	case w.successQueue <- detail:
	default:
		w.droppedSuccess.Add(1)
	}
}

func (w *requestDetailWriter) run() {
	defer close(w.done)
	flushTicker := time.NewTicker(requestDetailFlushInterval)
	maintenanceTicker := time.NewTicker(requestDetailMaintenanceInterval)
	defer flushTicker.Stop()
	defer maintenanceTicker.Stop()

	w.refreshStorageStats(context.Background())
	w.runMaintenance(context.Background(), true, 0)
	batch := make([]*queuedRequestDetail, 0, requestDetailBatchSize)
	for {
		if len(batch) < requestDetailBatchSize {
			select {
			case detail := <-w.failureQueue:
				batch = append(batch, detail)
				continue
			default:
			}
		}

		select {
		case detail := <-w.failureQueue:
			batch = append(batch, detail)
		case detail := <-w.successQueue:
			batch = append(batch, detail)
		case <-flushTicker.C:
			w.flush(batch)
			batch = batch[:0]
		case <-maintenanceTicker.C:
			w.flush(batch)
			batch = batch[:0]
			w.runMaintenance(context.Background(), false, 0)
		case <-w.stop:
			batch = w.drainQueues(batch)
			w.flush(batch)
			return
		}
		if len(batch) >= requestDetailBatchSize {
			w.flush(batch)
			batch = batch[:0]
		}
	}
}

func (w *requestDetailWriter) drainQueues(batch []*queuedRequestDetail) []*queuedRequestDetail {
	for {
		select {
		case detail := <-w.failureQueue:
			batch = append(batch, detail)
		case detail := <-w.successQueue:
			batch = append(batch, detail)
		default:
			return batch
		}
	}
}

func (w *requestDetailWriter) flush(queuedBatch []*queuedRequestDetail) {
	if len(queuedBatch) == 0 {
		return
	}
	settings := request_detail_setting.GetSetting()
	if settings.Mode == request_detail_setting.ModeNone {
		w.dropBatch(queuedBatch)
		return
	}
	if settings.Mode == request_detail_setting.ModeFailed {
		failedOnly := queuedBatch[:0]
		for _, queued := range queuedBatch {
			if queued.failed {
				failedOnly = append(failedOnly, queued)
			} else {
				w.droppedSuccess.Add(1)
			}
		}
		queuedBatch = failedOnly
		if len(queuedBatch) == 0 {
			return
		}
	}
	if w.localDiskLow() {
		w.diskSuspended.Store(true)
		w.dropBatch(queuedBatch)
		return
	}
	w.diskSuspended.Store(false)

	sort.SliceStable(queuedBatch, func(i, j int) bool {
		return queuedBatch[i].failed && !queuedBatch[j].failed
	})
	details := make([]*model.RequestDetail, 0, len(queuedBatch))
	for _, queued := range queuedBatch {
		detail, err := prepareRequestDetail(queued)
		if err != nil {
			if queued.failed {
				w.droppedFailure.Add(1)
			} else {
				w.droppedSuccess.Add(1)
			}
			continue
		}
		details = append(details, detail)
	}
	if len(details) == 0 {
		return
	}

	maxBytes := int64(settings.MaxStorageMB) << 20
	projectedBytes := int64(0)
	for _, detail := range details {
		projectedBytes += detail.StorageBytes
	}
	if w.storageBytes+projectedBytes >= maxBytes*9/10 {
		w.runMaintenance(context.Background(), false, projectedBytes)
	}

	accepted := details[:0]
	for _, detail := range details {
		if w.storageBytes+detail.StorageBytes > maxBytes {
			if detail.Outcome == model.RequestDetailOutcomeFailed {
				w.droppedFailure.Add(1)
			} else {
				w.droppedSuccess.Add(1)
			}
			continue
		}
		accepted = append(accepted, detail)
		w.storageBytes += detail.StorageBytes
	}
	if len(accepted) == 0 {
		return
	}
	if err := model.CreateRequestDetails(accepted); err != nil {
		for _, detail := range accepted {
			w.storageBytes -= detail.StorageBytes
		}
		w.dropModelDetails(accepted)
		logger.LogWarn(context.Background(), fmt.Sprintf("request detail batch write failed: %v", err))
		return
	}
	w.written.Add(uint64(len(accepted)))
}

func (w *requestDetailWriter) runMaintenance(ctx context.Context, forceRetention bool, anticipatedBytes int64) {
	settings := request_detail_setting.GetSetting()
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) && w.lastClickHouseTTL != settings.RetentionDays {
		if err := model.SyncRequestDetailClickHouseTTL(settings.RetentionDays); err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("request detail ClickHouse TTL update failed: %v", err))
		} else {
			w.lastClickHouseTTL = settings.RetentionDays
		}
	}

	cutoff := int64(0)
	now := time.Now()
	if forceRetention || now.Sub(w.lastRetentionRun) >= requestDetailRetentionCheckInterval {
		cutoff = now.Add(-time.Duration(settings.RetentionDays) * 24 * time.Hour).Unix()
		w.lastRetentionRun = now
	}
	maxBytes := int64(settings.MaxStorageMB) << 20
	if w.lastStatsRefresh.IsZero() || now.Sub(w.lastStatsRefresh) >= requestDetailStorageRefreshInterval {
		w.refreshStorageStats(ctx)
	}
	targetBytes := int64(0)
	if w.storageBytes+anticipatedBytes >= maxBytes*9/10 {
		targetBytes = maxBytes * 8 / 10
	}
	if common.IsMasterNode && (cutoff > 0 || targetBytes > 0) {
		result, err := model.CleanupRequestDetails(ctx, cutoff, targetBytes, requestDetailCleanupBatchSize)
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("request detail cleanup failed: %v", err))
		} else if result.DeletedCount > 0 {
			logger.LogInfo(ctx, fmt.Sprintf("request detail cleanup removed %d records and %d bytes", result.DeletedCount, result.FreedBytes))
			w.refreshStorageStats(ctx)
		}
	}
}

func (w *requestDetailWriter) refreshStorageStats(ctx context.Context) {
	stats, err := model.GetRequestDetailStorageStats(ctx)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("request detail storage stats failed: %v", err))
		return
	}
	w.storageBytes = stats.StorageBytes
	w.lastStatsRefresh = time.Now()
}

func (w *requestDetailWriter) localDiskLow() bool {
	if !common.UsingLogDatabase(common.DatabaseTypeSQLite) {
		return false
	}
	if time.Since(w.lastDiskCheck) < 5*time.Second {
		return w.lastDiskLow
	}
	w.lastDiskCheck = time.Now()
	databasePath := strings.TrimPrefix(common.SQLitePath, "file:")
	if queryIndex := strings.IndexByte(databasePath, '?'); queryIndex >= 0 {
		databasePath = databasePath[:queryIndex]
	}
	if databasePath == "" || databasePath == ":memory:" {
		return false
	}
	directory := filepath.Dir(databasePath)
	if absoluteDirectory, err := filepath.Abs(directory); err == nil {
		directory = absoluteDirectory
	}
	if _, err := os.Stat(directory); err != nil {
		return false
	}
	disk := common.GetDiskSpaceInfoForPath(directory)
	if disk.Total == 0 {
		return false
	}
	w.lastDiskLow = disk.Free < requestDetailMinFreeDiskBytes || disk.Free*100 < disk.Total*5
	return w.lastDiskLow
}

func (w *requestDetailWriter) dropBatch(batch []*queuedRequestDetail) {
	for _, detail := range batch {
		if detail.failed {
			w.droppedFailure.Add(1)
		} else {
			w.droppedSuccess.Add(1)
		}
	}
}

func (w *requestDetailWriter) dropModelDetails(details []*model.RequestDetail) {
	for _, detail := range details {
		if detail.Outcome == model.RequestDetailOutcomeFailed {
			w.droppedFailure.Add(1)
		} else {
			w.droppedSuccess.Add(1)
		}
	}
}

func prepareRequestDetail(queued *queuedRequestDetail) (*model.RequestDetail, error) {
	payload := RequestDetailPayload{
		Headers: queued.headers,
		Query:   sanitizeRequestDetailValues(queued.query),
		Routing: queued.routing,
	}
	if len(queued.body) > 0 {
		body, err := parseRequestDetailBody(queued.detail.ContentType, queued.body)
		if err != nil {
			queued.detail.BodyOmittedReason = "parse_failed"
		} else {
			payload.Body = body
		}
	}
	if queued.response != nil {
		response := &RequestDetailResponsePayload{
			StatusCode:    queued.response.StatusCode,
			ContentType:   queued.response.ContentType,
			BodySize:      queued.response.BodySize,
			Truncated:     queued.response.Truncated,
			OmittedReason: queued.response.OmittedReason,
		}
		if len(queued.response.Body) > 0 {
			body, err := parseRequestDetailResponseBody(
				queued.response.ContentType,
				queued.response.Body,
				queued.response.Truncated,
			)
			if err != nil {
				if response.OmittedReason == "" {
					response.OmittedReason = "parse_failed"
				}
			} else {
				response.Body = body
			}
		}
		payload.Response = response
	}
	serialized, err := common.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var compressed bytes.Buffer
	compressor, err := gzip.NewWriterLevel(&compressed, gzip.BestSpeed)
	if err != nil {
		return nil, err
	}
	if _, err := compressor.Write(serialized); err != nil {
		_ = compressor.Close()
		return nil, err
	}
	if err := compressor.Close(); err != nil {
		return nil, err
	}
	queued.detail.Payload = compressed.Bytes()
	queued.detail.StorageBytes = int64(len(queued.detail.Payload)) + requestDetailStorageOverheadBytes +
		int64(len(queued.detail.ErrorMessage)+len(queued.detail.Path)+len(queued.detail.ModelName)+len(queued.detail.Username))
	return queued.detail, nil
}

func parseRequestDetailResponseBody(contentType string, body []byte, truncated bool) (interface{}, error) {
	mediaType := requestDetailMediaType(contentType)
	if (mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")) && !truncated {
		var value interface{}
		if err := common.Unmarshal(body, &value); err != nil {
			return nil, err
		}
		return redactRequestDetailValue("", value), nil
	}
	if !utf8.Valid(body) {
		return nil, fmt.Errorf("response body is not valid UTF-8")
	}
	return common.MaskSensitiveInfo(string(body)), nil
}

func parseRequestDetailBody(contentType string, body []byte) (interface{}, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}
	if mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") {
		var value interface{}
		if err := common.Unmarshal(body, &value); err != nil {
			return nil, err
		}
		return redactRequestDetailValue("", value), nil
	}
	if mediaType == "application/x-www-form-urlencoded" {
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}
		return sanitizeRequestDetailValues(values), nil
	}
	if mediaType == "multipart/form-data" {
		boundary := params["boundary"]
		if boundary == "" {
			return nil, fmt.Errorf("multipart boundary is missing")
		}
		reader := multipart.NewReader(bytes.NewReader(body), boundary)
		fields := make(map[string]interface{})
		files := make([]map[string]interface{}, 0)
		for {
			part, partErr := reader.NextPart()
			if partErr == io.EOF {
				break
			}
			if partErr != nil {
				return nil, partErr
			}
			content, readErr := io.ReadAll(io.LimitReader(part, request_detail_setting.MaxBodyBytes+1))
			_ = part.Close()
			if readErr != nil {
				return nil, readErr
			}
			if part.FileName() != "" {
				files = append(files, map[string]interface{}{
					"field":        part.FormName(),
					"filename":     part.FileName(),
					"content_type": part.Header.Get("Content-Type"),
					"size":         len(content),
				})
				continue
			}
			fields[part.FormName()] = redactRequestDetailValue(part.FormName(), string(content))
		}
		result := map[string]interface{}{"fields": fields}
		if len(files) > 0 {
			result["files"] = files
		}
		return result, nil
	}
	return nil, fmt.Errorf("unsupported content type")
}

func redactRequestDetailValue(key string, value interface{}) interface{} {
	if isSensitiveRequestDetailKey(key) {
		return "[REDACTED]"
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		redacted := make(map[string]interface{}, len(typed))
		for childKey, childValue := range typed {
			redacted[childKey] = redactRequestDetailValue(childKey, childValue)
		}
		return redacted
	case []interface{}:
		redacted := make([]interface{}, len(typed))
		for index, childValue := range typed {
			redacted[index] = redactRequestDetailValue("", childValue)
		}
		return redacted
	default:
		return value
	}
}

func sanitizeRequestDetailValues(values url.Values) map[string]interface{} {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]interface{}, len(values))
	for key, value := range values {
		if isSensitiveRequestDetailKey(key) {
			result[key] = "[REDACTED]"
			continue
		}
		result[key] = append([]string(nil), value...)
	}
	return result
}

func isSensitiveRequestDetailKey(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("-", "", "_", "", ".", "").Replace(strings.TrimSpace(key)))
	switch normalized {
	case "authorization", "cookie", "setcookie", "auth", "apikey", "key", "privatekey", "token",
		"bearertoken", "idtoken", "sessiontoken", "accesstoken", "refreshtoken", "password", "passwd",
		"secret", "clientsecret", "credential", "credentials":
		return true
	default:
		return strings.HasSuffix(normalized, "apikey") || strings.HasSuffix(normalized, "accesstoken") ||
			strings.HasSuffix(normalized, "refreshtoken") || strings.HasSuffix(normalized, "password") ||
			strings.HasSuffix(normalized, "secret")
	}
}

func isRequestDetailBodyTypeSupported(contentType string) bool {
	mediaType := requestDetailMediaType(contentType)
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") ||
		mediaType == "application/x-www-form-urlencoded" || mediaType == "multipart/form-data"
}

func IsRequestDetailResponseBodyTypeSupported(contentType string) bool {
	mediaType := requestDetailMediaType(contentType)
	if mediaType == "" {
		return true
	}
	return strings.HasPrefix(mediaType, "text/") || mediaType == "application/json" ||
		strings.HasSuffix(mediaType, "+json") || mediaType == "application/xml" ||
		strings.HasSuffix(mediaType, "+xml") || mediaType == "application/javascript" ||
		mediaType == "application/x-javascript" || mediaType == "application/x-ndjson" ||
		mediaType == "application/graphql-response+json"
}

func requestDetailMediaType(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}
	return strings.ToLower(mediaType)
}

func requestDetailHeaderSnapshot(header http.Header) map[string]string {
	result := make(map[string]string)
	for _, key := range []string{"Content-Type", "Accept", "User-Agent", "Content-Encoding"} {
		if value := header.Get(key); value != "" {
			result[strings.ToLower(key)] = boundedRequestDetailString(value, 1024)
		}
	}
	return result
}

func requestDetailRoutingSnapshot(c *gin.Context, finalGroup string, finalFailure bool) *RoutingDiagnosticSnapshot {
	if !finalFailure {
		return nil
	}
	return BuildRoutingDiagnosticSnapshot(c, finalGroup)
}

func cloneURLValues(values url.Values) url.Values {
	if len(values) == 0 {
		return nil
	}
	cloned := make(url.Values, len(values))
	for key, value := range values {
		cloned[key] = append([]string(nil), value...)
	}
	return cloned
}

func boundedRequestDetailString(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	for maxBytes > 0 && !utf8.RuneStart(value[maxBytes]) {
		maxBytes--
	}
	return value[:maxBytes]
}
