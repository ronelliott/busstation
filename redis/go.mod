module github.com/ronelliott/busstation/redis

go 1.24

require (
	github.com/alicebob/miniredis/v2 v2.37.0
	github.com/redis/go-redis/v9 v9.7.3
	github.com/ronelliott/busstation v0.0.0-dev
	github.com/stretchr/testify v1.8.4
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// replace is for local development only. Remove this directive and update
// github.com/ronelliott/busstation to a real tagged version before publishing.
replace github.com/ronelliott/busstation => ../
