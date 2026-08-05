package setting

import "github.com/QuantumNous/new-api/common"

// KnowledgeBaseServiceConfig holds the third-party knowledge base service settings.
// All values are read from environment variables.
var KnowledgeBaseServiceConfig = struct {
	// BaseUrl is the base URL of the third-party knowledge base service, e.g. http://127.0.0.1:8500
	BaseUrl string
	// ApiKey is the shared secret used for both X-Api-Key (service-to-service) and HMAC signature (user request) authentication.
	ApiKey string
}{
	BaseUrl: common.GetEnvOrDefaultString("KNOWLEDGE_BASE_URL", ""),
	ApiKey:  common.GetEnvOrDefaultString("KNOWLEDGE_BASE_API_KEY", ""),
}

// IsKnowledgeBaseEnabled reports whether the knowledge base integration is configured.
func IsKnowledgeBaseEnabled() bool {
	return KnowledgeBaseServiceConfig.BaseUrl != "" && KnowledgeBaseServiceConfig.ApiKey != ""
}
