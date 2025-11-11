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

// MessageGroup holds a unique log message, its count, level and timestamps
type MessageGroup struct {
	Sample string
	Count  int
	Level  string
	Times  []time.Time
}

// mapPriority converts journalctl PRIORITY values to readable levels
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

// colorForLevel returns ANSI color codes for log levels
func colorForLevel(level string) string {
	switch level {
	case "CRIT", "ERR":
		return "\033[31m" // red
	case "WARN":
		return "\033[33m" // yellow
	case "INFO":
		return "\033[32m" // green
	default:
		return "\033[37m" // gray
	}
}

// spinner displays a rotating loader until done is closed
func spinner(done <-chan bool) {
	spinChars := []rune{'|', '/', '-', '\\'}
	i := 0
	fmt.Printf("\n")
	for {
		select {
		case <-done:
			fmt.Printf("\r\033[K") // clear line
			return
		default:
			fmt.Printf("\rLoading logs... %c ", spinChars[i%len(spinChars)])
			time.Sleep(100 * time.Millisecond)
			i++
		}
	}
}

// parseTimestamp parses journalctl's __REALTIME_TIMESTAMP into time.Time
func parseTimestamp(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	if v, err := parseInt64(s); err == nil {
		return time.Unix(0, v*1000), err // convert microseconds to nanoseconds
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp format")
}

// parseInt64 parses numeric strings safely
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
	since := flag.String("s", "today", "Since when to read logs (journalctl format)")
	showVersion := flag.Bool("v", false, "Show version")
	showHelp := flag.Bool("h", false, "Show help")
	all := flag.Bool("a", false, "Show all log lines without limit (alias: --all)")

	versionText := fmt.Sprintf("LogWizzy %s", Version)
	nameText := "Made by Knuspii, (M)"

	flag.Parse()

	// Handle help
	if *showHelp {
		fmt.Printf("%s\n", versionText)
		fmt.Printf("%s\n", nameText)
		fmt.Printf("Usage: logwizzy [options]\n")
		fmt.Printf("Options:\n")
		fmt.Printf("  -s VALUE   Since when to read logs (default: today)\n")
		fmt.Printf("  -v         Show version\n")
		fmt.Printf("  -h         Show help\n")
		fmt.Printf("  -a         Show all log lines without limit\n")
		return
	}

	// Handle version
	if *showVersion {
		fmt.Printf("%s\n", versionText)
		fmt.Printf("%s\n", nameText)
		return
	}

	fmt.Printf("%s\n", versionText)
	fmt.Printf("%s\n", nameText)

	// Start spinner
	done := make(chan bool)
	go spinner(done)

	// Prepare journalctl command
	args := []string{"-o", "json", "--since=" + *since}
	cmd := exec.Command("journalctl", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("\nFailed to get stdout: %v\n", err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("\nFailed to get stderr: %v\n", err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("\njournalctl start failed: %v\n", err)
		return
	}

	// Goroutine to print any errors from journalctl
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Printf("\njournalctl error: %s\n", scanner.Text())
		}
	}()

	// Parse logs and group by message
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

	// Convert map to slice and sort by count
	list := []*MessageGroup{}
	for _, g := range groups {
		list = append(list, g)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Count > list[j].Count })

	// Stop spinner
	done <- true
	fmt.Printf("\r\033[K") // clear spinner line

	// Print summary
	fmt.Printf("#[--- LogWizzy Summary (since %s) ---]#\n", *since)
	for i, g := range list {
		if i >= 30 && !*all {
			break
		}
		color := colorForLevel(g.Level)
		reset := "\033[0m"
		fmt.Printf("%s[%s] %dx %s%s\n", color, g.Level, g.Count, g.Sample, reset)
		fmt.Printf("---\n")
	}

	cmd.Wait()
	fmt.Println("LogWizzy Done!")
}
