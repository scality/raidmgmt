module github.com/scality/raidmgmt/scenario

go 1.25.6

require (
	github.com/cucumber/godog v0.16.0
	github.com/pkg/errors v0.9.1
	github.com/scality/raidmgmt v0.14.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/cucumber/gherkin/go/v42 v42.0.0 // indirect
	github.com/cucumber/messages/go/v34 v34.2.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-memdb v1.3.5 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Play the scenarios on the adapters of this repository.
replace github.com/scality/raidmgmt => ../
