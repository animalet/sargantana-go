//go:build tools

package tools

import (
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
	_ "github.com/securego/gosec/v2/cmd/gosec"
	_ "github.com/vladopajic/go-test-coverage/v2"
	_ "golang.org/x/tools/cmd/goimports"
)
