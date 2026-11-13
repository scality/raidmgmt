package commandrunner_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
)

// mockCommandRunner is a CommandRunner that also reports its binary path, like
// every concrete runner of the package.
type mockCommandRunner struct {
	mock.Mock
}

func (m *mockCommandRunner) Run(args []string) ([]byte, error) {
	arguments := m.Called(args)

	output, _ := arguments.Get(0).([]byte)

	return output, arguments.Error(1)
}

func (m *mockCommandRunner) CommandPath() string {
	return "/opt/vendor/cli"
}

// pathlessRunner is a CommandRunner that does not report a binary path.
type pathlessRunner struct{}

func (pathlessRunner) Run(_ []string) ([]byte, error) {
	return []byte("ok"), nil
}

// newCapturingLogger returns a logger writing JSON records to buf, at a level
// low enough to capture every record the decorator can emit.
func newCapturingLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// records decodes the captured JSON log records.
func records(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()

	var decoded []map[string]any

	for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		if len(line) == 0 {
			continue
		}

		var record map[string]any

		require.NoError(t, json.Unmarshal(line, &record))

		decoded = append(decoded, record)
	}

	return decoded
}

func TestLoggingLogsExecutedCommand(t *testing.T) {
	var buf bytes.Buffer

	runner := &mockCommandRunner{}
	runner.On("Run", []string{"/c0", "show", "all"}).Return([]byte(`{"Controllers":[]}`), nil)

	output, err := commandrunner.NewLogging(runner, newCapturingLogger(&buf)).
		Run([]string{"/c0", "show", "all"})
	require.NoError(t, err)
	assert.Equal(t, `{"Controllers":[]}`, string(output))

	logged := records(t, &buf)
	require.Len(t, logged, 1)

	assert.Equal(t, "INFO", logged[0]["level"])
	assert.Equal(t, "RAID command executed", logged[0]["msg"])
	assert.Equal(t, "/opt/vendor/cli", logged[0]["command"])
	assert.Equal(t, []any{"/c0", "show", "all"}, logged[0]["args"])
	assert.EqualValues(t, len(`{"Controllers":[]}`), logged[0]["output_bytes"])
	assert.Contains(t, logged[0], "duration")
	assert.NotContains(t, logged[0], "error")

	runner.AssertExpectations(t)
}

func TestLoggingLogsFailureAtErrorLevel(t *testing.T) {
	var buf bytes.Buffer

	wrapped := errors.New("exit status 1")

	runner := &mockCommandRunner{}
	runner.On("Run", []string{"/c0", "show"}).Return(nil, wrapped)

	output, err := commandrunner.NewLogging(runner, newCapturingLogger(&buf)).
		Run([]string{"/c0", "show"})
	require.ErrorIs(t, err, wrapped)
	assert.Nil(t, output)

	logged := records(t, &buf)
	require.Len(t, logged, 1)

	assert.Equal(t, "ERROR", logged[0]["level"])
	assert.Equal(t, "RAID command failed", logged[0]["msg"])
	assert.Equal(t, "exit status 1", logged[0]["error"])
	// No payload was returned, so no size is recorded.
	assert.NotContains(t, logged[0], "output_bytes")
}

// An ssacli controller without any logical drive is an empty inventory, not a
// failure, so it must not be reported at error level.
func TestLoggingLogsNoLogicalDrivesAtSuccessLevel(t *testing.T) {
	var buf bytes.Buffer

	runner := &mockCommandRunner{}
	runner.On("Run", []string{"ctrl", "slot=0", "ld", "all", "show"}).
		Return([]byte("Error: ..."), commandrunner.ErrNoLogicalDrives)

	_, err := commandrunner.NewLogging(runner, newCapturingLogger(&buf)).
		Run([]string{"ctrl", "slot=0", "ld", "all", "show"})
	require.ErrorIs(t, err, commandrunner.ErrNoLogicalDrives)

	logged := records(t, &buf)
	require.Len(t, logged, 1)

	assert.Equal(t, "INFO", logged[0]["level"])
	assert.Equal(t, "RAID command executed", logged[0]["msg"])
	assert.Equal(t, commandrunner.ErrNoLogicalDrives.Error(), logged[0]["error"])
}

func TestLoggingWithLevel(t *testing.T) {
	var buf bytes.Buffer

	runner := &mockCommandRunner{}
	runner.On("Run", []string{"show"}).Return([]byte("out"), nil)

	logging := commandrunner.NewLogging(
		runner,
		newCapturingLogger(&buf),
		commandrunner.WithLevel(slog.LevelDebug),
	)

	_, err := logging.Run([]string{"show"})
	require.NoError(t, err)

	logged := records(t, &buf)
	require.Len(t, logged, 1)
	assert.Equal(t, "DEBUG", logged[0]["level"])
}

// A logger filtering out the success level drops the record: the decorator
// still returns the command's result untouched.
func TestLoggingHonoursHandlerLevel(t *testing.T) {
	var buf bytes.Buffer

	runner := &mockCommandRunner{}
	runner.On("Run", []string{"show"}).Return([]byte("out"), nil)

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	output, err := commandrunner.NewLogging(runner, logger).Run([]string{"show"})
	require.NoError(t, err)
	assert.Equal(t, "out", string(output))
	assert.Empty(t, buf.String())
}

func TestLoggingNamesRunnerWithoutCommandPathByType(t *testing.T) {
	var buf bytes.Buffer

	logging := commandrunner.NewLogging(pathlessRunner{}, newCapturingLogger(&buf))
	assert.Equal(t, "commandrunner_test.pathlessRunner", logging.CommandPath())

	_, err := logging.Run([]string{"show"})
	require.NoError(t, err)

	logged := records(t, &buf)
	require.Len(t, logged, 1)
	assert.Equal(t, "commandrunner_test.pathlessRunner", logged[0]["command"])
}

// A nil logger must not panic: it falls back to slog.Default().
func TestLoggingWithNilLoggerUsesDefault(t *testing.T) {
	var buf bytes.Buffer

	original := slog.Default()
	defer slog.SetDefault(original)

	slog.SetDefault(newCapturingLogger(&buf))

	runner := &mockCommandRunner{}
	runner.On("Run", []string{"show"}).Return([]byte("out"), nil)

	_, err := commandrunner.NewLogging(runner, nil).Run([]string{"show"})
	require.NoError(t, err)

	logged := records(t, &buf)
	require.Len(t, logged, 1)
	assert.Equal(t, "/opt/vendor/cli", logged[0]["command"])
}

// Every concrete runner reports the binary it invokes, so that its commands are
// logged under the tool that ran them.
func TestRunnersReportCommandPath(t *testing.T) {
	custom := "/opt/custom/cli"

	testCases := []struct {
		name     string
		runner   commandrunner.CommandRunner
		expected string
	}{
		{"storcli2", commandrunner.NewStorCLI2(nil), commandrunner.StorCLI2Path},
		{"storcli2-custom", commandrunner.NewStorCLI2(&custom), custom},
		{"perccli2", commandrunner.NewPercCLI2(nil), commandrunner.PercCLI2Path},
		{"ssacli", commandrunner.NewSSACLI(nil), commandrunner.SSACLIPath},
		{"mdadm", commandrunner.NewMDADM(nil), commandrunner.MDADMBinaryPath},
		{"lsblk", commandrunner.NewLSBLK(nil), commandrunner.LSBLKBinaryPath},
		{"smartctl", commandrunner.NewSmartCTL(nil), commandrunner.SmartCTLBinaryPath},
		{"udevadm", commandrunner.NewUDevADM(nil), commandrunner.UDevADMBinaryPath},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pather, ok := tc.runner.(interface{ CommandPath() string })
			require.True(t, ok, "runner does not report its command path")
			assert.Equal(t, tc.expected, pather.CommandPath())

			// The decorator reports the wrapped runner's binary as its own, so
			// that wrapping stays transparent.
			assert.Equal(t, tc.expected, commandrunner.NewLogging(tc.runner, nil).CommandPath())
		})
	}
}
