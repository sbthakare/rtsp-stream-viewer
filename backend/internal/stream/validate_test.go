package stream

import "testing"

func TestValidateURL(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		allowed []string
		wantErr bool
	}{
		{"plain rtsp", "rtsp://example.com:8554/cam1", nil, false},
		{"rtsps with auth", "rtsps://user:pw@example.com/live", nil, false},
		{"empty", "", nil, true},
		{"http scheme", "http://example.com/video", nil, true},
		{"file scheme", "file:///etc/passwd", nil, true},
		{"option injection", "-i rtsp://example.com", nil, true},
		{"no host", "rtsp:///path", nil, true},
		{"host not allowed", "rtsp://evil.example.org/x", []string{"cams.example.com"}, true},
		{"host allowed", "rtsp://cams.example.com/x", []string{"cams.example.com"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ValidateURL(c.in, c.allowed, false)
			if (err != nil) != c.wantErr {
				t.Fatalf("ValidateURL(%q) error = %v, wantErr %v", c.in, err, c.wantErr)
			}
		})
	}
}

func TestFriendlyError(t *testing.T) {
	if got := friendlyError("method DESCRIBE failed: 401 Unauthorized"); got != "The camera rejected the username or password." {
		t.Fatalf("unexpected message: %q", got)
	}
	if got := friendlyError("something odd"); got == "" {
		t.Fatal("expected a fallback message")
	}
}
