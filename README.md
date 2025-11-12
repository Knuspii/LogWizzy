[![Golang](https://img.shields.io/static/v1?label=Made%20with&message=Go&logo=go&color=007ACC)](https://go.dev/)
[![Go version](https://img.shields.io/github/go-mod/go-version/knuspii/logwizzy)](https://github.com/knuspii/logwizzy)
[![Go Report Card](https://goreportcard.com/badge/github.com/Knuspii/logwizzy)](https://goreportcard.com/report/github.com/Knuspii/logwizzy)
[![Build](https://github.com/Knuspii/logwizzy/actions/workflows/go.yml/badge.svg)](https://github.com/Knuspii/logwizzy/actions/workflows/go.yml)
[![GitHub Issues](https://img.shields.io/github/issues/knuspii/logwizzy)](https://github.com/knuspii/logwizzy/issues)
[![GitHub Stars](https://img.shields.io/github/stars/knuspii/logwizzy?style=social)](https://github.com/knuspii/logwizzy/stargazers)

<h1>LogWizzy 🧙</h1>

✨ LogWizzy — tame your journalctl madness.

LogWizzy aggregates and groups systemd journal logs, highlights severity with colors, counts repeated messages, and optionally shows full logs for quick monitoring.

## 📥 [[Download here]](https://github.com/Knuspii/logwizzy/releases) <- Click here to download LogWizzy!

## ⚙️ Start options:
```
Usage:
  logwizzy [options]

Options:
  No option   Show top 10 and error summary
  -s <value>  Set start time for logs (default: today)
  -l <value>  Number of log entries to show (default 10)
  -v          Show version and exit
  -h          Show help
  -a          Show all logs without limit
  -i          Show only important logs (CRIT, WARN)
  -e          Show only errors (CRIT)
  ```

  
