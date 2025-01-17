package logicalvolumemanager_test

import (
	"logicalvolumemanager"
	"testing"
)

func TestSmallestPositive(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected int
	}{
		{
			name:     "empty array",
			input:    []int{},
			expected: 0,
		},
		{
			name:     "array with negative numbers",
			input:    []int{-1, -2, -3},
			expected: 0,
		},
		{
			name:     "array with positive numbers",
			input:    []int{1, 2, 3},
			expected: 0,
		},
		{
			name:     "nominal case",
			input:    []int{5, 2, 17},
			expected: 0,
		},
		{
			name:     "nominal case 2",
			input:    []int{0, 1, 3},
			expected: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := logicalvolumemanager.SmallestPositive(tc.input)
			if actual != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, actual)
			}
		})
	}
}
