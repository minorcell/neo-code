package main

import (
	"fmt"
	"os"
	"os/exec"
)

// main 作为兼容入口转调带 gatewaydocgen 标签的生成器，保持旧命令路径可用。
func main() {
	cmd := exec.Command("go", "run", "-tags", "gatewaydocgen", "./scripts/generate_gateway_rpc_examples.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "run gateway rpc example generator: %v\n", err)
		os.Exit(1)
	}
}
