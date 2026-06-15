package cmd

import (
	"strings"
	"testing"
)

func TestRootCommandVersionIsSet(t *testing.T) {
	if rootCmd.Version == "" {
		t.Fatal("rootCmd.Version must be non-empty (set at build via -ldflags, default \"dev\" in dev builds)")
	}
}

func TestRootCommandVersionTemplateIncludesVersion(t *testing.T) {
	output := new(strings.Builder)
	rootCmd.SetOut(output)
	rootCmd.SetErr(output)
	rootCmd.SetVersionTemplate("{{.Use}} version {{.Version}}\n")

	tpl := rootCmd.VersionTemplate()
	if !strings.Contains(tpl, "{{.Version}}") {
		t.Errorf("version template should reference {{.Version}}, got: %q", tpl)
	}
	if !strings.Contains(tpl, "{{.Use}}") {
		t.Errorf("version template should reference {{.Use}}, got: %q", tpl)
	}
}
