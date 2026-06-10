package output

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sfx1909/nole/internal/style"
)

type Summary struct {
	Built        int
	warnings     []string
	deprecations []string
	lines        []string
	warnCounts   map[string]int
	deprCounts   map[string]int
}

func NewSummary() *Summary {
	return &Summary{
		warnCounts: make(map[string]int),
		deprCounts: make(map[string]int),
	}
}

func (s *Summary) Parse(line string) {
	s.lines = append(s.lines, line)

	lower := strings.ToLower(line)
	if strings.Contains(lower, "warning:") {
		msg := trimPrefix(strings.TrimSpace(line), "warning:")
		if s.warnCounts[msg] == 0 {
			s.warnings = append(s.warnings, msg)
		}
		s.warnCounts[msg]++
	}
	if strings.Contains(lower, "deprecated") {
		msg := strings.TrimSpace(line)
		if s.deprCounts[msg] == 0 {
			s.deprecations = append(s.deprecations, msg)
		}
		s.deprCounts[msg]++
	}
	// "these 23 derivations will be built:"
	if strings.Contains(line, "derivations will be built") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "derivations" && i > 0 {
				if n, err := strconv.Atoi(fields[i-1]); err == nil {
					s.Built += n
				}
			}
		}
	}
}

func (s *Summary) Print() {
	fmt.Println()
	fmt.Println(style.Bold.Render("  Summary"))
	if s.Built > 0 {
		fmt.Printf("  %s  %s\n", style.Cyan.Render("󰏗"), plural(s.Built, "built"))
	} else {
		fmt.Printf("  %s  all cached\n", style.Faint.Render(""))
	}

	if len(s.warnings) > 0 {
		fmt.Printf("  %s  %s\n", style.Yellow.Render(""), plural(len(s.warnCounts), "warning"))
		for _, w := range s.warnings {
			suffix := ""
			if s.warnCounts[w] > 1 {
				suffix = style.Yellow.Render(fmt.Sprintf(" (x%d)", s.warnCounts[w]))
			}
			fmt.Printf("      %s %s%s\n", style.Yellow.Render("·"), style.Faint.Render(w), suffix)
		}
	}

	if len(s.deprecations) > 0 {
		fmt.Printf("  %s  %s\n", style.Magenta.Render(""), plural(len(s.deprCounts), "deprecation"))
		for _, d := range s.deprecations {
			suffix := ""
			if s.deprCounts[d] > 1 {
				suffix = style.Magenta.Render(fmt.Sprintf(" (x%d)", s.deprCounts[d]))
			}
			fmt.Printf("      %s %s%s\n", style.Magenta.Render("·"), style.Faint.Render(d), suffix)
		}
	}

	fmt.Println()
}

func (s *Summary) PrintLog() {
	if len(s.lines) == 0 {
		return
	}
	fmt.Println(style.Bold.Render("  Output"))
	for _, line := range s.lines {
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "error:"):
			fmt.Printf("  %s\n", style.Red.Render(line))
		case strings.Contains(lower, "warning:"):
			fmt.Printf("  %s\n", style.Yellow.Render(line))
		default:
			fmt.Printf("  %s\n", style.Faint.Render(line))
		}
	}
	fmt.Println()
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func trimPrefix(s, prefix string) string {
	lower := strings.ToLower(s)
	if idx := strings.Index(lower, prefix); idx != -1 {
		return strings.TrimSpace(s[idx+len(prefix):])
	}
	return s
}
