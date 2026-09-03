package console

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jorgefuertes/QDAAD/internal/util"
)

var output io.Writer

func isTerminal() bool {
	return output != nil && output == os.Stdout
}

type Level uint8

const (
	LevelNone Level = iota
	LevelError
	LevelOk
	LevelWarn
	LevelInfo
	LevelDebug
)

func (l Level) String() string {
	switch l {
	case LevelNone:
		return "NONE"
	case LevelError:
		return "ERROR"
	case LevelOk:
		return "OK"
	case LevelWarn:
		return "WARNING"
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

func (l Level) style() lipgloss.Style {
	switch l {
	case LevelError:
		return ErrStyle
	case LevelOk:
		return OkStyle
	case LevelWarn:
		return WarnStyle
	case LevelInfo:
		return ValueStyle
	case LevelDebug:
		return PathStyle
	default:
		return MutedStyle
	}
}

const (
	UpCaret    = "▲" // U+25B2 BLACK UP-POINTING TRIANGLE
	DownCaret  = "▼" // U+25BC BLACK DOWN-POINTING TRIANGLE
	LeftCaret  = "◀" // U+25C0 BLACK LEFT-POINTING TRIANGLE
	RightCaret = "▶" // U+25B6 BLACK RIGHT-POINTING TRIANGLE
)

func Sayf(level Level, format string, args ...any) {
	fmt.Printf("%s %s\n", level.style().Render(RightCaret), fmt.Sprintf(format, args...))
}

func Say(level Level, args ...any) {
	fmt.Printf("%s %s", level.style().Render(RightCaret), fmt.Sprint(args...))
}

func Sayln(level Level, args ...any) {
	Say(level, args...)
	fmt.Println()
}

func Make(desc string, fn func() error) error {
	Say(LevelInfo, desc, "...")
	start := time.Now()

	end := func(err error) {
		if err != nil {
			if isTerminal() {
				fmt.Printf("\r%s %s...%s...%s\n",
					ErrStyle.Render(DownCaret),
					desc,
					MutedStyle.Render(util.HumanDuration(time.Since(start))),
					ErrStyle.Render(err.Error()),
				)
			}
		} else {
			fmt.Printf("%s...%s\n",
				MutedStyle.Render(util.HumanDuration(time.Since(start))),
				OkStyle.Render("OK"),
			)
		}
	}

	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		end(err)

		return err
	}
	defer null.Close()

	stdout, stderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = null, null

	// A defer, so that a panic inside fn cannot leave the process mute.
	defer func() { os.Stdout, os.Stderr = stdout, stderr }()

	err = fn()
	os.Stdout, os.Stderr = stdout, stderr
	end(err)

	return err
}
