package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const Version = "0.1"

type MessageGroup struct {
	Sample string
	Count  int
	Level  string
	Times  []time.Time
}

func mapPriority(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	switch p {
	case "0", "emerg", "emergency", "1", "alert", "2", "crit":
		return "CRIT"
	case "3", "err", "error":
		return "ERR"
	case "4", "warn", "warning":
		return "WARN"
	case "5", "notice", "info":
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

func colorForLevel(level string) string {
	switch level {
	case "CRIT", "ERR":
		return "\033[31m" // rot
	case "WARN":
		return "\033[33m" // gelb
	case "INFO":
		return "\033[32m" // grün
	default:
		return "\033[37m" // grau
	}
}

func truncateCompact(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func parseTimestamp(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty")
	}
	if v, err := parseInt64(s); err == nil {
		return time.Unix(0, v*1000), nil
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp")
}

func parseInt64(s string) (int64, error) {
	var v int64
	for _, r := range s {
		if r >= '0' && r <= '9' {
			v = v*10 + int64(r-'0')
		}
	}
	return v, nil
}

func main() {
	// CLI flags
	since := flag.String("s", "today", "Since when to read logs (journalctl format) (alias: --since)")
	showVersion := flag.Bool("v", false, "Show version (alias: --version)")
	showHelp := flag.Bool("h", false, "Show help (alias: --help)")
	full := flag.Bool("f", false, "Show full log lines (alias: --full)")
	all := flag.Bool("a", false, "Show all log lines without limit (alias: --all)")

	flag.Parse()

	if *showHelp {
		fmt.Printf("LogWizard %s\n", Version)
		fmt.Printf("Made by Knuspii, (M)\n")
		fmt.Printf("Usage: logwizard [options]\n")
		fmt.Printf("Options:\n")
		fmt.Printf("  -s VALUE   Since when to read logs (default: today, alias: --since)\n")
		fmt.Printf("  -v         Show version (alias: --version)\n")
		fmt.Printf("  -h         Show help (alias: --help)\n")
		fmt.Printf("  -f         Show full log lines (alias: --full)\n")
		fmt.Printf("  -a         Show all log lines without limit (alias: --all)\n")
		return
	}

	if *showVersion {
		fmt.Printf("LogWizard version %s\n", Version)
		fmt.Printf("Made by Knuspii, (M)\n")
		return
	}

	args := []string{"-o", "json", "--since=" + *since}
	cmd := exec.Command("journalctl", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Failed to start journalctl: %v\n", err)
		return
	}
	if err := cmd.Start(); err != nil {
		fmt.Printf("journalctl start failed: %v\n", err)
		return
	}

	groups := map[string]*MessageGroup{}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		msg := ""
		if m, ok := raw["MESSAGE"].(string); ok {
			msg = m
		}

		pri := "UNKNOWN"
		if p, ok := raw["PRIORITY"].(string); ok {
			pri = mapPriority(p)
		}

		ts := time.Now()
		if tStr, ok := raw["__REALTIME_TIMESTAMP"].(string); ok {
			if tInt, err := parseTimestamp(tStr); err == nil {
				ts = tInt
			}
		}

		fp := msg
		if g, ok := groups[fp]; ok {
			g.Count++
			g.Times = append(g.Times, ts)
		} else {
			groups[fp] = &MessageGroup{Sample: msg, Count: 1, Level: pri, Times: []time.Time{ts}}
		}
	}

	list := []*MessageGroup{}
	for _, g := range groups {
		list = append(list, g)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Count > list[j].Count })

	fmt.Printf("#[--- LogWizard Summary (since %s) ---]#\n", *since)
	for i, g := range list {
		if i >= 30 && !*all {
			break
		}
		color := colorForLevel(g.Level)
		reset := "\033[0m"

		if *full {
			fmt.Printf("%s[%s] %dx %s%s\n", color, g.Level, g.Count, g.Sample, reset)
		} else {
			fmt.Printf("%s[%s] %dx %s%s\n", color, g.Level, g.Count, truncateCompact(g.Sample, 60), reset)
		}
	}

	cmd.Wait()
	fmt.Printf("LogWizard Done!\n")
}
