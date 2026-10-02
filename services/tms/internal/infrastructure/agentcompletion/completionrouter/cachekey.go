package completionrouter

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// promptCacheKey names the requests that share a prompt prefix: one agent, at
// one version, in one organization. It is hashed so a provider sees no
// identifier of ours, and names no person, so two people asking the same
// agent share the agent's cached rules and tools.
func promptCacheKey(req *serviceports.ChatCompletionRequest) string {
	agent := "chat"
	if id := req.Attribution.AgentDefinitionID; !id.IsNil() {
		agent = id.String()
		if version := req.Attribution.DefinitionVersion; version != nil {
			agent += "@" + strconv.FormatInt(*version, 10)
		}
	}
	sum := sha256.Sum256(
		[]byte("trenova/prompt-cache/v1|" + req.TenantInfo.OrgID.String() + "|" + agent),
	)

	return hex.EncodeToString(sum[:16])
}
