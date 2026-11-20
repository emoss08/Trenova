module github.com/emoss08/trenova/edi-partner-sim

go 1.27.4

require (
	github.com/bytedance/sonic v1.15.5
	github.com/cenkalti/backoff/v5 v5.0.3
	github.com/emoss08/trenova v0.0.0
	github.com/emoss08/trenova/shared v0.0.0
	github.com/goccy/go-yaml v1.19.2
	github.com/pkg/sftp v1.13.12
	github.com/shopspring/decimal v1.6.0
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8
	golang.org/x/crypto v0.58.0
	golang.org/x/time v0.16.0
)

require (
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/bytedance/gopkg v0.1.4 // indirect
	github.com/bytedance/sonic/loader v0.5.3 // indirect
	github.com/cloudwego/base64x v0.1.7 // indirect
	github.com/go-ozzo/ozzo-validation/v4 v4.5.0 // indirect
	github.com/klauspost/cpuid/v2 v2.4.2 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/smallstep/pkcs7 v0.2.3 // indirect
	github.com/tetratelabs/wazero v1.12.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/arch v0.32.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)

replace (
	github.com/emoss08/trenova => ../../services/tms
	github.com/emoss08/trenova/shared => ../../shared
)
