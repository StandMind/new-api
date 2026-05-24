package setting

var CreemApiKey = ""
var CreemProducts = "[]"
var CreemTestMode = false
var CreemTestApiKey = ""
var CreemTestProducts = "[]"
var CreemTestWebhookSecret = ""
var CreemWebhookSecret = ""

func GetActiveCreemApiKey() string {
	if CreemTestMode {
		return CreemTestApiKey
	}
	return CreemApiKey
}

func GetActiveCreemProducts() string {
	if CreemTestMode {
		return CreemTestProducts
	}
	return CreemProducts
}

func GetActiveCreemWebhookSecret() string {
	if CreemTestMode {
		return CreemTestWebhookSecret
	}
	return CreemWebhookSecret
}
