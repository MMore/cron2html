package main

import "testing"

// TODO: creationTime

func TestTemplateVersion(t *testing.T) {
	v := templateVersion()
	if v != "v"+VERSION {
		t.Errorf("wrong version, got: %s", v)
	}
}
