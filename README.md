# adhocore/gronx

[![Latest Version](https://img.shields.io/github/release/adhocore/gronx.svg?style=flat-square)](https://github.com/adhocore/gronx/releases)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE)
[![Go Report](https://goreportcard.com/badge/github.com/adhocore/gronx)](https://goreportcard.com/report/github.com/adhocore/gronx)
[![Test](https://github.com/adhocore/gronx/actions/workflows/test-action.yml/badge.svg)](https://github.com/adhocore/gronx/actions/workflows/test-action.yml)
[![Donate](https://img.shields.io/badge/donate-paypal-blue.svg?style=flat-square)](https://www.paypal.me/ji10/50usd)
[![Tweet](https://img.shields.io/twitter/url/http/shields.io.svg?style=social)](https://twitter.com/intent/tweet?text=Lightweight+fast+and+deps+free+cron+expression+parser+for+Golang&url=https://github.com/adhocore/gronx&hashtags=go,golang,parser,cron,cronexpr,cronparser)


`gronx` is Golang cron expression parser ported from [adhocore/cron-expr](https://github.com/adhocore/php-cron-expr).

- Zero dependency.
- Very **fast** because it bails early in case a segment doesn't match.

## Installation

```sh
go get -u github.com/adhocore/gronx
```

## Usage

```go
import (
	"time"

	"github.com/adhocore/gronx"
)

gron := gronx.New()
expr := "* * * * *"

// check if expr is even valid, returns bool
gron.IsValid(expr) // true

// check if expr is due for current time, returns bool and error
gron.IsDue(expr) // true|false, nil

// check if expr is due for given time
gron.IsDue(expr, time.Date(2021, time.April, 1, 1, 1, 0, 0, time.UTC)) // true|false, nil
```

In a more practical level, you would use this tool to manage and invoke jobs in app itself and not
mess around with `crontab` for each and every new tasks/jobs. ~~It doesn't yet replace that but rather supplements it.
There is a plan though [#1](https://github.com/adhocore/gronx/issues/1)~~.

In crontab just put one entry with `* * * * *` which points to your Go entry point that uses this tool.
Then in that entry point you would invoke different tasks if the corresponding Cron expr is due.
Simple map structure would work for this.

Check the section below for more sophisticated way of managing tasks automatically using `gronx` daemon called `tasker`.

---
### Go Tasker

Tasker is a task manager that can be programatically used in Golang applications. It runs as a daemon and and invokes tasks scheduled with cron expression:
```go
package main

import (
	"context"
	"time"

	"github.com/adhocore/gronx/pkg/tasker"
)

func main() {
	taskr := tasker.New(tasker.Option{
		Verbose: true,
		// optional: defaults to local
		Tz:      "Asia/Bangkok",
		// optional: defaults to stderr log stream
		Out:     "/full/path/to/output-file",
	})

	// add task to run every minute
	taskr.Task("* * * * *", func(ctx context.Context) (int, error) {
		// do something ...

		// then return exit code and error, for eg: if everything okay
		return 0, nil
	}).Task("*/5 * * * *", func(ctx context.Context) (int, error) { // every 5 minutes
		// you can also log the output to Out file as configured in Option above:
		taskr.Log.Printf("done something in %d s", 2)

		return 0, nil
	})

	// every 10 minute with arbitrary command
	taskr.Task("@10minutes", taskr.Taskify("command --option val -- args"))

	// ... add more tasks

	// optionally if you want tasker to stop after 2 hour, pass the duration with Until():
	taskr.Until(2 * time.Hour)

	// finally run the tasker, it ticks sharply on every minute and runs all the tasks due on that time!
	// it exits gracefully when ctrl+c is received making sure pending tasks are completed.
	taskr.Run()
}
```

### Task Daemon
It can also be used as standalone task daemon instead of programmatic usage for Golang application.

First, just install tasker command:
```sh
go get -u github.com/adhocore/gronx/cmd/tasker
```

Then prepare a taskfile ([example](./tests/../test/taskfile.txt)) in crontab format
(or can even point to existing crontab).
> `user` is not supported: it is just cron expr followed by the command.

Finally run the task daemon like so
```
tasker -file path/to/taskfile
```
> You can pass more options to control the behavior of task daemon, see below.

####  Tasker command options:
```txt
-file string <required>
    The task file in crontab format
-out string
    The fullpath to file where output from tasks are sent to
-shell string
    The shell to use for running tasks (default "/usr/bin/bash")
-tz string
    The timezone to use for tasks (default "Local")
-until int
    The timeout for task daemon in minutes
-verbose
    The verbose mode outputs as much as possible
```

Examples:
```sh
tasker -verbose -file path/to/taskfile -until 120 # run until next 120min (i.e 2hour) with all feedbacks echoed back
tasker -verbose -file path/to/taskfile -out path/to/output # with all feedbacks echoed to the output file
tasker -tz America/New_York -file path/to/taskfile -shell zsh # run all tasks using zsh shell based on NY timezone
```

> File extension of taskfile for (`-file` option) does not matter: can be any or none.
> The directory for outfile (`-out` option) must exist, file is created by task daemon.

> Same timezone applies for all tasks currently and it might support overriding timezone per task in future release.

#### Notes on Windows
In Windows if it doesn't find `bash.exe` or `git-bash.exe` it will use `powershell`.
`powershell` may not be compatible with Unix flavored commands. Also to note:
you can't do chaining with `cmd1 && cmd2` but rather `cmd1 ; cmd2`.

---
### Cron Expression

Cron expression normally consists of 5 segments viz:
```
<minute> <hour> <day> <month> <weekday>
```
and sometimes there can be 6th segment for `<year>` at the end.

For each segments you can have multiple choices separated by comma:
> Eg: `0,30 * * * *` means either 0th or 30th minute.

To specify range of values you can use dash:
> Eg: `10-15 * * * *` means 10th, 11th, 12th, 13th, 14th and 15th minute.

To specify range of step you can combine a dash and slash:
> Eg: `10-15/2 * * * *` means every 2 minutes between 10 and 15 i.e 10th, 12th and 14th minute.

For the 3rd and 5th segment, there are additional [modifiers](#modifiers) (optional).

And if you want, you can mix them up:
> `5,12-20/4,55 * * * *` matches if any one of `5` or `12-20/4` or `55` matches the minute.

### Real Abbreviations

You can use real abbreviations for month and week days. eg: `JAN`, `dec`, `fri`, `SUN`

### Tags

Following tags are available and they are converted to real cron expressions before parsing:

- *@yearly* or *@annually* - every year
- *@monthly* - every month
- *@daily* - every day
- *@weekly* - every week
- *@hourly* - every hour
- *@5minutes* - every 5 minutes
- *@10minutes* - every 10 minutes
- *@15minutes* - every 15 minutes
- *@30minutes* - every 30 minutes
- *@always* - every minute

```go
gron.IsDue("@5minutes")
```

### Modifiers

Following modifiers supported

- *Day of Month / 3rd segment:*
    - `L` stands for last day of month (eg: `L` could mean 29th for February in leap year)
    - `W` stands for closest week day (eg: `10W` is closest week days (MON-FRI) to 10th date)
- *Day of Week / 5th segment:*
    - `L` stands for last weekday of month (eg: `2L` is last monday)
    - `#` stands for nth day of week in the month (eg: `1#2` is second sunday)

---
## License

> &copy; [MIT](./LICENSE) | 2021-2099, Jitendra Adhikari

## Credits

This project is ported from [adhocore/cron-expr](https://github.com/adhocore/php-cron-expr) and
release managed by [please](https://github.com/adhocore/please).

---
### Other projects
My other golang projects you might find interesting and useful:

- [**urlsh**](https://github.com/adhocore/urlsh) - URL shortener and bookmarker service with UI, API, Cache, Hits Counter and forwarder using postgres and redis in backend, bulma in frontend; has [web](https://urlssh.xyz) and cli client
- [**fast**](https://github.com/adhocore/fast) - Check your internet speed with ease and comfort right from the terminal
