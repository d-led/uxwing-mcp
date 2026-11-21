package uxwing

import "testing"

func TestParseFormatAcceptsTheFormatsUXWingServes(t *testing.T) {
	for _, test := range []struct{ raw, want string }{
		{"svg", "svg"},
		{"SVG", "svg"},
		{"png", "png"},
		{" Png ", "png"},
	} {
		got, err := ParseFormat(test.raw)
		if err != nil {
			t.Fatalf("ParseFormat(%q) returned error: %v", test.raw, err)
		}
		if got.String() != test.want {
			t.Errorf("ParseFormat(%q) = %q, want %q", test.raw, got, test.want)
		}
	}
}

func TestParseFormatRejectsFormatsUXWingDoesNotServe(t *testing.T) {
	for _, raw := range []string{"", "   ", "gif", "jpg", "ico", "eps", "pdf"} {
		if _, err := ParseFormat(raw); err == nil {
			t.Errorf("ParseFormat(%q) accepted a format UXWing does not serve", raw)
		}
	}
}

func TestParseSlugAcceptsPageAddresses(t *testing.T) {
	for _, raw := range []string{
		"robot-icon",
		"  robot-icon  ",
		"/robot-icon/",
		"https://uxwing.com/robot-icon/",
		"https://uxwing.com/robot-icon",
		"https://uxwing.com/robot-icon/?ref=search",
	} {
		slug, err := ParseSlug(raw)
		if err != nil {
			t.Fatalf("ParseSlug(%q) returned error: %v", raw, err)
		}
		if slug.String() != "robot-icon" {
			t.Errorf("ParseSlug(%q) = %q, want robot-icon", raw, slug)
		}
	}
}

func TestParseSlugRejectsAnythingThatIsNotAnIconSlug(t *testing.T) {
	for _, raw := range []string{
		"",
		"   ",
		"Robot Icon",
		"robot_icon",
		"../etc/passwd",
		"https://example.com/robot-icon/",
		"https://uxwing.com/",
	} {
		if _, err := ParseSlug(raw); err == nil {
			t.Errorf("ParseSlug(%q) accepted a value that is not an icon slug", raw)
		}
	}
}
