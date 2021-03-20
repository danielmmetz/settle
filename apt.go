package main

import (
	"context"
	"fmt"
	"os/exec"
)

type Apt []string

func (a *Apt) Ensure(ctx context.Context) error {
	if a == nil {
		return nil
	}

	cmd := []string{"apt", "install", "-y"}
	cmd = append(cmd, *a...)
	fmt.Println("installing packages with `sudo apt install`")
	installCmd := exec.CommandContext(ctx, "sudo", cmd...)
	if output, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error running `sudo apt install`: %w\n%s", err, string(output))
	}

	fmt.Println("cleaning up orphan packages with `sudo apt autoremove`")
	cleanupCmd := exec.CommandContext(ctx, "sudo", "apt", "autoremove")
	if output, err := cleanupCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error running `sudo apt autoremove`: %w\n%s", err, string(output))
	}
	return nil
}
