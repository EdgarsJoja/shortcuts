package template

import (
	"os/exec"
	"testing"
)

func TestExpandShell(t *testing.T) {
	tests := []struct {
		name string
		tmpl string
		args []string
		want string
	}{
		{"no placeholder", "echo hi", nil, "echo hi"},
		{"args many", "echo {{args}}", []string{"a", "b c"}, "echo 'a' 'b c'"},
		{"args none", "echo {{args}}", nil, "echo "},
		{"positional present", "echo {{1}} {{2}}", []string{"a", "b"}, "echo 'a' 'b'"},
		{"positional missing", "echo {{1}} {{2}}", []string{"a"}, "echo 'a' ''"},
		{"embedded positional", "echo --msg={{1}}", []string{"x y"}, "echo --msg='x y'"},
		{"adjacent", "echo {{1}}{{2}}", []string{"a", "b"}, "echo 'a''b'"},
		{"repeated", "echo {{1}} {{1}}", []string{"a"}, "echo 'a' 'a'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts, err := Parse(tt.tmpl, true)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := ExpandShell(parts, tt.args); got != tt.want {
				t.Fatalf("ExpandShell = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandDirectElement(t *testing.T) {
	tests := []struct {
		name string
		tmpl string
		args []string
		want []string
	}{
		{"literal", "git", nil, []string{"git"}},
		{"empty element", "", nil, []string{""}},
		{"positional", "--msg={{1}}", []string{"a b"}, []string{"--msg=a b"}},
		{"missing positional", "{{1}}", nil, []string{""}},
		{"args standalone", "{{args}}", []string{"a", "b c"}, []string{"a", "b c"}},
		{"args none", "{{args}}", nil, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts, err := Parse(tt.tmpl, false)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			got := ExpandDirectElement(parts, tt.args)
			if len(got) != len(tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %#v, want %#v", got, tt.want)
				}
			}
		})
	}
}

func TestParseRejects(t *testing.T) {
	tests := []struct {
		name  string
		tmpl  string
		shell bool
	}{
		{"quoted placeholder", `echo "{{1}}"`, true},
		{"single quoted placeholder", `echo '{{1}}'`, true},
		{"embedded args", `echo x{{args}}y`, true},
		{"direct embedded args", `--x={{args}}`, false},
		{"direct args plus literal", `{{args}} --x`, false},
		{"unmatched", `echo {{1`, true},
		{"zero", `echo {{0}}`, true},
		{"unknown", `echo {{foo}}`, true},
		{"empty", `echo {{}}`, true},
		{"negative", `echo {{-1}}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Parse(tt.tmpl, tt.shell); err == nil {
				t.Fatalf("Parse(%q) succeeded, want error", tt.tmpl)
			}
		})
	}
}

func TestEscapedPlaceholderIsLiteral(t *testing.T) {
	parts, err := Parse(`echo \{{1}}`, true)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if HasPlaceholder(parts) {
		t.Fatalf("escaped placeholder detected as placeholder: %#v", parts)
	}
}

func TestShellQuoteRoundTrip(t *testing.T) {
	args := []string{
		"",
		"plain",
		"two words",
		"it's",
		"$(touch /tmp/should-not-exist)",
		"`id`",
		"a\nb",
		"\\",
		"*.go",
		"-n",
		"héllo",
		"'; echo pwned; '",
	}
	for _, arg := range args {
		script := "printf '%s' " + ShellQuote(arg)
		out, err := exec.Command("/bin/sh", "-c", script).Output()
		if err != nil {
			t.Fatalf("sh failed for %q: %v", arg, err)
		}
		if string(out) != arg {
			t.Fatalf("round trip %q -> %q", arg, string(out))
		}
	}
}
