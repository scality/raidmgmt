//go:build tools
// +build tools

package main

import (
	_ "github.com/vektra/mockery/v2@v2.46.3"
)

//go:generate mockery --name=PathResolver --output=mocks --outpkg=mocks
//go:generate mockery --name=Runner --output=mocks --outpkg=mocks
