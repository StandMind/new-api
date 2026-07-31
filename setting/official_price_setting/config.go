package official_price_setting

import (
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

const (
	OptionKey                    = "official_price_setting.model_prices"
	UnitUSDPerMillionInputTokens = "usd_per_million_input_tokens"
	UnitUSDPerRequest            = "usd_per_request"
)

type PriceTier struct {
	UpToInputTokens *int64  `json:"up_to_input_tokens,omitempty"`
	Price           float64 `json:"price"`
}

type ModelPrice struct {
	Unit        string      `json:"unit"`
	SourceModel string      `json:"source_model,omitempty"`
	SourceURL   string      `json:"source_url,omitempty"`
	VerifiedAt  string      `json:"verified_at,omitempty"`
	Notes       string      `json:"notes,omitempty"`
	Tiers       []PriceTier `json:"tiers"`
}

type OfficialPriceSetting struct {
	ModelPrices map[string]ModelPrice `json:"model_prices"`
}

var officialPriceSetting = OfficialPriceSetting{
	ModelPrices: make(map[string]ModelPrice),
}

func init() {
	config.GlobalConfig.Register("official_price_setting", &officialPriceSetting)
}

func ValidateModelPricesJSON(raw string) error {
	prices := make(map[string]ModelPrice)
	if err := common.UnmarshalJsonStr(raw, &prices); err != nil {
		return fmt.Errorf("official model prices must be a JSON object: %w", err)
	}
	for modelName, price := range prices {
		if strings.TrimSpace(modelName) == "" {
			return fmt.Errorf("official model price contains an empty model name")
		}
		if err := validateModelPrice(modelName, price); err != nil {
			return err
		}
	}
	return nil
}

func validateModelPrice(modelName string, modelPrice ModelPrice) error {
	switch modelPrice.Unit {
	case UnitUSDPerMillionInputTokens, UnitUSDPerRequest:
	default:
		return fmt.Errorf("official price for %s has unsupported unit %q", modelName, modelPrice.Unit)
	}
	if len(modelPrice.Tiers) == 0 {
		return fmt.Errorf("official price for %s must contain at least one tier", modelName)
	}
	if modelPrice.SourceURL != "" {
		parsedURL, err := url.ParseRequestURI(modelPrice.SourceURL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
			return fmt.Errorf("official price for %s has an invalid source URL", modelName)
		}
	}
	if modelPrice.VerifiedAt != "" {
		if _, err := time.Parse("2006-01-02", modelPrice.VerifiedAt); err != nil {
			return fmt.Errorf("official price for %s has an invalid verification date", modelName)
		}
	}

	if modelPrice.Unit == UnitUSDPerRequest && len(modelPrice.Tiers) != 1 {
		return fmt.Errorf("per-request official price for %s must contain exactly one tier", modelName)
	}
	var previousLimit int64
	for index, tier := range modelPrice.Tiers {
		if math.IsNaN(tier.Price) || math.IsInf(tier.Price, 0) || tier.Price <= 0 {
			return fmt.Errorf("official price for %s tier %d must be a positive finite number", modelName, index+1)
		}
		isLast := index == len(modelPrice.Tiers)-1
		if isLast {
			if tier.UpToInputTokens != nil {
				return fmt.Errorf("official price for %s final tier must not have an input token limit", modelName)
			}
			continue
		}
		if tier.UpToInputTokens == nil || *tier.UpToInputTokens <= previousLimit {
			return fmt.Errorf("official price for %s tier limits must be positive and strictly increasing", modelName)
		}
		previousLimit = *tier.UpToInputTokens
	}
	return nil
}

func GetModelPrice(modelName string) (*ModelPrice, bool) {
	price, ok := officialPriceSetting.ModelPrices[modelName]
	if !ok {
		return nil, false
	}
	price.Tiers = append([]PriceTier(nil), price.Tiers...)
	return &price, true
}

func GetModelPricesCopy() map[string]ModelPrice {
	prices := make(map[string]ModelPrice, len(officialPriceSetting.ModelPrices))
	for modelName, price := range officialPriceSetting.ModelPrices {
		price.Tiers = append([]PriceTier(nil), price.Tiers...)
		prices[modelName] = price
	}
	return prices
}
