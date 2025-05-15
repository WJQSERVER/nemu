package render

import (
	"os"
	"os/exec"
)

func HugoRender(dir string) error {
	// 调用 "hugo"命令生成public
	// 将hugo执行后的输出回传到当前shell输出
	cmd := exec.Command("hugo")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
