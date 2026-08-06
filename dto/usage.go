package dto

// UserQuotaData is the self-service usage aggregate. Internal channel
// dimensions are intentionally absent even though quota_data stores them.
type UserQuotaData struct {
	Id        int    `json:"id"`
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	ModelName string `json:"model_name"`
	CreatedAt int64  `json:"created_at"`
	UseGroup  string `json:"use_group"`
	TokenID   int    `json:"token_id"`
	NodeName  string `json:"node_name"`
	TokenUsed int    `json:"token_used"`
	Count     int    `json:"count"`
	Quota     int    `json:"quota"`
}
