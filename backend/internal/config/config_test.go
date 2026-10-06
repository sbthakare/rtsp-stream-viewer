package config

import "testing"

func TestEnvMPEG1FPS(t *testing.T) {
	t.Setenv("FPS", "15")
	if got := envMPEG1FPS("FPS", 25); got != 25 {
		t.Fatalf("envMPEG1FPS() = %d, want fallback 25", got)
	}

	t.Setenv("FPS", "30")
	if got := envMPEG1FPS("FPS", 25); got != 30 {
		t.Fatalf("envMPEG1FPS() = %d, want 30", got)
	}
}
