module github.com/scality/raidmgmt/core

go 1.24

replace github.com/scality/raidmgmt/domain => ../domain

require (
	github.com/pkg/errors v0.9.1
	github.com/scality/raidmgmt/domain v0.0.0-00010101000000-000000000000
)
