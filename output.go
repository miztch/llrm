package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rodaine/table"
	"gopkg.in/yaml.v3"

	"github.com/miztch/llrm/internal/cleaner"
)

var (
	styleError = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	styleWarn  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
)

// Output holds I/O streams and output format for the CLI.
type Output struct {
	Format string
	Out    io.Writer
	Err    io.Writer
	In     io.Reader
}

func newOutput(format string) *Output {
	return &Output{Format: format, Out: os.Stdout, Err: os.Stderr, In: os.Stdin}
}

func (o *Output) Candidates(candidates []cleaner.Candidate) error {
	if o.Format != "table" {
		return o.printFormatted(candidates)
	}
	tbl := table.New("Layer Name", "Version", "Runtimes", "Architectures", "Created").
		WithHeaderSeparatorRow('-').
		WithWriter(o.Out)
	for _, c := range candidates {
		tbl.AddRow(c.LayerName, c.Version, strings.Join(c.Runtimes, ", "), strings.Join(c.Architectures, ", "), c.CreatedDate)
	}
	tbl.Print()
	return nil
}

func (o *Output) AllVersions(versions []cleaner.VersionStatus) error {
	if o.Format != "table" {
		return o.printFormatted(versions)
	}
	tbl := table.New("Layer Name", "Version", "Attached", "Runtimes", "Architectures", "Created").
		WithHeaderSeparatorRow('-').
		WithWriter(o.Out)
	for _, v := range versions {
		tbl.AddRow(v.LayerName, v.Version, v.Attached, strings.Join(v.Runtimes, ", "), strings.Join(v.Architectures, ", "), v.CreatedDate)
	}
	tbl.Print()
	return nil
}

func (o *Output) NoCandidates() error {
	if o.Format != "table" {
		return o.Candidates([]cleaner.Candidate{})
	}
	fmt.Fprintln(o.Out, "No candidates found. Your layers look tidy!")
	return nil
}

func (o *Output) NoVersions() error {
	if o.Format != "table" {
		return o.AllVersions([]cleaner.VersionStatus{})
	}
	fmt.Fprintln(o.Out, "No layer versions found.")
	return nil
}

// Confirm prints a prompt and reads a y/N answer from In.
func (o *Output) Confirm() bool {
	fmt.Fprint(o.Out, "Do you want to proceed? [y/N] ")
	scanner := bufio.NewScanner(o.In)
	scanner.Scan()
	fmt.Fprintln(o.Out)
	return strings.ToLower(strings.TrimSpace(scanner.Text())) == "y"
}

func (o *Output) printFormatted(v any) error {
	payload := struct {
		Layers any `json:"layers" yaml:"layers"`
	}{Layers: v}

	switch o.Format {
	case "yaml":
		return yaml.NewEncoder(o.Out).Encode(payload)
	default:
		enc := json.NewEncoder(o.Out)
		enc.SetIndent("", "    ")
		return enc.Encode(payload)
	}
}
