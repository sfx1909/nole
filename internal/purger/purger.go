package purger

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sfx1909/nole/internal/flake"
	"github.com/sfx1909/nole/internal/git"
	"github.com/sfx1909/nole/internal/oplog"
	"github.com/sfx1909/nole/internal/output"
	"github.com/sfx1909/nole/internal/style"
)

// Finding is a directory or nix build symlink considered safe to remove.
type Finding struct {
	Path      string
	Size      int64
	IsSymlink bool
}

// Run finds and (optionally) removes dev build artifacts under root.
// If root is empty, the current flake's path is used, falling back to the
// current working directory.
func Run(root string, apply bool) error {
	root, err := resolveRoot(root)
	if err != nil {
		return err
	}

	findings, err := walk(root)
	if err != nil {
		return fmt.Errorf("failed to scan %s: %w", root, err)
	}

	if len(findings) == 0 {
		fmt.Println(style.Green.Render("  󰄬  No build artifacts found"))
		fmt.Println()
		return nil
	}

	sort.Slice(findings, func(i, j int) bool { return findings[i].Size > findings[j].Size })
	printFindings(root, findings)

	if !apply {
		fmt.Printf("  %s Run with %s to remove these\n\n",
			style.Faint.Render("→"),
			style.Cyan.Render("--apply"),
		)
		return nil
	}

	var total int64
	for _, f := range findings {
		total += f.Size
	}

	if !git.Confirm(fmt.Sprintf("  Delete these %d items, freeing ~%s?", len(findings), output.HumanBytes(total))) {
		return nil
	}

	removed, freed := remove(findings)

	fmt.Println()
	fmt.Printf("  %s  Removed %s, freeing %s\n\n", style.Green.Render("󰄬"), plural(removed, "item"), output.HumanBytes(freed))

	return oplog.Append(oplog.Entry{
		Action:  "purge",
		Summary: fmt.Sprintf("removed %s, freed %s", plural(removed, "item"), output.HumanBytes(freed)),
		Details: map[string]string{
			"path":  root,
			"items": fmt.Sprintf("%d", removed),
			"freed": output.HumanBytes(freed),
		},
	})
}

func resolveRoot(root string) (string, error) {
	if root != "" {
		abs, err := filepath.Abs(root)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("path does not exist: %s", abs)
		}
		return abs, nil
	}

	if ctx, err := flake.Detect(); err == nil {
		return ctx.FlakePath, nil
	}

	return os.Getwd()
}

func walk(root string) ([]Finding, error) {
	var findings []Finding

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == root {
			return nil
		}

		name := d.Name()

		if d.IsDir() {
			if name == ".git" {
				return filepath.SkipDir
			}
			if targets[name] {
				size, _ := dirSize(path)
				findings = append(findings, Finding{Path: path, Size: size})
				return filepath.SkipDir
			}
			return nil
		}

		if d.Type()&os.ModeSymlink != 0 && resultLinkRe.MatchString(name) {
			findings = append(findings, Finding{Path: path, IsSymlink: true})
		}

		return nil
	})

	return findings, err
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			if info, err := d.Info(); err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size, err
}

func remove(findings []Finding) (removed int, freed int64) {
	for _, f := range findings {
		var err error
		if f.IsSymlink {
			err = os.Remove(f.Path)
		} else {
			err = os.RemoveAll(f.Path)
		}

		if err != nil {
			fmt.Printf("  %s failed to remove %s: %v\n", style.Yellow.Render(""), f.Path, err)
			continue
		}

		removed++
		freed += f.Size
	}
	return removed, freed
}

func printFindings(root string, findings []Finding) {
	fmt.Printf("  %s\n", style.Bold.Render(fmt.Sprintf("Purge candidates (%s)", root)))

	var total int64
	for _, f := range findings {
		rel, err := filepath.Rel(root, f.Path)
		if err != nil {
			rel = f.Path
		}

		size := output.HumanBytes(f.Size)
		icon := style.Cyan.Render("󰉍")
		if f.IsSymlink {
			size = "—"
			icon = style.Cyan.Render("󰜺")
		}

		fmt.Printf("  %s  %-8s %s\n", icon, size, style.Faint.Render("./"+rel))
		total += f.Size
	}

	fmt.Printf("  %s\n", style.Faint.Render(fmt.Sprintf("Total: %s across %s", output.HumanBytes(total), plural(len(findings), "item"))))
	fmt.Println()
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
