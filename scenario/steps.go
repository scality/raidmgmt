package scenario

import (
	"context"
	"embed"
	"regexp"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/pkg/errors"
)

//go:embed features/*.feature
var features embed.FS //nolint:gochecknoglobals // embedded feature files

// Player returns what is reported for a captured controller once events
// happened, in order.
type Player func(ctx context.Context, capture string, events []Event) ([]Disk, error)

// Run plays, as subtests of t, the scenarios of the features tagged
// @<controller>: each "Then the drives are reported as:" step calls play with
// the capture and the events of its scenario so far. It fails when no
// scenario is tagged for the controller.
func Run(t *testing.T, controller string, play Player) {
	t.Helper()

	ts, played := suite(controller, play, &godog.Options{
		Format: "pretty", TestingT: t, FS: features, Paths: []string{"features"},
	})
	status := ts.Run()

	if status != 0 && !t.Failed() {
		t.Fatalf("godog exited with status %d", status)
	}

	if *played == 0 {
		t.Fatalf("no scenario tagged @%s", controller)
	}
}

// suite returns the godog suite of the scenarios of opts.Paths tagged
// @<controller>, and the number of scenarios it played once run. Undefined
// steps fail.
func suite(controller string, play Player, opts *godog.Options) (godog.TestSuite, *int) {
	played := new(int)

	opts.Tags = "@" + controller
	opts.Strict = true

	return godog.TestSuite{
		Name: controller,
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			(&player{play: play, played: played}).register(sc)
		},
		Options: opts,
	}, played
}

// Steps. Events read the same as a precondition ("Given drive 251:3 has
// failed") and as an action ("When drive 251:3 fails").
//
//nolint:gochecknoglobals // compiled once
var (
	capturedStep = regexp.MustCompile(`^the controller captured as "([^"]+)"$`)
	failsStep    = regexp.MustCompile(`^drive (\S+) (?:fails|has failed)$`)
	replacedStep = regexp.MustCompile(`^drive (\S+) (?:is|has been) replaced by a (used )?drive ` +
		`with serial "([^"]+)" and WWN "([^"]+)"(?:, exposed as "([^"]+)")?$`)
	volumeStep = regexp.MustCompile(
		`^a volume (?:is|has been) created on drive (\S+), exposed as "([^"]+)" with WWN "([^"]+)"$`)
	reportedStep = regexp.MustCompile(`^the drives are reported as:$`)
	knownBugStep = regexp.MustCompile(`^known bug ([A-Z][A-Z0-9]*-[0-9]+) reports instead:$`)
)

// player holds the state of one scenario.
type player struct {
	play    Player
	played  *int
	capture string
	events  []Event
	// knownBugs are the known bugs following a reported step, by step ID.
	knownBugs map[string]*KnownBug
	// step is the ID of the step being run.
	step string
}

func (p *player) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		*p.played++

		return ctx, withoutStack(p.collectKnownBugs(s))
	})
	sc.StepContext().Before(func(ctx context.Context, st *godog.Step) (context.Context, error) {
		p.step = st.Id

		return ctx, nil
	})

	sc.Given(capturedStep, func(capture string) { p.capture = capture })
	sc.Step(failsStep, func(slot string) {
		p.events = append(p.events, Event{Kind: DriveFails, Slot: slot})
	})
	sc.Step(replacedStep, func(slot, used, serial, wwn, device string) {
		p.events = append(p.events, Event{
			Kind: DriveReplaced, Slot: slot, Serial: serial, WWN: wwn,
			CarriesData: used != "", Device: device,
		})
	})
	sc.Step(volumeStep, func(slot, device, wwn string) {
		p.events = append(p.events,
			Event{Kind: VolumeCreated, Slot: slot, Device: device, VolumeWWN: wwn})
	})
	sc.Then(reportedStep, func(ctx context.Context, table *godog.Table) error {
		return withoutStack(p.drivesReported(ctx, table))
	})
	// Checked with the reported step before it.
	sc.Then(knownBugStep, func(string, *godog.Table) {})
}

// collectKnownBugs reads the known bug steps of a scenario, each one checked
// by the reported step right before it.
func (p *player) collectKnownBugs(s *godog.Scenario) error {
	p.knownBugs = map[string]*KnownBug{}

	for i, st := range s.Steps {
		m := knownBugStep.FindStringSubmatch(st.Text)
		if m == nil {
			continue
		}

		if i == 0 || !reportedStep.MatchString(s.Steps[i-1].Text) {
			return errors.Errorf("%q must follow a %q step", st.Text, "the drives are reported as:")
		}

		reported, err := ParseTable(stepRows(st))
		if err != nil {
			return errors.Wrapf(err, "known bug %s", m[1])
		}

		p.knownBugs[s.Steps[i-1].Id] = &KnownBug{Ticket: m[1], Reported: reported}
	}

	return nil
}

func (p *player) drivesReported(ctx context.Context, table *godog.Table) error {
	if p.capture == "" {
		return errors.New("no capture: start with \"Given the controller captured as\"")
	}

	expect, err := ParseTable(tableRows(table))
	if err != nil {
		return err
	}

	got, err := p.play(ctx, p.capture, p.events)
	if err != nil {
		return err
	}

	e := Expectation{Expect: expect, KnownBug: p.knownBugs[p.step]}

	notes, err := e.Check(got)
	if err != nil {
		return err
	}

	if len(notes) > 0 {
		godog.Logf(ctx, "known bug %s reproduced, differences with the expected table:\n%s",
			e.KnownBug.Ticket, strings.Join(notes, "\n"))
	}

	return nil
}

// messageError is an error without stack trace, which godog would print below
// its message.
type messageError string

func (e messageError) Error() string { return string(e) }

// withoutStack drops the stack trace of an error.
func withoutStack(err error) error {
	if err == nil {
		return nil
	}

	return messageError(err.Error())
}

func tableRows(table *godog.Table) [][]string {
	out := make([][]string, 0, len(table.Rows))

	for _, row := range table.Rows {
		cells := make([]string, 0, len(row.Cells))
		for _, cell := range row.Cells {
			cells = append(cells, cell.Value)
		}

		out = append(out, cells)
	}

	return out
}

func stepRows(st *godog.Step) [][]string {
	if st.Argument == nil || st.Argument.DataTable == nil {
		return nil
	}

	return tableRows(st.Argument.DataTable)
}
