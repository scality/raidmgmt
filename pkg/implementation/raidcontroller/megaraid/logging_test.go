package megaraid_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/implementation/raidcontroller/megaraid"
	"github.com/scality/raidmgmt/pkg/implementation/raidcontroller/megaraid/mocks"
)

// loggingRecords decodes the JSON log records captured in buf.
func loggingRecords(t *testing.T, buf *bytes.Buffer) []map[string]any {
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

func newLoggingTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestLoggingRunnerLogsExecutedCommand(t *testing.T) {
	var buf bytes.Buffer

	parsed := &megaraid.CmdOutput{}

	runner := &mocks.Runner{}
	runner.On("Run", []string{"/c0", "show", "all"}).Return(parsed, nil)

	output, err := megaraid.NewLoggingRunner(runner, newLoggingTestLogger(&buf)).
		Run([]string{"/c0", "show", "all"})
	require.NoError(t, err)
	assert.Same(t, parsed, output)

	logged := loggingRecords(t, &buf)
	require.Len(t, logged, 1)

	assert.Equal(t, "INFO", logged[0]["level"])
	assert.Equal(t, "RAID command executed", logged[0]["msg"])
	assert.Equal(t, []any{"/c0", "show", "all"}, logged[0]["args"])
	assert.Contains(t, logged[0], "duration")
	// This runner parses its own output, so no payload size is recorded.
	assert.NotContains(t, logged[0], "output_bytes")

	runner.AssertExpectations(t)
}

func TestLoggingRunnerLogsFailureAtErrorLevel(t *testing.T) {
	var buf bytes.Buffer

	wrapped := errors.New("no controllers found")

	runner := &mocks.Runner{}
	runner.On("Run", []string{"show"}).Return(nil, wrapped)

	output, err := megaraid.NewLoggingRunner(
		runner,
		newLoggingTestLogger(&buf),
		megaraid.WithLevel(slog.LevelDebug),
	).Run([]string{"show"})
	require.ErrorIs(t, err, wrapped)
	assert.Nil(t, output)

	logged := loggingRecords(t, &buf)
	require.Len(t, logged, 1)

	assert.Equal(t, "ERROR", logged[0]["level"])
	assert.Equal(t, "RAID command failed", logged[0]["msg"])
	assert.Equal(t, "no controllers found", logged[0]["error"])
}

// The mock runner does not report a binary path, so the record names it by type.
func TestLoggingRunnerNamesRunnerWithoutCommandPathByType(t *testing.T) {
	var buf bytes.Buffer

	runner := &mocks.Runner{}
	runner.On("Run", []string{"show"}).Return(&megaraid.CmdOutput{}, nil)

	logging := megaraid.NewLoggingRunner(runner, newLoggingTestLogger(&buf))
	assert.Equal(t, "*mocks.Runner", logging.CommandPath())

	_, err := logging.Run([]string{"show"})
	require.NoError(t, err)

	logged := loggingRecords(t, &buf)
	require.Len(t, logged, 1)
	assert.Equal(t, "*mocks.Runner", logged[0]["command"])
}
