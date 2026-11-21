package install

import (
	"testing"
)

func TestParseTargetKnowsTheSupportedClients(t *testing.T) {
	for raw, want := range map[string]Target{
		"vscode":    TargetVSCodeUser,
		" VSCODE ":  TargetVSCodeUser,
		"workspace": TargetVSCodeWorkspace,
		"claude":    TargetClaudeDesktop,
	} {
		target, err := ParseTarget(raw)
		if err != nil {
			t.Fatalf("ParseTarget(%q) returned error: %v", raw, err)
		}
		if target != want {
			t.Errorf("ParseTarget(%q) = %q, want %q", raw, target, want)
		}
	}
}

func TestParseTargetRejectsAnythingElse(t *testing.T) {
	for _, raw := range []string{"", "cursor", "vscod"} {
		if _, err := ParseTarget(raw); err == nil {
			t.Errorf("ParseTarget(%q) accepted an unknown client", raw)
		}
	}
}
