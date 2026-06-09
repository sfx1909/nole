package output

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

type Summary struct {
	Warnings     int
	Deprecations int
	Built        int
	lines        []string
}

func NewSummary() *Summary {
	return &Summary{}
}

func (s *Summary) Parse(line string) {
	s.lines = append(s.lines, line)

	lower := strings.ToLower(line)
	if strings.Contains(lower, "warning:") {
		s.Warnings++
	}
	if strings.Contains(lower, "deprecated") {
		s.Deprecations++
	}
	if strings.Contains(line, ".drv") && strings.Contains(line, "building") {
		s.Built++
	}
}

func (s *Summary) Print() {
	fmt.Println()
	fmt.Println(color.New(color.Bold).Sprint("  Summary"))
	fmt.Printf("  %s  %d built\n", color.CyanString(""), s.Built)
	if s.Warnings > 0 {
		fmt.Printf("  %s  %d warnings\n", color.YellowString(""), s.Warnings)
	}
	if s.Deprecations > 0 {
		fmt.Printf("  %s  %d deprecations\n", color.MagentaString(""), s.Deprecations)
	}
	fmt.Println()
}
