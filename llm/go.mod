module github.com/SmilingXinyi/gb/llm

go 1.24.0

require (
	github.com/SmilingXinyi/gb/jv v0.0.0-00010101000000-000000000000
	github.com/SmilingXinyi/gb/trace_id v0.0.0-00010101000000-000000000000
	github.com/SmilingXinyi/gb/utils v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/invopop/jsonschema v0.14.0 // indirect
	github.com/pb33f/ordered-map/v2 v2.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/SmilingXinyi/gb/jv => ../jv
	github.com/SmilingXinyi/gb/trace_id => ../trace_id
	github.com/SmilingXinyi/gb/utils => ../utils
)
