package billing_setting

const (
	RequestPriceDimensionImageResolution = "image_resolution"
	ImageResolutionBillingRatioPrefix    = RequestPriceDimensionImageResolution + ":"
)

type RequestPriceTier struct {
	Value      string  `json:"value"`
	Multiplier float64 `json:"multiplier"`
}

type RequestPricePolicy struct {
	Dimension    string             `json:"dimension"`
	DefaultValue string             `json:"default_value"`
	Tiers        []RequestPriceTier `json:"tiers"`
}

var requestPricePolicies = map[string]RequestPricePolicy{
	"gemini-3-pro-image":         imageResolutionPolicy(proImageResolutionTiers()),
	"gemini-3-pro-image-preview": imageResolutionPolicy(proImageResolutionTiers()),
	"gemini-3.1-flash-image":     imageResolutionPolicy(flashImageResolutionTiers()),
	"gemini-3.1-flash-image-preview": imageResolutionPolicy(
		flashImageResolutionTiers(),
	),
}

func imageResolutionPolicy(tiers []RequestPriceTier) RequestPricePolicy {
	return RequestPricePolicy{
		Dimension:    RequestPriceDimensionImageResolution,
		DefaultValue: "1K",
		Tiers:        tiers,
	}
}

func proImageResolutionTiers() []RequestPriceTier {
	return []RequestPriceTier{
		{Value: "1K", Multiplier: 1},
		{Value: "2K", Multiplier: 1},
		{Value: "4K", Multiplier: 1.79},
	}
}

func flashImageResolutionTiers() []RequestPriceTier {
	return []RequestPriceTier{
		{Value: "512", Multiplier: 0.66696},
		{Value: "1K", Multiplier: 1},
		{Value: "2K", Multiplier: 1},
		{Value: "4K", Multiplier: 1.78571},
	}
}

func GetRequestPricePolicy(model string) (RequestPricePolicy, bool) {
	policy, ok := requestPricePolicies[model]
	if !ok {
		return RequestPricePolicy{}, false
	}
	policy.Tiers = append([]RequestPriceTier(nil), policy.Tiers...)
	return policy, true
}

func (p RequestPricePolicy) FindTier(value string) (RequestPriceTier, bool) {
	for _, tier := range p.Tiers {
		if tier.Value == value {
			return tier, true
		}
	}
	return RequestPriceTier{}, false
}
