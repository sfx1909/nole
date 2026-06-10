package status

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/sfx1909/nole/internal/flake"
	"github.com/sfx1909/nole/internal/git"
)

// staleLockAge is the threshold beyond which flake.lock is flagged as stale.
const staleLockAge = 30 * 24 * time.Hour

type generation struct {
	Generation int    `json:"generation"`
	Date       string `json:"date"`
	Current    bool   `json:"current"`
}

type diskUsage struct {
	Size       string
	Used       string
	Avail      string
	UsePercent string
}

// Run prints a quick, read-only dashboard of system and flake state.
func Run() error {
	fmt.Println(color.New(color.Bold).Sprint("  Status"))

	printGenerations()
	printDiskUsage()
	printFlakeStatus()

	fmt.Println()
	return nil
}

func printGenerations() {
	gens, err := collectGenerations()
	if err != nil || len(gens) == 0 {
		fmt.Printf("  %s %s\n", color.New(color.Faint).Sprint(""), color.New(color.Faint).Sprint("generations: unavailable"))
		return
	}

	var current *generation
	oldest := gens[0]
	for i := range gens {
		if gens[i].Current {
			current = &gens[i]
		}
		if gens[i].Generation < oldest.Generation {
			oldest = gens[i]
		}
	}

	if current != nil {
		fmt.Printf("  %s  Generation %d (current, built %s)\n", color.CyanString(""), current.Generation, formatGenDate(current.Date))
	}
	fmt.Printf("  %s  %s (oldest: %s)\n", color.CyanString("󰓦"), plural(len(gens), "generation"), formatGenDate(oldest.Date))
}

func collectGenerations() ([]generation, error) {
	out, err := exec.Command("nixos-rebuild", "list-generations", "--json").Output()
	if err != nil {
		return nil, err
	}

	var gens []generation
	if err := json.Unmarshal(out, &gens); err != nil {
		return nil, err
	}
	return gens, nil
}

func formatGenDate(s string) string {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
	if err != nil {
		return s
	}
	return t.Format("2006-01-02 15:04")
}

func printDiskUsage() {
	du, err := collectDiskUsage()
	if err != nil {
		fmt.Printf("  %s %s\n", color.New(color.Faint).Sprint(""), color.New(color.Faint).Sprint("/nix/store: unavailable"))
		return
	}

	fmt.Printf("  %s  /nix/store: %s used / %s (%s, %s free)\n",
		color.CyanString("󰋊"), du.Used, du.Size, du.UsePercent, du.Avail)
}

func collectDiskUsage() (diskUsage, error) {
	out, err := exec.Command("df", "-h", "/nix/store").Output()
	if err != nil {
		return diskUsage{}, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return diskUsage{}, fmt.Errorf("unexpected df output")
	}

	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 5 {
		return diskUsage{}, fmt.Errorf("unexpected df output")
	}

	return diskUsage{
		Size:       fields[1],
		Used:       fields[2],
		Avail:      fields[3],
		UsePercent: fields[4],
	}, nil
}

func printFlakeStatus() {
	ctx, err := flake.Detect()
	if err != nil {
		fmt.Printf("  %s %s\n", color.New(color.Faint).Sprint(""), color.New(color.Faint).Sprint("flake: not found"))
		return
	}

	fmt.Printf("  %s  flake: %s#%s\n", color.CyanString("󱄅"), ctx.FlakePath, ctx.ConfigName)

	if git.IsDirty(ctx.FlakePath) {
		fmt.Printf("  %s  flake repo: uncommitted changes\n", color.YellowString(""))
	} else {
		fmt.Printf("  %s  flake repo: clean\n", color.CyanString("󰊢"))
	}

	lockPath := filepath.Join(ctx.FlakePath, "flake.lock")
	info, err := os.Stat(lockPath)
	if err != nil {
		return
	}

	age := time.Since(info.ModTime())
	if age > staleLockAge {
		fmt.Printf("  %s  flake.lock: %s old (consider %s)\n", color.YellowString(""), formatAge(age), color.CyanString("nole maintain"))
	} else {
		fmt.Printf("  %s  flake.lock: %s old\n", color.CyanString("󰚰"), formatAge(age))
	}
}

func formatAge(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days <= 0 {
		return "less than a day"
	}
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
