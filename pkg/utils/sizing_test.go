package utils_test

import (
	"testing"

	"github.com/scality/raidmgmt/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These RAID CLIs report binary quantities. ssacli and megaraid use decimal-style
// labels (GB/TB) for binary values: ssacli "800 GB" is 858993459200 bytes (= 800
// GiB, confirmed against the lsblk byte count) and megaraid "16.370 TB" is
// 17999005346693 bytes (= 16.370 TiB). storcli2 uses proper IEC labels (TiB/KiB).
// So every unit, decimal-labelled or IEC, resolves to a 1024-based multiplier.
const (
	kib uint64 = 1 << 10
	mib uint64 = 1 << 20
	gib uint64 = 1 << 30
	tib uint64 = 1 << 40
	pib uint64 = 1 << 50
)

// bytesOf mirrors the conversion done by ConvertSizeBytes (including float
// truncation) at runtime, avoiding constant float-to-integer conversion errors.
func bytesOf(value float64, unit uint64) uint64 {
	return uint64(value * float64(unit))
}

func TestConvertSizeBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    uint64
		wantErr bool
	}{
		// Ground truth from existing adapters: decimal labels carry binary values.
		{name: "ssacli 800 GB is 800 GiB", input: "800 GB", want: 858993459200},
		{name: "megaraid 16.370 TB is 16.370 TiB", input: "16.370 TB", want: 17999005346693},

		// storcli2 IEC labels.
		{name: "storcli2 drive 9.094 TiB", input: "9.094 TiB", want: bytesOf(9.094, tib)},
		{name: "storcli2 strip 256 KiB", input: "256 KiB", want: 256 * kib},
		{name: "storcli2 sector 4 KiB", input: "4 KiB", want: 4 * kib},

		// One case per decimal-labelled unit (binary value).
		{name: "KB", input: "1 KB", want: kib},
		{name: "MB", input: "1 MB", want: mib},
		{name: "GB", input: "1 GB", want: gib},
		{name: "TB", input: "1 TB", want: tib},
		{name: "PB", input: "1 PB", want: pib},

		// One case per IEC unit.
		{name: "KiB", input: "1 KiB", want: kib},
		{name: "MiB", input: "1 MiB", want: mib},
		{name: "GiB", input: "1 GiB", want: gib},
		{name: "TiB", input: "1 TiB", want: tib},
		{name: "PiB", input: "1 PiB", want: pib},

		// Fractional value with comma decimal separator.
		{name: "fractional comma GiB", input: "1,5 GiB", want: bytesOf(1.5, gib)},

		// Malformed input.
		{name: "missing unit", input: "12", wantErr: true},
		{name: "non numeric value", input: "abc GB", wantErr: true},
		{name: "unknown decimal unit", input: "12 ZB", wantErr: true},
		{name: "unknown binary unit", input: "12 ZiB", wantErr: true},
		{name: "empty string", input: "", wantErr: true},
		{name: "too many parts", input: "1 2 GB", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := utils.ConvertSizeBytes(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Zero(t, got)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestConvertSizeBytesDecimalLabelsAreBinary documents an intentional, domain-specific
// choice: a decimal label and its IEC sibling resolve to the SAME byte count, because
// the RAID CLIs report binary quantities regardless of the label they print.
func TestConvertSizeBytesDecimalLabelsAreBinary(t *testing.T) {
	t.Parallel()

	pairs := [][2]string{
		{"1 KB", "1 KiB"},
		{"1 MB", "1 MiB"},
		{"1 GB", "1 GiB"},
		{"1 TB", "1 TiB"},
		{"1 PB", "1 PiB"},
	}

	for _, pair := range pairs {
		decimalLabel, iecLabel := pair[0], pair[1]

		decimalValue, err := utils.ConvertSizeBytes(decimalLabel)
		require.NoError(t, err)

		iecValue, err := utils.ConvertSizeBytes(iecLabel)
		require.NoError(t, err)

		assert.Equal(t, decimalValue, iecValue,
			"%q and %q must resolve to the same binary value", decimalLabel, iecLabel)
	}
}
