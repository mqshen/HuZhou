package policy

import (
	"github.com/HuZhou/apiserver/pkg/authorization/authorizer"
	"github.com/HuZhou/apiserver/pkg/apis/audit"
)

// Checker exposes methods for checking the policy rules.
type Checker interface {
	// Check the audit level for a request with the given authorizer attributes.
	Level(authorizer.Attributes) audit.Level
}
