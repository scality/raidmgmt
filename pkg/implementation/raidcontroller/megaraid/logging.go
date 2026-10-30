package megaraid

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
)

// commandPather is implemented by MegaRAIDRunner to report the binary it
// invokes, so that a log record names the tool that ran.
type commandPather interface {
	CommandPath() string
}

// LoggingRunner decorates a Runner so that every storcli/perccli command it
// executes is recorded on an slog.Logger, in the same shape as the decomposed
// adapters' commandrunner.Logging. The decorator is transparent: the parsed
// output and the error are returned unchanged.
//
// This runner parses its own output, so no payload size is recorded; the
// arguments recorded are the ones the adapter asked for, without the JSON
// output flag Run appends on the wire.
type LoggingRunner struct {
	runner Runner
	logger *slog.Logger
	level  slog.Level
}

// LoggingRunnerOption configures a LoggingRunner.
type LoggingRunnerOption func(*LoggingRunner)

// WithLevel sets the level successful commands are recorded at. It defaults to
// slog.LevelInfo; failures are always recorded at slog.LevelError.
func WithLevel(level slog.Level) LoggingRunnerOption {
	return func(l *LoggingRunner) {
		l.level = level
	}
}

var _ Runner = &LoggingRunner{}

// NewLoggingRunner wraps runner so that every command it runs is logged to
// logger. A nil logger falls back to slog.Default().
func NewLoggingRunner(
	runner Runner,
	logger *slog.Logger,
	opts ...LoggingRunnerOption,
) *LoggingRunner {
	target := logger
	if target == nil {
		target = slog.Default()
	}

	logging := &LoggingRunner{
		runner: runner,
		logger: target,
		level:  slog.LevelInfo,
	}

	for _, opt := range opts {
		opt(logging)
	}

	return logging
}

// Run logs the command, then returns the wrapped runner's output and error
// unchanged.
func (l *LoggingRunner) Run(args []string) (*CmdOutput, error) {
	start := time.Now()

	output, err := l.runner.Run(args)

	commandrunner.LogCommand(l.logger, l.level, commandrunner.ExecutedCommand{
		Path:     l.CommandPath(),
		Args:     args,
		Duration: time.Since(start),
		Err:      err,
	})

	return output, err //nolint:wrapcheck // Transparent decorator: error returned as-is.
}

// CommandPath reports the binary the wrapped runner invokes. A runner that does
// not report its binary is named by its type.
func (l *LoggingRunner) CommandPath() string {
	if pather, ok := l.runner.(commandPather); ok {
		return pather.CommandPath()
	}

	return fmt.Sprintf("%T", l.runner)
}
