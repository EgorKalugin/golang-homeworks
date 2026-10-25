module gateway

go 1.27.1

require (
	github.com/stretchr/testify v1.12.1
	ledger v0.0.0
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

replace ledger => ../ledger
