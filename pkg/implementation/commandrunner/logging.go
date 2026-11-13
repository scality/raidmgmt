package commandrunner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"
)

// ExecutedCommand describes a single vendor CLI invocation, as handed to
// LogCommand.
type ExecutedCommand struct {
	// Path is the binary that was invoked.
	Path string

	// Args are the arguments the caller asked for.
	Args []string

	// OutputBytes is the size of the command output. The output itself is
	// never logged: vendor payloads carry drive serials and other identifying
	// data. A runner that exposes no payload size leaves this zero, and the
	// attribute is then omitted.
	OutputBytes int

	// Duration is how long the invocation took.
	Duration time.Duration

	// Err is the error the invocation returned, if any.
	Err error
}

// LogCommand writes one record describing an executed command: a successful
// invocation at level, a failed one at slog.LevelError. ErrNoLogicalDrives
// reports an empty inventory rather than a failure (see the SSACLI runner), so
// it is recorded at level too, with its error attached.
//
// It is exported so that a command runner outside this package -- the legacy
// megaraid.Runner, whose Run returns parsed output instead of bytes -- logs in
// the same shape.
func LogCommand(logger *slog.Logger, level slog.Level, cmd ExecutedCommand) {
	attrs := []slog.Attr{
		slog.String("command", cmd.Path),
		slog.Any("args", cmd.Args),
		slog.Duration("duration", cmd.Duration),
	}

	if cmd.OutputBytes > 0 {
		attrs = append(attrs, slog.Int("output_bytes", cmd.OutputBytes))
	}

	recordLevel, message := level, "RAID command executed"

	if cmd.Err != nil {
		attrs = append(attrs, slog.String("error", cmd.Err.Error()))

		if !errors.Is(cmd.Err, ErrNoLogicalDrives) {
			recordLevel, message = slog.LevelError, "RAID command failed"
		}
	}

	logger.LogAttrs(context.Background(), recordLevel, message, attrs...)
}

// Logging decorates a CommandRunner so that every command it executes is
// recorded on an slog.Logger: the binary invoked, its arguments, how long it
// took and the outcome. The decorator is transparent -- output and error are
// returned unchanged -- so it wraps any runner without altering adapter
// behaviour.
//
// The arguments recorded are the ones the adapter asked for. A runner that adds
// flags of its own does so after this decorator has seen the arguments, so
// those flags are absent from the record: a storcli2 or perccli2 command also
// carries the JSON output flag on the wire.
type Logging struct {
	runner CommandRunner
	logger *slog.Logger
	level  slog.Level
}

// LoggingOption configures a Logging runner.
type LoggingOption func(*Logging)

// WithLevel sets the level successful commands are recorded at. It defaults to
// slog.LevelInfo; failures are always recorded at slog.LevelError.
func WithLevel(level slog.Level) LoggingOption {
	return func(l *Logging) {
		l.level = level
	}
}

// commandPather is implemented by the runners of this package to report the
// binary they invoke, so that a log record names the tool that ran.
type commandPather interface {
	CommandPath() string
}

var _ CommandRunner = &Logging{}

// NewLogging wraps runner so that every command it runs is logged to logger. A
// nil logger falls back to slog.Default().
func NewLogging(runner CommandRunner, logger *slog.Logger, opts ...LoggingOption) *Logging {
	target := logger
	if target == nil {
		target = slog.Default()
	}

	logging := &Logging{
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
func (l *Logging) Run(args []string) ([]byte, error) {
	start := time.Now()

	output, err := l.runner.Run(args)

	LogCommand(l.logger, l.level, ExecutedCommand{
		Path:        l.CommandPath(),
		Args:        args,
		OutputBytes: len(output),
		Duration:    time.Since(start),
		Err:         err,
	})

	return output, err //nolint:wrapcheck // Transparent decorator: error returned as-is.
}

// CommandPath reports the binary the wrapped runner invokes, which also keeps
// the decorator itself transparent to another decorator wrapping it. A runner
// that does not report its binary is named by its type.
func (l *Logging) CommandPath() string {
	if pather, ok := l.runner.(commandPather); ok {
		return pather.CommandPath()
	}

	return fmt.Sprintf("%T", l.runner)
}
