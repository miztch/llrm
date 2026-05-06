package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/alecthomas/kong"

	"github.com/miztch/llrm/internal/aws"
	"github.com/miztch/llrm/internal/cleaner"
)

var version = func() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "dev"
}()

var cli struct {
	Region       string             `help:"AWS region (defaults to AWS_DEFAULT_REGION / config file)"`
	Name         string             `help:"Exact layer name to target"`
	Filter       string             `help:"Filter layers by name (substring match)"`
	KeepVersions int                `name:"keep-versions" help:"Keep the N most recent versions per layer; delete older ones"`
	List         bool               `help:"Print candidates without deleting"`
	ListAll      bool               `name:"list-all" help:"Print all layer versions including attached ones"`
	Yes          bool               `help:"Skip confirmation prompt"`
	Output       string             `help:"Output format for --list / --list-all" default:"table" enum:"table,json,yaml"`
	Version      kong.VersionFlag   `help:"Print version and quit" short:"v"`
}

func runDelete(ctx context.Context, c *cleaner.Cleaner, o *Output) error {
	candidates, err := c.Scan(ctx)
	if err != nil {
		return err
	}

	if len(candidates) == 0 {
		return o.NoCandidates()
	}

	if err := o.Candidates(candidates); err != nil {
		return err
	}

	fmt.Fprintf(o.Out, "\n%s\n", styleWarn.Render(fmt.Sprintf("About to delete %d layer version(s) and cannot be undone.", len(candidates))))

	if !cli.Yes && !o.Confirm() {
		fmt.Fprintln(o.Out, "Aborted.")
		return nil
	}

	fmt.Fprintln(o.Out, "deleting...")
	failed := 0
	c.Delete(ctx, candidates, func(t cleaner.Candidate, err error) {
		if err != nil {
			fmt.Fprintln(o.Err, styleError.Render(err.Error()))
			failed++
		} else {
			fmt.Fprintf(o.Out, "deleted: %s:%d\n", t.LayerName, t.Version)
		}
	})
	if failed == 0 {
		fmt.Fprintln(o.Out, "\nDone!")
	}
	return nil
}

func runList(ctx context.Context, c *cleaner.Cleaner, o *Output) error {
	if cli.ListAll {
		versions, err := c.ListAll(ctx)
		if err != nil {
			return err
		}
		if len(versions) == 0 {
			return o.NoVersions()
		}
		return o.AllVersions(versions)
	}

	candidates, err := c.Scan(ctx)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return o.NoCandidates()
	}
	return o.Candidates(candidates)
}

func run() error {
	ctx := context.Background()

	client, err := aws.NewClient(ctx, cli.Region)
	if err != nil {
		return err
	}

	opts := cleaner.Options{
		Name:         cli.Name,
		Filter:       cli.Filter,
		KeepVersions: cli.KeepVersions,
	}
	c := cleaner.New(client, opts)

	if cli.ListAll || cli.List {
		return runList(ctx, c, newOutput(cli.Output))
	}
	return runDelete(ctx, c, newOutput("table"))
}

func main() {
	kong.Parse(&cli,
		kong.Name("llrm"),
		kong.Description("Find and delete AWS Lambda Layer versions not attached to any function."),
		kong.Vars{"version": version},
	)
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, styleError.Render("Error: "+err.Error()))
		os.Exit(1)
	}
}
