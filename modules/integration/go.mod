module integration

go 1.24.2

toolchain go1.24.4

// Use the local modular framework
replace github.com/CrisisTextLine/modular => ../..

require (
	github.com/CrisisTextLine/modular v1.4.0
	github.com/CrisisTextLine/modular/modules/httpclient v0.1.1
	github.com/CrisisTextLine/modular/modules/reverseproxy v1.1.2
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golobby/cast v1.3.3 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
