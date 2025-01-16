module github.com/scality/raidmgmt/core

go 1.23

replace github.com/scality/raidmgmt/domain => ../domain

require (
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.33.0
	github.com/scality/raidmgmt/domain v0.0.0-00010101000000-000000000000
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/sys v0.12.0 // indirect
)
