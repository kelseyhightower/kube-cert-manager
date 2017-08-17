package main

import (
	"flag"
	"testing"
)

func TestDefaultDisco(t *testing.T) {
	v := discoAliases[defaultDisco]
	if v == "" {
		t.Fatalf("alias for %q is zero; all aliases: %v", defaultDisco, discoAliases)
	}
}

func TestDiscoAliasFlag(t *testing.T) {
	tests := []struct {
		a    discoAliasFlag
		args []string
		want string
	}{
		{defaultDisco, []string{"-d", "letsencrypt-staging"}, discoAliases["letsencrypt-staging"]},
		{defaultDisco, []string{"-d", "https://disco"}, "https://disco"},
	}
	for i, test := range tests {
		var a = test.a
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.Var(&a, "d", "")
		if err := fs.Parse(test.args); err != nil {
			t.Errorf("%d: parse(%v): %v", i, test.args, err)
			continue
		}
		if a.String() != test.want {
			t.Errorf("%d: a = %q; want %q", i, a, test.want)
		}
	}
}
