package scenario

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reported is what the fake controller of the testdata features reports.
var reported = []Disk{{Slot: "251:1", Status: "Failed", Serial: "A"}} //nolint:gochecknoglobals // test data

// playFixture runs testdata/<name>.feature on a fake controller reporting
// got, and returns the godog status, its output and the events of each play.
func playFixture(t *testing.T, name string, got []Disk) (status int, output string, plays [][]Event) {
	t.Helper()

	play := func(_ context.Context, capture string, events []Event) ([]Disk, error) {
		assert.Equal(t, "fake", capture)

		plays = append(plays, events)

		return got, nil
	}

	var out strings.Builder

	ts, played := suite("fake", play, &godog.Options{
		FS: os.DirFS("testdata"), Paths: []string{name + ".feature"}, Output: &out, Format: "progress",
	})
	status = ts.Run()

	require.NotZero(t, *played, "%s: no scenario played", name)

	return status, out.String(), plays
}

// TestStepsPass checks the features that must pass, and that each scenario
// plays its own events only.
func TestStepsPass(t *testing.T) {
	status, out, plays := playFixture(t, "passes", reported)
	require.Zero(t, status, out)
	assert.Equal(t, [][]Event{{{Kind: DriveFails, Slot: "251:1"}}, nil}, plays)

	status, out, _ = playFixture(t, "known-bug-reproduced", reported)
	require.Zero(t, status, "known bug reproduced:\n%s", out)

	status, out, plays = playFixture(t, "replaced", reported)
	require.Zero(t, status, out)
	assert.Equal(t, [][]Event{
		{{Kind: DriveReplaced, Slot: "251:1", Serial: "B", WWN: "W"}},
		{{Kind: DriveReplaced, Slot: "251:1", Serial: "B", WWN: "W", CarriesData: true, Device: "/dev/sdi"}},
	}, plays)
}

// TestStepsFail checks that the harness fails when it should: otherwise a
// broken check would pass every scenario.
func TestStepsFail(t *testing.T) {
	tests := []struct {
		fixture string
		got     []Disk
		wantErr string
	}{
		{"differs", reported, "251:1 status: got Failed, want Used"},
		{"not-reported", reported, "251:2: not reported"},
		{"passes", append(reported, reported...), "251:1: reported 2 times"},
		{"known-bug-fixed", reported, "the bug looks fixed"},
		{"known-bug-changed", reported, "251:1 serial: got A, want B"},
		{"known-bug-misplaced", reported, "must follow"},
		{"undefined-step", reported, "undefined"},
		{"duplicate-slot", reported, "slot 251:1 already in the table"},
		{"wrong-header", reported, "table header must be"},
		{"no-capture", reported, "no capture"},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			status, out, _ := playFixture(t, tt.fixture, tt.got)
			require.NotZero(t, status, "want a failure, got status 0:\n%s", out)
			assert.Contains(t, out, tt.wantErr)
		})
	}
}
