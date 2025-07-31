module observer-pattern

go 1.23.0

require (
	github.com/CrisisTextLine/modular v0.0.0-00010101000000-000000000000
	github.com/CrisisTextLine/modular/modules/eventlogger v0.0.0-00010101000000-000000000000
)

require (
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/golobby/cast v1.3.3 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/CrisisTextLine/modular => ../..

replace github.com/CrisisTextLine/modular/modules/eventlogger => ../../modules/eventlogger
