package util

import (
	"fmt"
	"os/exec"
	"runtime"

	"backup/consts"
	"backup/internal/config"
)

var commands = map[string]string{
	"windows": "cmd",
	"darwin":  "open",
	"linux":   "xdg-open",
}

func OpenBrowser() error {
	url := fmt.Sprintf(consts.AuthorizationCodeUrl, config.Config.PcsConfig.AppKey)
	run, ok := commands[runtime.GOOS]
	if !ok {
		return fmt.Errorf("don't know how to open things on %s platform", runtime.GOOS)
	}

	var cmd *exec.Cmd
	if run == "cmd" {
		url := fmt.Sprintf(consts.AuthorizationCodeUrlWindows, config.Config.PcsConfig.AppKey)
		cmd = exec.Command(run, "/c", "start", url)
	} else {
		cmd = exec.Command(run, url)
	}
	return cmd.Run()
}
