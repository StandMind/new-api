package i18n

// Message keys for i18n translations
// Use these constants instead of hardcoded strings

// Common error messages
const (
	MsgInvalidParams      = "common.invalid_params"
	MsgDatabaseError      = "common.database_error"
	MsgRetryLater         = "common.retry_later"
	MsgGenerateFailed     = "common.generate_failed"
	MsgNotFound           = "common.not_found"
	MsgUnauthorized       = "common.unauthorized"
	MsgForbidden          = "common.forbidden"
	MsgInvalidId          = "common.invalid_id"
	MsgIdEmpty            = "common.id_empty"
	MsgFeatureDisabled    = "common.feature_disabled"
	MsgOperationSuccess   = "common.operation_success"
	MsgOperationFailed    = "common.operation_failed"
	MsgUpdateSuccess      = "common.update_success"
	MsgUpdateFailed       = "common.update_failed"
	MsgCreateSuccess      = "common.create_success"
	MsgCreateFailed       = "common.create_failed"
	MsgDeleteSuccess      = "common.delete_success"
	MsgDeleteFailed       = "common.delete_failed"
	MsgAlreadyExists      = "common.already_exists"
	MsgNameCannotBeEmpty  = "common.name_cannot_be_empty"
	MsgBatchTooMany       = "common.batch_too_many"
	MsgInvalidLanguage    = "common.invalid_language"
	MsgInvalidRequestBody = "common.invalid_request_body"
)

// Access policy messages
const (
	MsgAccessPolicyCodeInvalid          = "access_policy.code_invalid"
	MsgAccessPolicyDisplayNameRequired  = "access_policy.display_name_required"
	MsgAccessPolicyTopupRatioInvalid    = "access_policy.topup_ratio_invalid"
	MsgAccessPolicyRateLimitInvalid     = "access_policy.rate_limit_invalid"
	MsgAccessPolicyReservedUserLevel    = "access_policy.reserved_user_level"
	MsgAccessPolicyDefaultMustBeEnabled = "access_policy.default_must_be_enabled"
	MsgAccessPolicyDefaultRequired      = "access_policy.default_required"
	MsgAccessPolicyDeleteDefaultDenied  = "access_policy.delete_default_denied"
	MsgAccessPolicyUserLevelReferenced  = "access_policy.user_level_referenced"
	MsgAccessPolicyRouteGroupDuplicate  = "access_policy.route_group_duplicate"
	MsgAccessPolicyRouteGroupRetired    = "access_policy.route_group_retired"
	MsgAccessPolicyPriceRatioInvalid    = "access_policy.price_ratio_invalid"
	MsgAccessPolicyRouteGroupNotFound   = "access_policy.route_group_not_found"
	MsgAccessPolicyBaseRatioInvalid     = "access_policy.base_ratio_invalid"
	MsgAccessPolicyRouteGroupReferenced = "access_policy.route_group_referenced"
	MsgAccessPolicyRetiredGroupManaged  = "access_policy.retired_group_managed"
)

// Auth middleware messages
const (
	MsgAuthNotLoggedIn                   = "auth.not_logged_in"
	MsgAuthAccessTokenInvalid            = "auth.access_token_invalid"
	MsgAuthUserInfoInvalid               = "auth.user_info_invalid"
	MsgAuthUserIdNotProvided             = "auth.user_id_not_provided"
	MsgAuthUserIdFormatError             = "auth.user_id_format_error"
	MsgAuthUserIdMismatch                = "auth.user_id_mismatch"
	MsgAuthUserBanned                    = "auth.user_banned"
	MsgAuthInsufficientPrivilege         = "auth.insufficient_privilege"
	MsgAuthClientIPInvalid               = "auth.client_ip_invalid"
	MsgAuthIPNotAllowed                  = "auth.ip_not_allowed"
	MsgAuthRoutingModeInvalid            = "auth.routing_mode_invalid"
	MsgAuthLegacyAutoToken               = "auth.legacy_auto_token"
	MsgAuthNoSmartRoutingGroup           = "auth.no_smart_routing_group"
	MsgAuthGroupChainMissing             = "auth.group_chain_missing"
	MsgAuthGroupChainMismatch            = "auth.group_chain_mismatch"
	MsgAuthGroupChainContainsAuto        = "auth.group_chain_contains_auto"
	MsgAuthGroupChainDuplicate           = "auth.group_chain_duplicate"
	MsgAuthGroupNamedAccessDenied        = "auth.group_named_access_denied"
	MsgAuthGroupRetired                  = "auth.group_retired"
	MsgAuthSpecificChannelDenied         = "auth.specific_channel_denied"
	MsgAuthTurnstileTokenRequired        = "auth.turnstile_token_required"
	MsgAuthTurnstileFailed               = "auth.turnstile_failed"
	MsgAuthSecureVerificationRequired    = "auth.secure_verification_required"
	MsgAuthSecureVerificationInvalid     = "auth.secure_verification_invalid"
	MsgAuthSecureVerificationExpired     = "auth.secure_verification_expired"
	MsgAuthVerificationMethodUnavailable = "auth.verification_method_unavailable"
	MsgAuthVerificationMethodUnsupported = "auth.verification_method_unsupported"
	MsgAuthVerificationCodeRequired      = "auth.verification_code_required"
	MsgAuthVerificationFailed            = "auth.verification_failed"
	MsgAuthVerificationSuccess           = "auth.verification_success"
)

// Token related messages
const (
	MsgTokenNameTooLong          = "token.name_too_long"
	MsgTokenQuotaNegative        = "token.quota_negative"
	MsgTokenQuotaExceedMax       = "token.quota_exceed_max"
	MsgTokenGenerateFailed       = "token.generate_failed"
	MsgTokenGetInfoFailed        = "token.get_info_failed"
	MsgTokenExpiredCannotEnable  = "token.expired_cannot_enable"
	MsgTokenExhaustedCannotEable = "token.exhausted_cannot_enable"
	MsgTokenInvalid              = "token.invalid"
	MsgTokenNotProvided          = "token.not_provided"
	MsgTokenExpired              = "token.expired"
	MsgTokenExhausted            = "token.exhausted"
	MsgTokenStatusUnavailable    = "token.status_unavailable"
	MsgTokenDbError              = "token.db_error"
	MsgTokenCountLimitReached    = "token.count_limit_reached"
)

// Redemption related messages
const (
	MsgRedemptionNameLength        = "redemption.name_length"
	MsgRedemptionCountPositive     = "redemption.count_positive"
	MsgRedemptionCountMax          = "redemption.count_max"
	MsgRedemptionCreateFailed      = "redemption.create_failed"
	MsgRedemptionInvalid           = "redemption.invalid"
	MsgRedemptionUsed              = "redemption.used"
	MsgRedemptionExpired           = "redemption.expired"
	MsgRedemptionFailed            = "redemption.failed"
	MsgRedemptionNotProvided       = "redemption.not_provided"
	MsgRedemptionExpireTimeInvalid = "redemption.expire_time_invalid"
)

// User related messages
const (
	MsgUserPasswordLoginDisabled     = "user.password_login_disabled"
	MsgUserRegisterDisabled          = "user.register_disabled"
	MsgUserPasswordRegisterDisabled  = "user.password_register_disabled"
	MsgUserUsernameOrPasswordEmpty   = "user.username_or_password_empty"
	MsgUserUsernameOrPasswordError   = "user.username_or_password_error"
	MsgUserEmailOrPasswordEmpty      = "user.email_or_password_empty"
	MsgUserExists                    = "user.exists"
	MsgUserNotExists                 = "user.not_exists"
	MsgUserDisabled                  = "user.disabled"
	MsgUserSessionSaveFailed         = "user.session_save_failed"
	MsgUserRequire2FA                = "user.require_2fa"
	MsgUserEmailVerificationRequired = "user.email_verification_required"
	MsgUserVerificationCodeError     = "user.verification_code_error"
	MsgUserEmailAlreadyTaken         = "user.email_already_taken"
	MsgUserPasswordUnset             = "user.password_unset"
	MsgUserPasswordResetLinkInvalid  = "user.password_reset_link_invalid"
	MsgUserInputInvalid              = "user.input_invalid"
	MsgUserNoPermissionSameLevel     = "user.no_permission_same_level"
	MsgUserNoPermissionHigherLevel   = "user.no_permission_higher_level"
	MsgUserCannotCreateHigherLevel   = "user.cannot_create_higher_level"
	MsgUserCannotDeleteRootUser      = "user.cannot_delete_root_user"
	MsgUserCannotDisableRootUser     = "user.cannot_disable_root_user"
	MsgUserCannotDemoteRootUser      = "user.cannot_demote_root_user"
	MsgUserAlreadyAdmin              = "user.already_admin"
	MsgUserAlreadyCommon             = "user.already_common"
	MsgUserAdminCannotPromote        = "user.admin_cannot_promote"
	MsgUserOriginalPasswordError     = "user.original_password_error"
	MsgUserInviteQuotaInsufficient   = "user.invite_quota_insufficient"
	MsgUserTransferQuotaMinimum      = "user.transfer_quota_minimum"
	MsgUserTransferSuccess           = "user.transfer_success"
	MsgUserTransferFailed            = "user.transfer_failed"
	MsgUserTopUpProcessing           = "user.topup_processing"
	MsgUserRegisterFailed            = "user.register_failed"
	MsgUserDefaultTokenFailed        = "user.default_token_failed"
	MsgUserAffCodeEmpty              = "user.aff_code_empty"
	MsgUserEmailEmpty                = "user.email_empty"
	MsgUserGitHubIdEmpty             = "user.github_id_empty"
	MsgUserDiscordIdEmpty            = "user.discord_id_empty"
	MsgUserOidcIdEmpty               = "user.oidc_id_empty"
	MsgUserWeChatIdEmpty             = "user.wechat_id_empty"
	MsgUserTelegramIdEmpty           = "user.telegram_id_empty"
	MsgUserTelegramNotBound          = "user.telegram_not_bound"
	MsgUserLinuxDOIdEmpty            = "user.linux_do_id_empty"
	MsgUserQuotaChangeZero           = "user.quota_change_zero"
	MsgUserLevelUnavailable          = "user.level_unavailable"
	MsgUserGroupFieldDeprecated      = "user.group_field_deprecated"
	MsgUserEmailDomainNotAllowed     = "user.email_domain_not_allowed"
	MsgUserEmailAliasNotAllowed      = "user.email_alias_not_allowed"
)

// Quota related messages
const (
	MsgQuotaNegative        = "quota.negative"
	MsgQuotaExceedMax       = "quota.exceed_max"
	MsgQuotaInsufficient    = "quota.insufficient"
	MsgQuotaWarningInvalid  = "quota.warning_invalid"
	MsgQuotaThresholdGtZero = "quota.threshold_gt_zero"
)

// Subscription related messages
const (
	MsgSubscriptionNotEnabled       = "subscription.not_enabled"
	MsgSubscriptionTitleEmpty       = "subscription.title_empty"
	MsgSubscriptionPriceNegative    = "subscription.price_negative"
	MsgSubscriptionPriceMax         = "subscription.price_max"
	MsgSubscriptionPurchaseLimitNeg = "subscription.purchase_limit_negative"
	MsgSubscriptionQuotaNegative    = "subscription.quota_negative"
	MsgSubscriptionGroupNotExists   = "subscription.group_not_exists"
	MsgSubscriptionResetCycleGtZero = "subscription.reset_cycle_gt_zero"
	MsgSubscriptionPurchaseMax      = "subscription.purchase_max"
	MsgSubscriptionInvalidId        = "subscription.invalid_id"
	MsgSubscriptionInvalidUserId    = "subscription.invalid_user_id"
)

// Payment related messages
const (
	MsgPaymentNotConfigured         = "payment.not_configured"
	MsgPaymentMethodNotExists       = "payment.method_not_exists"
	MsgPaymentCallbackError         = "payment.callback_error"
	MsgPaymentCreateFailed          = "payment.create_failed"
	MsgPaymentStartFailed           = "payment.start_failed"
	MsgPaymentAmountTooLow          = "payment.amount_too_low"
	MsgPaymentStripeNotConfig       = "payment.stripe_not_configured"
	MsgPaymentWebhookNotConfig      = "payment.webhook_not_configured"
	MsgPaymentPriceIdNotConfig      = "payment.price_id_not_configured"
	MsgPaymentCreemNotConfig        = "payment.creem_not_configured"
	MsgPaymentComplianceRequired    = "payment.compliance_required"
	MsgPaymentSessionRequired       = "payment.session_required"
	MsgPaymentConfirmCompliance     = "payment.confirm_compliance"
	MsgPaymentWaffoNotConfigured    = "payment.waffo_not_configured"
	MsgPaymentProductNotConfig      = "payment.product_not_configured"
	MsgPaymentCustomerEmailRequired = "payment.customer_email_required"
)

// Topup related messages
const (
	MsgTopupNotProvided    = "topup.not_provided"
	MsgTopupOrderNotExists = "topup.order_not_exists"
	MsgTopupOrderStatus    = "topup.order_status"
	MsgTopupFailed         = "topup.failed"
	MsgTopupInvalidQuota   = "topup.invalid_quota"
	MsgTopupAmountMinimum  = "topup.amount_minimum"
	MsgTopupAmountMaximum  = "topup.amount_maximum"
)

// Channel related messages
const (
	MsgChannelNotExists                  = "channel.not_exists"
	MsgChannelIdFormatError              = "channel.id_format_error"
	MsgChannelNoAvailableKey             = "channel.no_available_key"
	MsgChannelGetListFailed              = "channel.get_list_failed"
	MsgChannelGetTagsFailed              = "channel.get_tags_failed"
	MsgChannelGetKeyFailed               = "channel.get_key_failed"
	MsgChannelGetOllamaFailed            = "channel.get_ollama_failed"
	MsgChannelQueryFailed                = "channel.query_failed"
	MsgChannelNoValidUpstream            = "channel.no_valid_upstream"
	MsgChannelUpstreamSaturated          = "channel.upstream_saturated"
	MsgChannelGetAvailableFailed         = "channel.get_available_failed"
	MsgChannelTagRequired                = "channel.tag_required"
	MsgChannelNotMultiKey                = "channel.not_multi_key"
	MsgChannelKeyIndexRequired           = "channel.key_index_required"
	MsgChannelKeyIndexOutOfRange         = "channel.key_index_out_of_range"
	MsgChannelKeyDisabled                = "channel.key_disabled"
	MsgChannelKeyEnabled                 = "channel.key_enabled"
	MsgChannelKeyDeleted                 = "channel.key_deleted"
	MsgChannelKeysEnabled                = "channel.keys_enabled"
	MsgChannelKeysDisabled               = "channel.keys_disabled"
	MsgChannelKeysAutoDeleted            = "channel.keys_auto_deleted"
	MsgChannelNoKeysToDisable            = "channel.no_keys_to_disable"
	MsgChannelLastKeyDeleteDenied        = "channel.last_key_delete_denied"
	MsgChannelNoAutoDisabledKeys         = "channel.no_auto_disabled_keys"
	MsgChannelUnsupportedAction          = "channel.unsupported_action"
	MsgChannelOnlyOllama                 = "channel.only_ollama"
	MsgChannelOllamaPullSuccess          = "channel.ollama_pull_success"
	MsgChannelOllamaDeleteSuccess        = "channel.ollama_delete_success"
	MsgChannelTaskRunning                = "channel.task_running"
	MsgChannelMultiKeyBalanceUnsupported = "channel.multi_key_balance_unsupported"
)

// Initial setup messages
const (
	MsgSetupAlreadyCompleted      = "setup.already_completed"
	MsgSetupUsernameTooLong       = "setup.username_too_long"
	MsgSetupPasswordMismatch      = "setup.password_mismatch"
	MsgSetupPasswordTooShort      = "setup.password_too_short"
	MsgSetupInitializationFailed  = "setup.initialization_failed"
	MsgSetupInitializationSuccess = "setup.initialization_success"
)

// Model related messages
const (
	MsgModelNameEmpty     = "model.name_empty"
	MsgModelNameExists    = "model.name_exists"
	MsgModelIdMissing     = "model.id_missing"
	MsgModelGetListFailed = "model.get_list_failed"
	MsgModelGetFailed     = "model.get_failed"
	MsgModelResetSuccess  = "model.reset_success"
)

// Vendor related messages
const (
	MsgVendorNameEmpty  = "vendor.name_empty"
	MsgVendorNameExists = "vendor.name_exists"
	MsgVendorIdMissing  = "vendor.id_missing"
)

// Group related messages
const (
	MsgGroupNameTypeEmpty = "group.name_type_empty"
	MsgGroupNameExists    = "group.name_exists"
	MsgGroupIdMissing     = "group.id_missing"
)

// Checkin related messages
const (
	MsgCheckinDisabled     = "checkin.disabled"
	MsgCheckinAlreadyToday = "checkin.already_today"
	MsgCheckinFailed       = "checkin.failed"
	MsgCheckinQuotaFailed  = "checkin.quota_failed"
	MsgCheckinSuccess      = "checkin.success"
)

// Usage data messages
const (
	MsgUsageInvalidStartTimestamp = "usage.invalid_start_timestamp"
	MsgUsageInvalidEndTimestamp   = "usage.invalid_end_timestamp"
	MsgUsageInvalidTimeRange      = "usage.invalid_time_range"
	MsgUsageTimeRangeTooLong      = "usage.time_range_too_long"
)

// Request detail messages
const (
	MsgRequestDetailInvalidOutcome = "request_detail.invalid_outcome"
	MsgRequestDetailIDRequired     = "request_detail.id_required"
	MsgRequestDetailNotFound       = "request_detail.not_found"
)

// Passkey related messages
const (
	MsgPasskeyCreateFailed                 = "passkey.create_failed"
	MsgPasskeyLoginAbnormal                = "passkey.login_abnormal"
	MsgPasskeyUpdateFailed                 = "passkey.update_failed"
	MsgPasskeyInvalidUserId                = "passkey.invalid_user_id"
	MsgPasskeyVerifyFailed                 = "passkey.verify_failed"
	MsgPasskeySecurityVerificationRequired = "passkey.security_verification_required"
	MsgPasskeyMatchingVerificationRequired = "passkey.matching_verification_required"
	MsgPasskeyDisabled                     = "passkey.disabled"
	MsgPasskeyRegistered                   = "passkey.registered"
	MsgPasskeyUnbound                      = "passkey.unbound"
	MsgPasskeyNotBound                     = "passkey.not_bound"
	MsgPasskeyReset                        = "passkey.reset"
	MsgPasskeyVerified                     = "passkey.verified"
	MsgPasskeySessionInvalid               = "passkey.session_invalid"
)

// 2FA related messages
const (
	MsgTwoFANotEnabled            = "twofa.not_enabled"
	MsgTwoFAUserIdEmpty           = "twofa.user_id_empty"
	MsgTwoFAAlreadyExists         = "twofa.already_exists"
	MsgTwoFARecordIdEmpty         = "twofa.record_id_empty"
	MsgTwoFACodeInvalid           = "twofa.code_invalid"
	MsgTwoFAAlreadyEnabled        = "twofa.already_enabled"
	MsgTwoFASecretGenerateFailed  = "twofa.secret_generate_failed"
	MsgTwoFABackupGenerateFailed  = "twofa.backup_generate_failed"
	MsgTwoFABackupSaveFailed      = "twofa.backup_save_failed"
	MsgTwoFASetupSuccess          = "twofa.setup_success"
	MsgTwoFASetupRequired         = "twofa.setup_required"
	MsgTwoFAEnableSuccess         = "twofa.enable_success"
	MsgTwoFADisableSuccess        = "twofa.disable_success"
	MsgTwoFABackupRegenerated     = "twofa.backup_regenerated"
	MsgTwoFASessionExpired        = "twofa.session_expired"
	MsgTwoFASessionInvalid        = "twofa.session_invalid"
	MsgTwoFAAdminPermissionDenied = "twofa.admin_permission_denied"
	MsgTwoFAAdminDisabled         = "twofa.admin_disabled"
)

// Rate limit related messages
const (
	MsgRateLimitReached      = "rate_limit.reached"
	MsgRateLimitTotalReached = "rate_limit.total_reached"
	MsgRateLimitEmailReached = "rate_limit.email_reached"
	MsgRateLimitEmailWait    = "rate_limit.email_wait"
)

// Relay error messages
const (
	MsgRelayInvalidRequest            = "relay.invalid_request"
	MsgRelaySensitiveWordsDetected    = "relay.sensitive_words_detected"
	MsgRelayCountTokenFailed          = "relay.count_token_failed"
	MsgRelayModelPriceError           = "relay.model_price_error"
	MsgRelayInvalidAPIType            = "relay.invalid_api_type"
	MsgRelayUpstreamRequestFailed     = "relay.upstream_request_failed"
	MsgRelayGetChannelFailed          = "relay.get_channel_failed"
	MsgRelayChannelConfigurationError = "relay.channel_configuration_error"
	MsgRelayChannelKeyInvalid         = "relay.channel_key_invalid"
	MsgRelayUpstreamTimeout           = "relay.upstream_timeout"
	MsgRelayRequestConversionFailed   = "relay.request_conversion_failed"
	MsgRelayInvalidUpstreamResponse   = "relay.invalid_upstream_response"
	MsgRelayModelNotFound             = "relay.model_not_found"
	MsgRelayPromptBlocked             = "relay.prompt_blocked"
	MsgRelayInternalPanic             = "relay.internal_panic"
	MsgRelayAPINotImplemented         = "relay.api_not_implemented"
	MsgRelayInvalidURL                = "relay.invalid_url"
	MsgRelayTaskUnavailable           = "relay.task_unavailable"
)

// Setting related messages
const (
	MsgSettingInvalidType                = "setting.invalid_type"
	MsgSettingWebhookEmpty               = "setting.webhook_empty"
	MsgSettingWebhookInvalid             = "setting.webhook_invalid"
	MsgSettingEmailInvalid               = "setting.email_invalid"
	MsgSettingBarkUrlEmpty               = "setting.bark_url_empty"
	MsgSettingBarkUrlInvalid             = "setting.bark_url_invalid"
	MsgSettingGotifyUrlEmpty             = "setting.gotify_url_empty"
	MsgSettingGotifyTokenEmpty           = "setting.gotify_token_empty"
	MsgSettingGotifyUrlInvalid           = "setting.gotify_url_invalid"
	MsgSettingUrlMustHttp                = "setting.url_must_http"
	MsgSettingSaved                      = "setting.saved"
	MsgSettingDeprecated                 = "setting.deprecated"
	MsgSettingComplianceReadOnly         = "setting.compliance_read_only"
	MsgSettingAccessPolicyMigrated       = "setting.access_policy_migrated"
	MsgSettingGitHubOAuthConfigRequired  = "setting.github_oauth_config_required"
	MsgSettingDiscordOAuthConfigRequired = "setting.discord_oauth_config_required"
	MsgSettingOIDCConfigRequired         = "setting.oidc_config_required"
	MsgSettingLinuxDOConfigRequired      = "setting.linuxdo_config_required"
	MsgSettingEmailDomainRequired        = "setting.email_domain_required"
	MsgSettingWeChatConfigRequired       = "setting.wechat_config_required"
	MsgSettingTurnstileConfigRequired    = "setting.turnstile_config_required"
	MsgSettingTelegramConfigRequired     = "setting.telegram_config_required"
	MsgSettingThemeInvalid               = "setting.theme_invalid"
	MsgSettingRoutingPriorityInvalid     = "setting.routing_priority_invalid"
	MsgSettingValueInvalid               = "setting.value_invalid"
)

// Deployment related messages (io.net)
const (
	MsgDeploymentNotEnabled     = "deployment.not_enabled"
	MsgDeploymentIdRequired     = "deployment.id_required"
	MsgDeploymentContainerIdReq = "deployment.container_id_required"
	MsgDeploymentNameEmpty      = "deployment.name_empty"
	MsgDeploymentNameTaken      = "deployment.name_taken"
	MsgDeploymentHardwareIdReq  = "deployment.hardware_id_required"
	MsgDeploymentHardwareInvId  = "deployment.hardware_invalid_id"
	MsgDeploymentApiKeyRequired = "deployment.api_key_required"
	MsgDeploymentInvalidPayload = "deployment.invalid_payload"
	MsgDeploymentNotFound       = "deployment.not_found"
	MsgDeploymentNameRequired   = "deployment.name_required"
)

// Performance related messages
const (
	MsgPerfDiskCacheCleared    = "performance.disk_cache_cleared"
	MsgPerfStatsReset          = "performance.stats_reset"
	MsgPerfGcExecuted          = "performance.gc_executed"
	MsgPerfCleanupModeInvalid  = "performance.cleanup_mode_invalid"
	MsgPerfCleanupValueInvalid = "performance.cleanup_value_invalid"
	MsgPerfLogDirMissing       = "performance.log_directory_missing"
	MsgPerfCleanupPartial      = "performance.cleanup_partial"
)

// System instance messages
const (
	MsgSystemNodeNameRequired = "system.node_name_required"
	MsgSystemInstanceNotStale = "system.instance_not_stale"
)

// Ability related messages
const (
	MsgAbilityDbCorrupted   = "ability.db_corrupted"
	MsgAbilityRepairRunning = "ability.repair_running"
)

// OAuth related messages
const (
	MsgOAuthInvalidCode     = "oauth.invalid_code"
	MsgOAuthGetUserErr      = "oauth.get_user_error"
	MsgOAuthAccountUsed     = "oauth.account_used"
	MsgOAuthUnknownProvider = "oauth.unknown_provider"
	MsgOAuthStateInvalid    = "oauth.state_invalid"
	MsgOAuthNotEnabled      = "oauth.not_enabled"
	MsgOAuthUserDeleted     = "oauth.user_deleted"
	MsgOAuthUserBanned      = "oauth.user_banned"
	MsgOAuthBindSuccess     = "oauth.bind_success"
	MsgOAuthAlreadyBound    = "oauth.already_bound"
	MsgOAuthConnectFailed   = "oauth.connect_failed"
	MsgOAuthTokenFailed     = "oauth.token_failed"
	MsgOAuthUserInfoEmpty   = "oauth.user_info_empty"
	MsgOAuthTrustLevelLow   = "oauth.trust_level_low"
)

// Model layer error messages (for translation in controller)
const (
	MsgRedeemFailed          = "redeem.failed"
	MsgCreateDefaultTokenErr = "user.create_default_token_error"
	MsgUuidDuplicate         = "common.uuid_duplicate"
	MsgInvalidInput          = "common.invalid_input"
)

// Distributor related messages
const (
	MsgDistributorInvalidRequest            = "distributor.invalid_request"
	MsgDistributorInvalidChannelId          = "distributor.invalid_channel_id"
	MsgDistributorChannelDisabled           = "distributor.channel_disabled"
	MsgDistributorAffinityChannelDisabled   = "distributor.affinity_channel_disabled"
	MsgDistributorTokenNoModelAccess        = "distributor.token_no_model_access"
	MsgDistributorTokenModelForbidden       = "distributor.token_model_forbidden"
	MsgDistributorModelNameRequired         = "distributor.model_name_required"
	MsgDistributorInvalidPlayground         = "distributor.invalid_playground_request"
	MsgDistributorGroupAccessDenied         = "distributor.group_access_denied"
	MsgDistributorGetChannelFailed          = "distributor.get_channel_failed"
	MsgDistributorNoAvailableChannel        = "distributor.no_available_channel"
	MsgDistributorInvalidMidjourney         = "distributor.invalid_midjourney_request"
	MsgDistributorInvalidParseModel         = "distributor.invalid_request_parse_model"
	MsgDistributorPlaygroundGroupDeprecated = "distributor.playground_group_deprecated"
	MsgDistributorPlaygroundRouteConflict   = "distributor.playground_route_conflict"
	MsgDistributorPlaygroundPriorityInvalid = "distributor.playground_priority_invalid"
	MsgDistributorNoSmartRoutingGroup       = "distributor.no_smart_routing_group"
)

// Jimeng request adapter messages
const (
	MsgJimengActionRequired = "jimeng.action_required"
	MsgJimengInvalidBody    = "jimeng.invalid_body"
	MsgJimengMarshalFailed  = "jimeng.marshal_failed"
	MsgJimengTaskIDRequired = "jimeng.task_id_required"
)

// Custom OAuth provider related messages
const (
	MsgCustomOAuthNotFound               = "custom_oauth.not_found"
	MsgCustomOAuthSlugEmpty              = "custom_oauth.slug_empty"
	MsgCustomOAuthSlugExists             = "custom_oauth.slug_exists"
	MsgCustomOAuthNameEmpty              = "custom_oauth.name_empty"
	MsgCustomOAuthHasBindings            = "custom_oauth.has_bindings"
	MsgCustomOAuthBindingNotFound        = "custom_oauth.binding_not_found"
	MsgCustomOAuthProviderIdInvalid      = "custom_oauth.provider_id_field_invalid"
	MsgCustomOAuthDiscoveryURLRequired   = "custom_oauth.discovery_url_required"
	MsgCustomOAuthDiscoveryURLInvalid    = "custom_oauth.discovery_url_invalid"
	MsgCustomOAuthDiscoveryRequestFailed = "custom_oauth.discovery_request_failed"
	MsgCustomOAuthDiscoveryFetchFailed   = "custom_oauth.discovery_fetch_failed"
	MsgCustomOAuthDiscoveryParseFailed   = "custom_oauth.discovery_parse_failed"
	MsgCustomOAuthBuiltinSlugConflict    = "custom_oauth.builtin_slug_conflict"
	MsgCustomOAuthBindingCheckFailed     = "custom_oauth.binding_check_failed"
)
