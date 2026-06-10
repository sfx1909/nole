package builder

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/sfx1909/nole/internal/flake"
	"github.com/sfx1909/nole/internal/git"
	"github.com/sfx1909/nole/internal/output"
	"github.com/sfx1909/nole/internal/style"
	"golang.org/x/term"
)

func Run() error {
	ctx, err := flake.Detect()
	if err != nil {
		return err
	}
	return RunWithContext(ctx)
}

func RunWithContext(ctx *flake.Context) error {
	if files, err := git.UntrackedNixFiles(ctx.FlakePath); err == nil && len(files) > 0 {
		if err := git.PromptStage(ctx.FlakePath, files); err != nil {
			return err
		}
	}

	// only acquire+revoke sudo if not already held by the caller
	if exec.Command("sudo", "-n", "true").Run() != nil {
		if err := EnsureSudo(); err != nil {
			return err
		}
		defer exec.Command("sudo", "-k").Run()
	}

	summary := output.NewSummary()
	title := fmt.Sprintf("  Building NixOS (%s#%s)", ctx.FlakePath, ctx.ConfigName)
	buildErr := style.Spin(title, func() error {
		cmd := exec.Command("sudo", "nixos-rebuild", "switch", "--flake", fmt.Sprintf("%s#%s", ctx.FlakePath, ctx.ConfigName))

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start nixos-rebuild: %w", err)
		}

		var wg sync.WaitGroup
		scanPipe := func(r io.Reader) {
			defer wg.Done()
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				summary.Parse(scanner.Text())
			}
		}

		wg.Add(2)
		go scanPipe(stdout)
		go scanPipe(stderr)
		wg.Wait()

		return cmd.Wait()
	})

	if buildErr != nil {
		fmt.Println(style.Red.Render("  Build failed"))
		summary.Print()
		summary.PrintLog()
		return buildErr
	}

	fmt.Println(style.Green.Render("  󰄬  Build successful"))
	summary.Print()

	if err := git.PromptStageAndCommit(ctx.FlakePath); err != nil {
		return err
	}

	return nil
}

func EnsureSudo() error {
	if exec.Command("sudo", "-n", "true").Run() == nil {
		return nil
	}

	fmt.Println(style.Faint.Render("  Sudo is required to rebuild NixOS"))
	fmt.Print(style.Cyan.Render("  Password: "))
	pwd, err := term.ReadPassword(int(os.Stdin.Fd()))
	// clear the 2 prompt lines
	fmt.Print("\033[2K\r\033[1A\033[2K\r")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	cmd := exec.Command("sudo", "-S", "-v")
	cmd.Stdin = bytes.NewReader(append(pwd, '\n'))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("incorrect password")
	}

	return nil
}
