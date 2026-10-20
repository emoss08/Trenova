package codemode

import _ "embed"

//go:embed sdk/trenova.py
var ShimSource []byte

//go:embed sdk/main.py
var MainSource []byte

//go:embed sdk/runner.py
var RunnerSource []byte

const (
	SandboxConfigPath = SandboxRoot + "/config.json"
	SandboxIndexPath  = SandboxRoot + "/index.json"
	SandboxInputsPath = SandboxRoot + "/inputs.json"
	SandboxScriptPath = SandboxRoot + "/script.py"
	SandboxShimPath   = SandboxSDKDir + "/trenova.py"
	SandboxMainPath   = SandboxSDKDir + "/main.py"
)

type SandboxConfig struct {
	Nonce       string `json:"nonce"`
	PollMs      int64  `json:"pollMs"`
	WallSeconds int    `json:"wallSeconds"`
	FrameBytes  int    `json:"frameBytes"`
	ResultBytes int    `json:"resultBytes"`
	MemoryBytes int64  `json:"memoryBytes"`
	RPCDir      string `json:"rpcDir"`
}

func ConfigFor(nonce string, limits Limits) SandboxConfig {
	return SandboxConfig{
		Nonce:       nonce,
		PollMs:      PollInterval.Milliseconds(),
		WallSeconds: int(limits.Wall.Seconds()),
		FrameBytes:  limits.FrameBytes,
		ResultBytes: limits.ResultBytes,
		MemoryBytes: limits.MemoryBytes,
		RPCDir:      SandboxRPCDir,
	}
}
