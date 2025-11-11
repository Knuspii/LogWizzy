package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

const Version = "0.2"

// MessageGroup represents a group of identical log messages
// storing sample text, count of occurrences, log level, and timestamps.
type MessageGroup struct {
	Sample string
	Count  int
	Level  string
	Times  []time.Time
}

// mapPriority maps journalctl numeric/text PRIORITY values to human-readable log levels.
func mapPriority(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	switch p {
	case "0", "emerg", "emergency", "1", "alert", "2", "crit":
		return "CRIT"
	case "3", "err", "error":
		return "ERRO"
	case "4", "warn", "warning":
		return "WARN"
	case "5", "notice", "info":
		return "INFO"
	default:
		return "UNKN"
	}
}

// colorForLevel returns ANSI color codes for different log levels for terminal output.
func colorForLevel(level string) string {
	switch level {
	case "CRIT", "ERRO":
		return "\033[31m" // red for critical/error
	case "WARN":
		return "\033[33m" // yellow for warning
	case "INFO":
		return "\033[32m" // green for info
	default:
		return "\033[37m" // gray for unknown
	}
}

// spinner shows a simple rotating animation while logs are being read.
// Stops when the done channel is closed.
func spinner(done <-chan bool) {
	spinChars := []rune{'|', '/', '-', '\\'}
	i := 0
	fmt.Printf("\n")
	for {
		select {
		case <-done:
			fmt.Printf("\r\033[K") // clear spinner line
			return
		default:
			fmt.Printf("\rLoading logs... %c ", spinChars[i%len(spinChars)])
			time.Sleep(100 * time.Millisecond)
			i++
		}
	}
}

// parseTimestamp converts a string representing microseconds since epoch to time.Time
func parseTimestamp(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	if v, err := parseInt64(s); err == nil {
		return time.Unix(0, v*1000), err // microseconds -> nanoseconds
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp format")
}

// parseInt64 safely converts numeric string to int64 ignoring non-digit chars.
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
	// -------------------------------
	// CLI Flags
	// -------------------------------
	since := flag.String("s", "today", "Since when to read logs")
	showVersion := flag.Bool("v", false, "Show version")
	showHelp := flag.Bool("h", false, "Show help")
	all := flag.Bool("a", false, "Show all logs without limit")
	important := flag.Bool("i", false, "Show only important logs (CRIT, ERRO, WARN)")
	errorsOnly := flag.Bool("e", false, "Show only errors (CRIT + ERRO)")

	defaultLimit := 10    // default number of logs to display
	limit := defaultLimit // store limit value
	limitSet := false     // tracks if user manually set -l flag

	// Custom flag function to allow -l without pointer issues
	flag.Func("l", "Number of logs to show (default 10)", func(s string) error {
		v, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		limit = v
		limitSet = true
		return nil
	})

	flag.Parse()

	versionText := fmt.Sprintf("LogWizzy %s", Version)
	nameText := "Made by Knuspii, (M)"

	// -------------------------------
	// Show Help / Version
	// -------------------------------
	if *showHelp {
		fmt.Printf("#[--- LogWizzy Help ---]#\n%s\n%s\n\nUsage:\n  logwizzy [options]\n\nOptions:\n", versionText, nameText)
		fmt.Printf("  -s VALUE   Set start time for logs (default: today)\n")
		fmt.Printf("  -l VALUE   Number of log entries to show (default 10)\n")
		fmt.Printf("  -v         Show version and exit\n")
		fmt.Printf("  -h         Show help\n")
		fmt.Printf("  -a         Show all logs without limit\n")
		fmt.Printf("  -i         Show only important logs (CRIT, ERRO, WARN)\n")
		fmt.Printf("  -e         Show only errors (CRIT + ERRO)\n")
		return
	}
	if *showVersion {
		fmt.Printf("#[--- LogWizzy Version Info ---]#\n%s\n%s\n", versionText, nameText)
		return
	}

	// -------------------------------
	// Print Header
	// -------------------------------
	fmt.Printf("%s\n%s\n", versionText, nameText)

	title := fmt.Sprintf("#[--- LogWizzy Summary (top %d) (since %s) ---]#", limit, *since)
	if *errorsOnly {
		title = fmt.Sprintf("#[--- LogWizzy Errors Only (since %s) ---]#", *since)
	} else if *important {
		title = fmt.Sprintf("#[--- LogWizzy Important Logs (since %s) ---]#", *since)
	} else if *all {
		title = fmt.Sprintf("#[--- LogWizzy Full Log Dump (since %s) ---]#", *since)
	}
	fmt.Printf(title)

	// -------------------------------
	// Start spinner animation
	// -------------------------------
	done := make(chan bool)
	go spinner(done)

	// -------------------------------
	// Run journalctl command
	// -------------------------------
	args := []string{"-o", "json", "--since=" + *since}
	cmd := exec.Command("journalctl", args...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	cmd.Start()

	// Separate goroutine to handle journalctl stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Printf("\njournalctl error: %s\n", scanner.Text())
		}
	}()

	// -------------------------------
	// Parse logs into message groups
	// -------------------------------
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

		pri := "UNKN"
		if p, ok := raw["PRIORITY"].(string); ok {
			pri = mapPriority(p)
		}

		ts := time.Now()
		if tStr, ok := raw["__REALTIME_TIMESTAMP"].(string); ok {
			if tInt, err := parseTimestamp(tStr); err == nil {
				ts = tInt
			}
		}

		// group messages by exact text
		fp := msg
		if g, ok := groups[fp]; ok {
			g.Count++
			g.Times = append(g.Times, ts)
		} else {
			groups[fp] = &MessageGroup{Sample: msg, Count: 1, Level: pri, Times: []time.Time{ts}}
		}
	}

	// convert map to slice for sorting
	var list []*MessageGroup
	for _, g := range groups {
		list = append(list, g)
	}

	// -------------------------------
	// Sort logs
	// Priority: errors first if -e or -i, else by count descending
	// -------------------------------
	sort.Slice(list, func(i, j int) bool {
		priority := map[string]int{"CRIT": 3, "ERRO": 3, "WARN": 2, "INFO": 1, "UNKN": 0}
		pi := priority[list[i].Level]
		pj := priority[list[j].Level]

		if *errorsOnly || *important {
			if pi == pj {
				if list[i].Count == list[j].Count {
					return list[i].Sample < list[j].Sample
				}
				return list[i].Count > list[j].Count
			}
			return pi > pj
		}

		// default: count descending
		if list[i].Count == list[j].Count {
			return list[i].Sample < list[j].Sample
		}
		return list[i].Count > list[j].Count
	})

	done <- true
	fmt.Printf("\r\033[K") // clear spinner line

	// -------------------------------
	// Print logs
	// -------------------------------
	shown := 0
	errorsList := []*MessageGroup{} // collect errors separately

	for _, g := range list {
		// collect errors for additional display at end
		if g.Level == "CRIT" || g.Level == "ERRO" {
			errorsList = append(errorsList, g)
		}

		// filtering for current mode
		if *errorsOnly && !(g.Level == "CRIT" || g.Level == "ERRO") {
			continue
		}
		if *important && !(g.Level == "CRIT" || g.Level == "ERRO" || g.Level == "WARN") {
			continue
		}
		if !*all && !*important && !*errorsOnly && shown >= limit {
			break
		}

		color := colorForLevel(g.Level)
		reset := "\033[0m"
		fmt.Printf("%s[%s] %dx %s%s\n", color, g.Level, g.Count, g.Sample, reset)
		fmt.Printf("---\n")
		shown++
	}

	// -------------------------------
	// Extra: Show all errors at the end in default mode
	// Only if user did not set -l manually
	// -------------------------------
	if !*errorsOnly && !*important && !*all && !limitSet {
		fmt.Printf("#[--- Additional Errors (since %s) ---]#\n", *since)
		for _, g := range list {
			if g.Level == "CRIT" || g.Level == "ERRO" {
				color := colorForLevel(g.Level)
				reset := "\033[0m"
				fmt.Printf("%s[%s] %dx %s%s\n", color, g.Level, g.Count, g.Sample, reset)
				fmt.Printf("---\n")
			}
		}
	}

	cmd.Wait()
	fmt.Println("LogWizzy Done!")
}
