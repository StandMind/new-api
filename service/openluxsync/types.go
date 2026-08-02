package openluxsync

import (
	"fmt"
	"net/http"
)

const (
	Endpoint       = "https://api.openlux.ai/api/pricing"
	SaleMultiplier = "1.3"
)

type ServiceError struct {
	Status  int
	Code    string
	Message string
	Err     error
}

func (err *ServiceError) Error() string {
	if err.Err == nil {
		return err.Message
	}
	return fmt.Sprintf("%s: %v", err.Message, err.Err)
}

func (err *ServiceError) Unwrap() error {
	return err.Err
}

func badRequest(message string) error {
	return &ServiceError{Status: http.StatusBadRequest, Code: "invalid_request", Message: message}
}

func conflict(message string) error {
	return &ServiceError{Status: http.StatusConflict, Code: "stale_preview", Message: message}
}

func unprocessable(message string) error {
	return &ServiceError{Status: http.StatusUnprocessableEntity, Code: "blocked_change", Message: message}
}

type ChannelSummary struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	LocalGroup  string `json:"local_group"`
	Status      int    `json:"status"`
	ModelCount  int    `json:"model_count"`
	BoundSource string `json:"bound_source_group,omitempty"`
}

type BindingView struct {
	SourceGroup       string           `json:"source_group"`
	LocalGroup        string           `json:"local_group"`
	Channels          []ChannelSummary `json:"channels"`
	MixedChannelCount int              `json:"mixed_channel_count"`
	Healthy           bool             `json:"healthy"`
	Issues            []string         `json:"issues"`
	Candidate         bool             `json:"candidate,omitempty"`
}

type SourceGroupView struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type BindingsResponse struct {
	BindingRevision int64             `json:"binding_revision"`
	Bindings        []BindingView     `json:"bindings"`
	Candidates      []BindingView     `json:"candidates"`
	SourceGroups    []SourceGroupView `json:"source_groups"`
	OpenLuxChannels []ChannelSummary  `json:"openlux_channels"`
}

type SaveBinding struct {
	SourceGroup string `json:"source_group"`
	ChannelIDs  []int  `json:"channel_ids"`
}

type SaveBindingsRequest struct {
	ExpectedRevision int64         `json:"expected_revision"`
	Bindings         []SaveBinding `json:"bindings"`
}

type Notice struct {
	Kind        string `json:"kind"`
	Model       string `json:"model,omitempty"`
	SourceGroup string `json:"source_group,omitempty"`
	LocalGroup  string `json:"local_group,omitempty"`
	Message     string `json:"message"`
}

type ChangeDetail struct {
	Field   string `json:"field"`
	Current string `json:"current,omitempty"`
	Target  string `json:"target,omitempty"`
}

type Change struct {
	ID                string         `json:"id"`
	Kind              string         `json:"kind"`
	Model             string         `json:"model,omitempty"`
	SourceGroup       string         `json:"source_group,omitempty"`
	LocalGroup        string         `json:"local_group,omitempty"`
	CurrentValue      string         `json:"current_value,omitempty"`
	TargetValue       string         `json:"target_value,omitempty"`
	CurrentPrice      string         `json:"current_price,omitempty"`
	TargetPrice       string         `json:"target_price,omitempty"`
	PriceUnit         string         `json:"price_unit,omitempty"`
	PercentChange     string         `json:"percent_change,omitempty"`
	Actionable        bool           `json:"actionable"`
	Destructive       bool           `json:"destructive"`
	Requires          []string       `json:"requires"`
	BlockedReasons    []string       `json:"blocked_reasons"`
	AffectedGroups    []string       `json:"affected_groups,omitempty"`
	MixedChannelCount int            `json:"mixed_channel_count,omitempty"`
	RemoveMode        string         `json:"remove_mode,omitempty"`
	Details           []ChangeDetail `json:"details,omitempty"`
	mutations         []optionMutation
	abilityRemovals   []abilityRemoval
	deleteChannelIDs  []int
	deleteBindings    []string
}

type PreviewSummary struct {
	Actionable int `json:"actionable"`
	Blocked    int `json:"blocked"`
	Notices    int `json:"notices"`
	Price      int `json:"price"`
	Removal    int `json:"removal"`
}

type PreviewResponse struct {
	SourceHash       string         `json:"source_hash"`
	LocalFingerprint string         `json:"local_fingerprint"`
	BindingRevision  int64          `json:"binding_revision"`
	FetchedAt        int64          `json:"fetched_at"`
	Changes          []Change       `json:"changes"`
	Notices          []Notice       `json:"notices"`
	Summary          PreviewSummary `json:"summary"`
}

type ApplyRequest struct {
	SourceHash       string   `json:"source_hash"`
	LocalFingerprint string   `json:"local_fingerprint"`
	BindingRevision  int64    `json:"binding_revision"`
	ChangeIDs        []string `json:"change_ids"`
}

type ApplyResponse struct {
	AppliedCount    int      `json:"applied_count"`
	UpdatedModels   []string `json:"updated_models"`
	UpdatedGroups   []string `json:"updated_groups"`
	RemovedChannels []int    `json:"removed_channels"`
	BindingRevision int64    `json:"binding_revision"`
	SourceHash      string   `json:"source_hash"`
}

type optionMutation struct {
	Key    string
	Action string
	Group  string
	Model  string
	Value  string
}

type abilityRemoval struct {
	ChannelID int
	Group     string
	Model     string
}
