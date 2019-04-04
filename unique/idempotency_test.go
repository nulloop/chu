package unique_test

import (
	"testing"

	"github.com/nulloop/chu/unique"
)

func TestIdempotency(t *testing.T) {
	testCases := []struct {
		size         int
		initialItems []string
		tests        []string
		expected     []bool
	}{
		{
			size:         4,
			initialItems: []string{"1", "2", "3"},
			tests:        []string{"1", "2", "3", "4", "4"},
			expected:     []bool{false, false, false, true, false},
		},
		{
			size:         4,
			initialItems: []string{},
			tests:        []string{"1", "2", "3", "4", "4", "1"},
			expected:     []bool{true, true, true, true, false, true},
		},
		{
			size:         4,
			initialItems: []string{},
			tests:        []string{"1", "2", "3", "1", "2", "3", "4"},
			expected:     []bool{true, true, true, false, false, false, true},
		},
	}

	for _, testCase := range testCases {
		idempotency := unique.New(testCase.size)
		for _, item := range testCase.initialItems {
			idempotency.IsUnique(item)
		}

		for i, item := range testCase.tests {
			result := idempotency.IsUnique(item)
			if result != testCase.expected[i] {
				t.Fatalf("expected %s to be %+v but got %+v", item, testCase.expected[i], result)
			}
		}
	}
}
