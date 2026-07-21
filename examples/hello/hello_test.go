package hello

import "testing"

func TestGreeting(t *testing.T) {
	if got, want := Greeting("modrel"), "Hello, modrel!"; got != want {
		t.Fatalf("Greeting() = %q, want %q", got, want)
	}
}
