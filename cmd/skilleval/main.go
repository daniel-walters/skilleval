package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daniel-walters/skilleval/eval"
	"github.com/daniel-walters/skilleval/result"
	"github.com/daniel-walters/skilleval/runner"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "run":
		if err := runCmd(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "skilleval: %v\n", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "skilleval: unknown command %q\n", os.Args[1])
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage:
  skilleval run <eval.yaml> [--model ID] [--out result.json]

`)
}

func runCmd(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	model := fs.String("model", "", "model id for the Cursor agent")
	out := fs.String("out", "result.json", "path to write Result JSON")

	flagArgs, positional, err := splitFlagsAndPositionals(args)
	if err != nil {
		return err
	}
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if len(positional) != 1 {
		return fmt.Errorf("run: expected <eval.yaml>")
	}
	evalPath := positional[0]
	if !filepath.IsAbs(evalPath) {
		abs, err := filepath.Abs(evalPath)
		if err != nil {
			return err
		}
		evalPath = abs
	}

	ev, err := eval.Load(evalPath)
	if err != nil {
		return err
	}
	if *model == "" {
		return fmt.Errorf("run: --model is required")
	}

	hb := startHeartbeat(os.Stderr, stderrIsTerminal())
	defer hb.Stop()

	r, workspace, err := runner.Run(context.Background(), ev, evalPath, runner.Options{
		Model:   *model,
		Attempt: 1,
	})
	if err != nil {
		return err
	}

	outPath := *out
	if !filepath.IsAbs(outPath) {
		abs, err := filepath.Abs(outPath)
		if err != nil {
			return err
		}
		outPath = abs
	}
	if err := result.Write(outPath, r); err != nil {
		return err
	}
	fmt.Printf("wrote %s (status=%s workspace=%s)\n", outPath, r.Status, workspace)
	return nil
}

// splitFlagsAndPositionals allows flags before or after the eval path.
// The stdlib flag package stops at the first non-flag argument.
func splitFlagsAndPositionals(args []string) (flagArgs, positional []string, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			positional = append(positional, args[i+1:]...)
			return flagArgs, positional, nil
		case a == "-h", a == "--help":
			flagArgs = append(flagArgs, a)
		case strings.HasPrefix(a, "-"):
			flagArgs = append(flagArgs, a)
			// Keep "--flag=value" as one token; otherwise take the next arg as value
			// when this looks like a boolean-less long/short flag without '='.
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		default:
			positional = append(positional, a)
		}
	}
	return flagArgs, positional, nil
}
