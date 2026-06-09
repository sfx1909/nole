package output

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
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
	fmt.Println(color.New(color.Bold).Sprint("  Summary"))
	if s.Built > 0 {
		fmt.Printf("  %s  %s\n", color.CyanString("󰏗"), plural(s.Built, "built"))
	} else {
		fmt.Printf("  %s  all cached\n", color.New(color.Faint).Sprint(""))
	}

	if len(s.warnings) > 0 {
		fmt.Printf("  %s  %s\n", color.YellowString(""), plural(len(s.warnCounts), "warning"))
		for _, w := range s.warnings {
			suffix := ""
			if s.warnCounts[w] > 1 {
				suffix = color.YellowString(fmt.Sprintf(" (x%d)", s.warnCounts[w]))
			}
			fmt.Printf("      %s %s%s\n", color.YellowString("·"), color.New(color.Faint).Sprint(w), suffix)
		}
	}

	if len(s.deprecations) > 0 {
		fmt.Printf("  %s  %s\n", color.MagentaString(""), plural(len(s.deprCounts), "deprecation"))
		for _, d := range s.deprecations {
			suffix := ""
			if s.deprCounts[d] > 1 {
				suffix = color.MagentaString(fmt.Sprintf(" (x%d)", s.deprCounts[d]))
			}
			fmt.Printf("      %s %s%s\n", color.MagentaString("·"), color.New(color.Faint).Sprint(d), suffix)
		}
	}

	fmt.Println()
}

func (s *Summary) PrintLog() {
	if len(s.lines) == 0 {
		return
	}
	fmt.Println(color.New(color.Bold).Sprint("  Output"))
	for _, line := range s.lines {
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "error:"):
			fmt.Printf("  %s\n", color.RedString(line))
		case strings.Contains(lower, "warning:"):
			fmt.Printf("  %s\n", color.YellowString(line))
		default:
			fmt.Printf("  %s\n", color.New(color.Faint).Sprint(line))
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
