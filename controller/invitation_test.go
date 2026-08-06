package controller

import (
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type invitationRewardsResponse struct {
	Items []model.UserInvitationReward `json:"items"`
	Total int                          `json:"total"`
}

func setupInvitationControllerTestDB(t *testing.T) {
	t.Helper()
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Option{},
		&model.TopUp{},
		&model.InvitationTopupReward{},
	))
}

func TestGetInvitationRewardsReturnsOnlyPublicRewardFields(t *testing.T) {
	setupInvitationControllerTestDB(t)
	require.NoError(t, model.DB.Create(&model.User{
		Id:       7101,
		Username: "reward-owner",
		Email:    "owner@example.com",
		AffCode:  "owner-code",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, model.DB.Create(&model.User{
		Id:        7102,
		Username:  "Tys756",
		Email:     "invitee@example.com",
		AffCode:   "invitee-code",
		InviterId: 7101,
		Status:    common.UserStatusEnabled,
	}).Error)
	topUp := model.TopUp{
		UserId:          7102,
		Amount:          10,
		Money:           10,
		TradeNo:         "private-order-number",
		PaymentMethod:   model.PaymentProviderStripe,
		PaymentProvider: model.PaymentProviderStripe,
		CreateTime:      common.GetTimestamp(),
		CompleteTime:    common.GetTimestamp(),
		Status:          common.TopUpStatusSuccess,
	}
	require.NoError(t, model.DB.Create(&topUp).Error)
	require.NoError(t, model.DB.Create(&model.InvitationTopupReward{
		TopUpId:       topUp.Id,
		InviterId:     7101,
		InviteeId:     7102,
		TopupOrdinal:  1,
		CreditedQuota: 5000000,
		RebateBps:     500,
		RewardQuota:   250000,
		CreatedAt:     common.GetTimestamp(),
	}).Error)

	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/user/aff/rewards?p=1&page_size=20", nil, 7101)
	GetInvitationRewards(ctx)

	response := decodeAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	var page invitationRewardsResponse
	require.NoError(t, common.Unmarshal(response.Data, &page))
	assert.Equal(t, 1, page.Total)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "T***6", page.Items[0].Invitee)

	body := recorder.Body.String()
	for _, privateValue := range []string{
		"private-order-number",
		"invitee@example.com",
		"owner@example.com",
		model.PaymentProviderStripe,
		`"top_up_id"`,
		`"inviter_id"`,
		`"invitee_id"`,
	} {
		assert.False(t, strings.Contains(body, privateValue), "response leaked %q", privateValue)
	}
}

func TestGetInvitationRewardsRejectsNegativePageSize(t *testing.T) {
	setupInvitationControllerTestDB(t)

	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/user/aff/rewards?page_size=-1", nil, 7101)
	GetInvitationRewards(ctx)

	response := decodeAPIResponse(t, recorder)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.False(t, response.Success)
}
