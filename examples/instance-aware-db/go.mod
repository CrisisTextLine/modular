module instance-aware-db

go 1.24.2

replace github.com/CrisisTextLine/modular => ../..

replace github.com/CrisisTextLine/modular/modules/database => ../../modules/database

require (
	github.com/CrisisTextLine/modular v1.4.0
	github.com/CrisisTextLine/modular/modules/database v1.1.0
	github.com/mattn/go-sqlite3 v1.14.28
)

require (
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.36.3 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.29.14 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.67 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30 // indirect
	github.com/aws/aws-sdk-go-v2/feature/rds/auth v1.5.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.19 // indirect
	github.com/aws/smithy-go v1.22.2 // indirect
	github.com/golobby/cast v1.3.3 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
