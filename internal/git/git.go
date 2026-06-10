package git

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

func porcelain(repoPath string) ([]string, error) {
	out, err := exec.Command("git", "-C", repoPath, "status", "--porcelain").Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(string(out), "\n"), nil
}

// UntrackedNixFiles returns untracked .nix files or directories containing them.
// Used pre-build — untracked files are invisible to the nix flake evaluator.
func UntrackedNixFiles(repoPath string) ([]string, error) {
	lines, err := porcelain(repoPath)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range lines {
		if len(line) < 4 || line[:2] != "??" {
			continue
		}
		file := strings.TrimSpace(line[3:])
		if strings.HasSuffix(file, ".nix") {
			files = append(files, file)
		} else if strings.HasSuffix(file, "/") && dirHasNixFiles(filepath.Join(repoPath, file)) {
			files = append(files, file)
		}
	}
	return files, nil
}

// PromptStageAndCommit finds all changed .nix files post-build, offers to stage
// any unstaged ones, then prompts to commit if there is anything staged.
func PromptStageAndCommit(repoPath string) error {
	lines, err := porcelain(repoPath)
	if err != nil {
		return nil
	}

	type entry struct {
		file   string
		staged bool
	}
	var changed []entry
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		status := line[:2]
		file := strings.TrimSpace(line[3:])
		if status == "??" || !strings.HasSuffix(file, ".nix") {
			continue
		}
		if status[0] != ' ' || status[1] != ' ' {
			changed = append(changed, entry{file, status[0] != ' '})
		}
	}

	if len(changed) == 0 {
		return nil
	}

	fmt.Println(color.YellowString("\n  Changed .nix files:"))
	var unstaged []string
	for _, e := range changed {
		marker := color.New(color.Faint).Sprint("·")
		if !e.staged {
			marker = color.YellowString("·")
			unstaged = append(unstaged, e.file)
		}
		fmt.Printf("    %s %s\n", marker, e.file)
	}
	fmt.Println()

	if !confirm("  Stage and commit?") {
		return nil
	}

	if len(unstaged) > 0 {
		args := append([]string{"-C", repoPath, "add", "--"}, unstaged...)
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("git add failed: %s", strings.TrimSpace(string(out)))
		}
	}

	fmt.Print(color.CyanString("  Commit message: "))
	msg, err := readLine()
	if err != nil || strings.TrimSpace(msg) == "" {
		fmt.Println(color.New(color.Faint).Sprint("  Skipping commit — empty message"))
		return nil
	}

	cmd := exec.Command("git", "-C", repoPath, "commit", "-m", strings.TrimSpace(msg))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %s", strings.TrimSpace(string(out)))
	}

	fmt.Println(color.GreenString("  󰄬  Committed"))
	return nil
}

// CommitLockIfOnly commits flake.lock with a standard message if it's the only changed file.
func CommitLockIfOnly(repoPath string) error {
	lines, err := porcelain(repoPath)
	if err != nil {
		return nil
	}

	var changed []string
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		changed = append(changed, strings.TrimSpace(line[3:]))
	}

	if len(changed) != 1 || changed[0] != "flake.lock" {
		return nil
	}

	if err := exec.Command("git", "-C", repoPath, "add", "flake.lock").Run(); err != nil {
		return fmt.Errorf("git add flake.lock: %w", err)
	}
	if out, err := exec.Command("git", "-C", repoPath, "commit", "-m", "chore: updated lock file").CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %s", strings.TrimSpace(string(out)))
	}

	fmt.Println(color.GreenString("  󰄬  Committed lock file"))
	return nil
}

// PromptStage lists unstaged .nix files and asks the user whether to stage them.
func PromptStage(repoPath string, files []string) error {
	fmt.Println(color.YellowString("\n  Uncommitted .nix files detected:"))
	for _, f := range files {
		fmt.Printf("    %s %s\n", color.New(color.Faint).Sprint("·"), f)
	}
	fmt.Println()

	if !confirm("  Stage these files?") {
		return nil
	}

	args := append([]string{"-C", repoPath, "add", "--"}, files...)
	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s", strings.TrimSpace(string(out)))
	}

	fmt.Println(color.GreenString("  󰄬  Files staged"))
	return nil
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N] ", color.CyanString(prompt))
	input, _ := readLine()
	return strings.ToLower(strings.TrimSpace(input)) == "y"
}

// Confirm prompts the user with a y/N question and returns true for "y"/"Y".
func Confirm(prompt string) bool {
	return confirm(prompt)
}

// IsDirty reports whether the git repo at path has any uncommitted changes.
func IsDirty(path string) bool {
	lines, err := porcelain(path)
	if err != nil {
		return false
	}
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			return true
		}
	}
	return false
}

var stdinReader = bufio.NewReader(os.Stdin)

func readLine() (string, error) {
	line, err := stdinReader.ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}

var errNixFound = errors.New("found")

func dirHasNixFiles(dir string) bool {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".nix") {
			return errNixFound
		}
		return nil
	})
	return errors.Is(err, errNixFound)
}
