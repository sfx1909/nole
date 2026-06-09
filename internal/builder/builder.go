package builder

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/sfx1909/nole/internal/output"
)

func Run() error {
	fmt.Println(color.CyanString("  Building NixOS..."))

	cmd := exec.Command("nixos-rebuild", "switch", "--use-remote-sudo")
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start nixos-rebuild: %w", err)
	}

	summary := output.NewSummary()
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		summary.Parse(line)
	}

	if err := cmd.Wait(); err != nil {
		color.Red("  Build failed")
		summary.Print()
		return err
	}

	color.Green("  Build successful")
	summary.Print()
	return nil
}

func hasChanged(lines []string) bool {
	for _, l := range lines {
		if strings.Contains(l, "these") && strings.Contains(l, "derivations will be built") {
			return true
		}
	}
	return false
}
