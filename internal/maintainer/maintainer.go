package maintainer

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/sfx1909/nole/internal/builder"
	"github.com/sfx1909/nole/internal/cleaner"
	"github.com/sfx1909/nole/internal/flake"
	"github.com/sfx1909/nole/internal/git"
	"github.com/sfx1909/nole/internal/style"
)

// Run updates flake inputs and rebuilds if needed. If clean is true, it also
// runs garbage collection and store optimisation afterwards (nole clean --apply).
func Run(clean bool) error {
	ctx, err := flake.Detect()
	if err != nil {
		return err
	}

	if err := updateFlake(ctx.FlakePath); err != nil {
		return err
	}

	if err := git.CommitLockIfOnly(ctx.FlakePath); err != nil {
		return err
	}

	needed, diff, err := rebuildNeeded(ctx)
	if err != nil {
		return err
	}

	var buildErr error
	if !needed {
		fmt.Println(style.Green.Render("  󰄬  System is up to date"))
		fmt.Println()
		if err := git.PromptStageAndCommit(ctx.FlakePath); err != nil {
			return err
		}
	} else {
		// sudo only needed for the actual rebuild
		notify("Nole requires sudo to apply NixOS changes")
		if err := builder.EnsureSudo(); err != nil {
			return err
		}
		defer exec.Command("sudo", "-k").Run()

		printDiff(diff)
		buildErr = builder.RunWithContext(ctx)
	}

	printTips(ctx, clean)

	if buildErr != nil {
		return buildErr
	}

	if clean {
		fmt.Println()
		return cleaner.Run(true)
	}

	return nil
}

func updateFlake(flakePath string) error {
	var out []byte
	err := style.Spin("  Updating flake inputs", func() error {
		var cmdErr error
		out, cmdErr = exec.Command("nix", "flake", "update", "--flake", flakePath).CombinedOutput()
		return cmdErr
	})
	if err != nil {
		return fmt.Errorf("flake update failed: %s", strings.TrimSpace(string(out)))
	}

	fmt.Println(style.Green.Render("  󰚰  Flake inputs updated"))
	return nil
}

func rebuildNeeded(ctx *flake.Context) (bool, string, error) {
	var newDrv string
	err := style.Spin("  Checking for changes", func() error {
		var evalErr error
		newDrv, evalErr = newSystemDrv(ctx)
		return evalErr
	})
	if err != nil {
		return false, "", err
	}

	currentDrv, err := currentSystemDrv()
	if err != nil {
		return false, "", err
	}

	if newDrv == currentDrv {
		return false, "", nil
	}

	var diffOut string
	_ = style.Spin("  Building new system", func() error {
		diffOut, _ = storeDiff(ctx)
		return nil
	})
	return true, diffOut, nil
}

func newSystemDrv(ctx *flake.Context) (string, error) {
	out, err := exec.Command("nix", "path-info", "--derivation",
		fmt.Sprintf("%s#nixosConfigurations.%s.config.system.build.toplevel", ctx.FlakePath, ctx.ConfigName),
	).Output()
	if err != nil {
		return "", fmt.Errorf("could not evaluate new system derivation: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func currentSystemDrv() (string, error) {
	out, err := exec.Command("nix-store", "--query", "--deriver", "/run/current-system").Output()
	if err != nil {
		return "", fmt.Errorf("could not query current system derivation: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func storeDiff(ctx *flake.Context) (string, error) {
	buildCmd := exec.Command("nix", "build", "--no-link", "--print-out-paths",
		fmt.Sprintf("%s#nixosConfigurations.%s.config.system.build.toplevel", ctx.FlakePath, ctx.ConfigName),
	)
	out, err := buildCmd.Output()
	if err != nil {
		return "", err
	}

	newSystem := strings.TrimSpace(string(out))
	diffCmd := exec.Command("nix", "store", "diff-closures", "/run/current-system", newSystem)
	diffOut, _ := diffCmd.Output()
	return string(diffOut), nil
}

func printDiff(diff string) {
	if diff == "" {
		return
	}

	fmt.Println(style.Bold.Render("  Changes"))
	for _, line := range strings.Split(strings.TrimSpace(diff), "\n") {
		if strings.Contains(line, "→") {
			fmt.Printf("  %s %s\n", style.Cyan.Render(""), style.Faint.Render(line))
		} else {
			fmt.Printf("    %s\n", style.Faint.Render(line))
		}
	}
	fmt.Println()
}

func printTips(ctx *flake.Context, clean bool) {
	var tips []string

	if git.IsDirty(ctx.FlakePath) {
		tips = append(tips, "Your config has uncommitted changes — consider running "+style.Cyan.Render("git commit")+" to keep your history clean")
	}

	if !clean {
		if dead, err := cleaner.PreviewDead(); err == nil && dead > 0 {
			tips = append(tips, fmt.Sprintf("%d store paths are garbage — run %s to reclaim space", dead, style.Cyan.Render("nole clean --apply")))
		}
	}

	if len(tips) == 0 {
		return
	}

	fmt.Println(style.Bold.Render("  Tips"))
	for _, t := range tips {
		fmt.Printf("  %s %s\n", style.Yellow.Render(""), t)
	}
	fmt.Println()
}

// notify sends a desktop notification if notify-send is available.
func notify(msg string) {
	if path, err := exec.LookPath("notify-send"); err == nil {
		exec.Command(path, "-a", "Nole", "-i", "system-software-update", msg).Run()
	}
}
