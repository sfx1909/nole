package maintainer

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/sfx1909/nole/internal/builder"
	"github.com/sfx1909/nole/internal/cleaner"
	"github.com/sfx1909/nole/internal/flake"
	"github.com/sfx1909/nole/internal/git"
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
		fmt.Println(color.GreenString("  󰄬  System is up to date"))
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
	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond)
	s.Suffix = color.New(color.Faint).Sprint("  Updating flake inputs")
	s.Start()

	cmd := exec.Command("nix", "flake", "update", "--flake", flakePath)
	out, err := cmd.CombinedOutput()
	s.Stop()

	if err != nil {
		return fmt.Errorf("flake update failed: %s", strings.TrimSpace(string(out)))
	}

	fmt.Println(color.GreenString("  󰚰  Flake inputs updated"))
	return nil
}

func rebuildNeeded(ctx *flake.Context) (bool, string, error) {
	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond)
	s.Suffix = color.New(color.Faint).Sprint("  Checking for changes")
	s.Start()

	// compare derivation paths — evaluation only, no building
	newDrv, err := newSystemDrv(ctx)
	s.Stop()
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

	diffOut, _ := storeDiff(ctx)
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

	fmt.Println(color.New(color.Bold).Sprint("  Changes"))
	for _, line := range strings.Split(strings.TrimSpace(diff), "\n") {
		if strings.Contains(line, "→") {
			fmt.Printf("  %s %s\n", color.CyanString(""), color.New(color.Faint).Sprint(line))
		} else {
			fmt.Printf("    %s\n", color.New(color.Faint).Sprint(line))
		}
	}
	fmt.Println()
}

func printTips(ctx *flake.Context, clean bool) {
	var tips []string

	if git.IsDirty(ctx.FlakePath) {
		tips = append(tips, "Your config has uncommitted changes — consider running "+color.CyanString("git commit")+" to keep your history clean")
	}

	if !clean {
		if dead, err := cleaner.PreviewDead(); err == nil && dead > 0 {
			tips = append(tips, fmt.Sprintf("%d store paths are garbage — run %s to reclaim space", dead, color.CyanString("nole clean --apply")))
		}
	}

	if len(tips) == 0 {
		return
	}

	fmt.Println(color.New(color.Bold).Sprint("  Tips"))
	for _, t := range tips {
		fmt.Printf("  %s %s\n", color.YellowString(""), t)
	}
	fmt.Println()
}

// notify sends a desktop notification if notify-send is available.
func notify(msg string) {
	if path, err := exec.LookPath("notify-send"); err == nil {
		exec.Command(path, "-a", "Nole", "-i", "system-software-update", msg).Run()
	}
}
