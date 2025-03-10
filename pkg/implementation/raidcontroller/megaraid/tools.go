//go:build tools
// +build tools

package megaraid

import (
	_ "github.com/vektra/mockery/v2"
)

//go:generate mockery --name=PathResolver --output=mocks --outpkg=mocks
//go:generate mockery --name=Runner --output=mocks --outpkg=mocks
