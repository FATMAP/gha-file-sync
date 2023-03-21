package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/exp/slog"
)

// The log record contains the source position of the caller of Infof.
func Infof(format string, args ...any) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.LevelInfo) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, Infof]
	r := slog.NewRecord(time.Now(), slog.LevelInfo, fmt.Sprintf(format, args...), pcs[0])
	_ = l.Handler().Handle(context.Background(), r)
}

// The log record contains the source position of the caller of Errorf.
func Errorf(format string, args ...any) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.LevelError) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, Errorf]
	r := slog.NewRecord(time.Now(), slog.LevelError, fmt.Sprintf(format, args...), pcs[0])
	_ = l.Handler().Handle(context.Background(), r)
}

// The log record contains the source position of the caller of Warnf.
func Warnf(format string, args ...any) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.LevelWarn) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, Warnf]
	r := slog.NewRecord(time.Now(), slog.LevelWarn, fmt.Sprintf(format, args...), pcs[0])
	_ = l.Handler().Handle(context.Background(), r)
}

func Init() {
	replace := func(groups []string, a slog.Attr) slog.Attr {
		// Remove time.
		if a.Key == slog.TimeKey && len(groups) == 0 {
			a.Key = ""
		}
		// Remove the directory from the source's filename.
		if a.Key == slog.SourceKey {
			a.Value = slog.StringValue(filepath.Base(a.Value.String()))
		}
		return a
	}
	logger := slog.New(slog.HandlerOptions{AddSource: true, ReplaceAttr: replace}.NewTextHandler(os.Stdout))
	slog.SetDefault(logger)
	Infof("logger init done")
}
