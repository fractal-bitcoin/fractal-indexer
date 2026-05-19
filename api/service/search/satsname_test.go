package search_test

import (
	"fractal-indexer/api/constant"
	"strings"
	"testing"
)

func validateSnsInscriptionName(name string) (nameFinal, suffix string, ok bool) {
	if strings.HasPrefix(name, " ") {
		return "", "", false
	}
	if strings.HasPrefix(name, "\n") {
		return "", "", false
	}

	names := strings.Fields(name)
	if len(names) > 0 {
		nameFinal = names[0]
	} else {
		return "", "", false
	}
	parts := strings.Split(nameFinal, ".")
	if len(parts) != 2 {
		return "", "", false
	}
	suffix = strings.ToLower(parts[1])
	if suffix == constant.FB_NAME_SUFFIX {

		return strings.ToLower(nameFinal), suffix, true
	}
	return "", "", false
}

func TestSatsNames(t *testing.T) {
	testCases := []struct {
		input string
		want  string
		err   bool
	}{
		{"bob.fb", "bob.fb", false},
		{"Bob.fb", "bob.fb", false},
		{"bob.FB", "bob.fb", false},
		{"bob.fb ", "bob.fb", false},
		{"bob.fb\n", "bob.fb", false},
		{" bob.fb", "bob.fb", true},
		{"\nbob.fb", "bob.fb", true},
		{"bo b.fb", "bob.fb", true},
		{"bo.b.fb", "bob.fb", true},
		{"😄bob.fb", "😄bob.fb", false},
		{"bob.sats", "bob.sats", true},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got, _, ok := validateSnsInscriptionName(tc.input)
			if (!ok) != tc.err {
				t.Fatalf("unexpected error: %v", ok)
			}
			if ok && got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
