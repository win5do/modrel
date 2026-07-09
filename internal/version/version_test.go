package version

import "testing"

func TestVersionValidationAndNext(t *testing.T) {
	t.Run("validate accepted versions", func(t *testing.T) {
		for _, raw := range []string{"v0.1.0", "v1.2.3", "v1.2.3-rc.1", "v10.20.30-rc.40"} {
			if err := Validate(raw); err != nil {
				t.Fatalf("Validate(%q) returned error: %v", raw, err)
			}
		}
	})

	t.Run("reject invalid versions", func(t *testing.T) {
		for _, raw := range []string{"1.2.3", "v1.2", "v1.2.3-rc1", "v01.2.3", "v1.02.3", "v1.2.03"} {
			if err := Validate(raw); err == nil {
				t.Fatalf("Validate(%q) returned nil", raw)
			}
		}
	})

	t.Run("next stable", func(t *testing.T) {
		cases := map[string]string{
			"":                 "v0.1.0",
			"v1.2.3":           "v1.2.4",
			"v1.2.4-rc.2":      "v1.2.4",
			"boot/v1.2.3":      "v1.2.4",
			"boot/v1.2.4-rc.2": "v1.2.4",
		}
		for input, want := range cases {
			got, err := NextStable(TrimTagPrefix(input))
			if err != nil {
				t.Fatalf("NextStable(%q) returned error: %v", input, err)
			}
			if got != want {
				t.Fatalf("NextStable(%q) = %q, want %q", input, got, want)
			}
		}
	})

	t.Run("next rc", func(t *testing.T) {
		cases := map[string]string{
			"":            "v0.1.0-rc.1",
			"v1.2.3":      "v1.2.4-rc.1",
			"v1.2.4-rc.2": "v1.2.4-rc.3",
		}
		for input, want := range cases {
			got, err := NextRC(input)
			if err != nil {
				t.Fatalf("NextRC(%q) returned error: %v", input, err)
			}
			if got != want {
				t.Fatalf("NextRC(%q) = %q, want %q", input, got, want)
			}
		}
	})

	t.Run("compare treats stable newer than rc with same base", func(t *testing.T) {
		if got := Compare("v1.2.3", "v1.2.3-rc.9"); got <= 0 {
			t.Fatalf("Compare stable vs rc = %d, want positive", got)
		}
		if got := Compare("v1.2.4-rc.1", "v1.2.3"); got <= 0 {
			t.Fatalf("Compare next rc base vs stable = %d, want positive", got)
		}
	})
}
