package cleaner

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/sfx1909/nole/internal/builder"
	"github.com/sfx1909/nole/internal/git"
	"github.com/sfx1909/nole/internal/oplog"
)

var freedRe = regexp.MustCompile(`freeing ([\d.]+\s*\w+)`)

// Run garbage-collects old generations and optimises the Nix store.
// Without apply, it only previews how many store paths are garbage.
func Run(apply bool) error {
	dead, err := previewDead()
	if err != nil {
		fmt.Printf("  %s could not check for garbage: %v\n\n", color.New(color.Faint).Sprint(""), err)
	} else if dead == 0 {
		fmt.Println(color.GreenString("  󰄬  Nothing to clean"))
		fmt.Println()
		return nil
	} else {
		fmt.Printf("  %s  %s ready to be collected\n", color.CyanString(""), plural(dead, "store path"))
		fmt.Printf("  %s %s\n", color.New(color.Faint).Sprint("→"), color.New(color.Faint).Sprint("run as root for a complete picture across all profiles"))
	}

	if !apply {
		fmt.Printf("  %s Run with %s to remove old generations and optimise the store\n\n",
			color.New(color.Faint).Sprint("→"),
			color.CyanString("--apply"),
		)
		return nil
	}

	if !git.Confirm("  Run garbage collection and delete old generations? This cannot be undone.") {
		return nil
	}

	if err := builder.EnsureSudo(); err != nil {
		return err
	}
	defer exec.Command("sudo", "-k").Run()

	freed, err := collectGarbage()
	if err != nil {
		return err
	}
	fmt.Printf("  %s  Garbage collected, freeing %s\n", color.GreenString("󰄬"), freed)

	if err := optimiseStore(); err != nil {
		return err
	}
	fmt.Printf("  %s  Store optimised\n\n", color.GreenString("󰄬"))

	return oplog.Append(oplog.Entry{
		Action:  "clean",
		Summary: fmt.Sprintf("garbage collected, freed %s; store optimised", freed),
		Details: map[string]string{"freed": freed},
	})
}

// previewDead returns the number of store paths reachable for garbage
// collection, without deleting anything.
func previewDead() (int, error) {
	out, err := exec.Command("nix-store", "--gc", "--print-dead").Output()
	if err != nil {
		return 0, err
	}

	count := 0
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}

func collectGarbage() (string, error) {
	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond)
	s.Suffix = color.New(color.Faint).Sprint("  Collecting garbage")
	s.Start()

	out, err := exec.Command("sudo", "nix-collect-garbage", "-d").CombinedOutput()
	s.Stop()
	if err != nil {
		return "", fmt.Errorf("nix-collect-garbage failed: %s", strings.TrimSpace(string(out)))
	}

	if m := freedRe.FindStringSubmatch(string(out)); m != nil {
		return m[1], nil
	}
	return "0 bytes", nil
}

func optimiseStore() error {
	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond)
	s.Suffix = color.New(color.Faint).Sprint("  Optimising store")
	s.Start()

	out, err := exec.Command("sudo", "nix", "store", "optimise").CombinedOutput()
	s.Stop()
	if err != nil {
		return fmt.Errorf("nix store optimise failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
