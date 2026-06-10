package oplog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sfx1909/nole/internal/style"
)

// Entry is a single recorded operation.
type Entry struct {
	Time    time.Time         `json:"time"`
	Action  string            `json:"action"`
	Summary string            `json:"summary"`
	Details map[string]string `json:"details,omitempty"`
}

// Path returns the location of the operations log, creating its parent
// directory if needed.
func Path() (string, error) {
	dir, err := dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "operations.log"), nil
}

func dir() (string, error) {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "nole"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "nole"), nil
}

// Append records a new entry, setting Time to now if it is zero.
func Append(e Entry) error {
	if e.Time.IsZero() {
		e.Time = time.Now()
	}

	path, err := Path()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	line, err := json.Marshal(e)
	if err != nil {
		return err
	}

	_, err = f.Write(append(line, '\n'))
	return err
}

// Recent returns up to the last n entries, most recent last.
func Recent(n int) ([]Entry, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(entries) > n {
		entries = entries[len(entries)-n:]
	}
	return entries, nil
}

// PrintRecent prints the last n entries in the project's standard
// "Bold header / Faint rows" style.
func PrintRecent(n int) error {
	entries, err := Recent(n)
	if err != nil {
		return fmt.Errorf("failed to read history: %w", err)
	}

	fmt.Println(style.Bold.Render("  History"))
	if len(entries) == 0 {
		fmt.Println(style.Faint.Render("  No operations recorded yet"))
		fmt.Println()
		return nil
	}

	for _, e := range entries {
		ts := e.Time.Local().Format("2006-01-02 15:04")
		fmt.Printf("  %s  %s  %-7s %s\n",
			style.Cyan.Render(""),
			style.Faint.Render(ts),
			e.Action,
			e.Summary,
		)
	}
	fmt.Println()
	return nil
}
