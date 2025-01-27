module github.com/scality/raidmgmt/megaraid

go 1.23

require (
	github.com/stretchr/testify v1.10.0
 	github.com/scality/raidmgmt/utils v0.0.0-00010101000000-000000000000 // indirect
)

replace github.com/scality/raidmgmt/utils => ./../../../utils

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
