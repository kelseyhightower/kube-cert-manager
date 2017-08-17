package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"
)

var (
	usageTemplate = `acme is a client tool for managing certificates
with ACME-compliant servers.

Usage:
	acme command [arguments]

The commands are:
{{range .}}{{if .Runnable}}
	{{.Name | printf "%-11s"}} {{.Short}}{{end}}{{end}}

Use "acme help [command]" for more information about a command.

Additional help topics:
{{range .}}{{if not .Runnable}}
	{{.Name | printf "%-11s"}} {{.Short}}{{end}}{{end}}

Use "acme help [topic]" for more information about that topic.

`
	helpAccount = &command{
		UsageLine: "account",
		Short:     "account and configuration",
		Long: `
The program keeps all configuration, including issued certificates and
the corresponding keys, in a single directory which is tied to a specific account
identified by a private key.

The account metadata are stored in {{.AccountFile}} file, while the account
private key is kept in {{.AccountKey}} file.

Default location of the account config dir is
{{.ConfigDir}}.

Use -c argument with any acme command to override the default location
of the config dir. Alternatively, set ACME_CONFIG environment variable.
		`,
	}

	helpDisco = &command{
		UsageLine: "disco",
		Short:     "discovery and directory",
		Long: `
The program uses CA directory endpoint as defined by the ACME spec,
to discover endpoints such as account registration.

A directory alias can also be used. Currently defined aliases are:
{{range $alias, $url := .DiscoAliases}}
	{{$alias}}: {{$url}}{{end}}

For more information about the spec see
https://tools.ietf.org/html/draft-ietf-acme-acme.
		`,
	}
)

// An errWriter wraps a writer, recording whether a write error occurred.
type errWriter struct {
	w   io.Writer
	err error
}

func (w *errWriter) Write(b []byte) (int, error) {
	n, err := w.w.Write(b)
	if err != nil {
		w.err = err
	}
	return n, err
}

// tmpl executes the given template text on data, writing the result to w.
func tmpl(w io.Writer, text string, data interface{}) {
	t := template.New("top")
	t.Funcs(template.FuncMap{
		"trim":       strings.TrimSpace,
		"capitalize": capitalize,
	})
	template.Must(t.Parse(text))
	ew := &errWriter{w: w}
	err := t.Execute(ew, data)
	if ew.err != nil {
		// I/O error writing; ignore write on closed pipe
		if strings.Contains(ew.err.Error(), "pipe") {
			os.Exit(1)
		}
		fatalf("writing output: %v", ew.err)
	}
	if err != nil {
		panic(err)
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToTitle(r)) + s[n:]
}

// usage prints acme usage to stderr and exits with code 2.
func usage() {
	printUsage(os.Stderr)
	os.Exit(2)
}

// printUsage prints usageTemplate to w.
func printUsage(w io.Writer) {
	bw := bufio.NewWriter(w)
	tmpl(bw, usageTemplate, commands)
	bw.Flush()
}

func help(args []string) {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return
	}
	if len(args) != 1 {
		fatalf("usage: acme help command\n\nToo many arguments given.\n")
	}

	arg := args[0]
	for _, cmd := range commands {
		if cmd.Name() == arg {
			if cmd.Runnable() {
				fmt.Fprintf(os.Stdout, "usage: acme %s\n", cmd.UsageLine)
			}
			data := struct {
				ConfigDir    string
				AccountFile  string
				AccountKey   string
				DefaultDisco string
				DiscoAliases map[string]string
			}{
				ConfigDir:    configDir,
				AccountFile:  accountFile,
				AccountKey:   accountKey,
				DefaultDisco: defaultDisco,
				DiscoAliases: discoAliases,
			}
			tmpl(os.Stdout, cmd.Long, data)
			return
		}
	}

	fatalf("Unknown help topic %q. Run 'acme help'.\n", arg)
}
