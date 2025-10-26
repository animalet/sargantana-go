package config

import (
	"github.com/hashicorp/vault/api"
)

// Legacy vault manager instance - kept for backward compatibility during transition
// New code should use the PropertyResolver system via RegisterPropertyResolver
var vaultManagerInstance *api.Client
var vaultPath string
