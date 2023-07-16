package pacman

import (
	"context"
	"fmt"
	"os/exec"
)

type Pacman []string

func (p *Pacman) Ensure(ctx context.Context) error {
	if p == nil {
		return nil
	}

	cmd := []string{"pacman", "-S", "--noconfirm"}
	cmd = append(cmd, *p...)
	fmt.Println("installing packages with `sudo pacman -S --noconfirm`")
	installCmd := exec.CommandContext(ctx, "sudo", cmd...)
	if output, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error running `pacman`: %w\n%s", err, string(output))
	}
	return nil
}
