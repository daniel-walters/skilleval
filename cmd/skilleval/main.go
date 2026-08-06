package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/daniel-walters/skilleval/checker"
	"github.com/daniel-walters/skilleval/eval"
	"github.com/daniel-walters/skilleval/result"
	"github.com/daniel-walters/skilleval/runner"
	"github.com/daniel-walters/skilleval/summary"
)

func main() {
	if err := loadDotEnv(".env"); err != nil {
		fmt.Fprintf(os.Stderr, "skilleval: %v\n", err)
		os.Exit(1)
	}
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

Credentials:
  CURSOR_API_KEY from the process environment, or a .env file in the
  current directory (process environment wins if both are set).

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

	outPath := *out
	if !filepath.IsAbs(outPath) {
		abs, err := filepath.Abs(outPath)
		if err != nil {
			return err
		}
		outPath = abs
	}

	if ev.Attempts <= 1 {
		return runSingle(ev, evalPath, *model, outPath)
	}
	return runMulti(ev, evalPath, *model, outPath)
}

func runSingle(ev *eval.Eval, evalPath, model, outPath string) error {
	hb := startHeartbeat(os.Stderr, stderrIsTerminal())
	defer hb.Stop()

	r, workspace, err := runner.Run(context.Background(), ev, evalPath, runner.Options{
		Model:   model,
		Attempt: 1,
	})
	if err != nil {
		return err
	}
	if err := result.Write(outPath, r); err != nil {
		return err
	}
	fmt.Printf("wrote %s (status=%s workspace=%s)\n", outPath, r.Status, workspace)
	return reportVerdict(os.Stdout, checker.Check(r, ev.Expects, workspace))
}

func runMulti(ev *eval.Eval, evalPath, model, outPath string) error {
	n := ev.Attempts
	attempts := make([]summary.Attempt, 0, n)

	for i := 1; i <= n; i++ {
		hb := startHeartbeat(os.Stderr, stderrIsTerminal())
		r, workspace, err := runner.Run(context.Background(), ev, evalPath, runner.Options{
			Model:         model,
			Attempt:       i,
			TotalAttempts: n,
		})
		hb.Stop()
		if err != nil {
			fmt.Printf("attempt %d/%d: error: %v\n", i, n, err)
			attempts = append(attempts, summary.Attempt{Err: err})
			continue
		}

		attemptPath := attemptOutPath(outPath, i, n)
		if err := result.Write(attemptPath, r); err != nil {
			return err
		}
		fmt.Printf("attempt %d/%d: wrote %s (status=%s workspace=%s)\n", i, n, attemptPath, r.Status, workspace)

		v := checker.Check(r, ev.Expects, workspace)
		if err := printVerdict(os.Stdout, v); err != nil {
			return err
		}
		attempts = append(attempts, summary.Attempt{Result: r, Verdict: v})
	}

	rep := summary.Aggregate(attempts)
	sumPath := summaryOutPath(outPath)
	if err := writeSummary(sumPath, rep); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", sumPath)
	if err := printSummary(os.Stdout, rep); err != nil {
		return err
	}
	if ev.PassRate != nil && ev.PassRate.Min != nil && !summary.MeetsPassRate(rep, *ev.PassRate.Min) {
		return fmt.Errorf("pass rate %g below min %g", rep.PassRate, *ev.PassRate.Min)
	}
	return nil
}

// attemptOutPath returns out unchanged for a single attempt, otherwise stem-N.ext.
func attemptOutPath(out string, attempt, total int) string {
	if total <= 1 {
		return out
	}
	ext := filepath.Ext(out)
	stem := strings.TrimSuffix(out, ext)
	return fmt.Sprintf("%s-%d%s", stem, attempt, ext)
}

// summaryOutPath returns stem-summary.ext beside the --out path.
func summaryOutPath(out string) string {
	ext := filepath.Ext(out)
	stem := strings.TrimSuffix(out, ext)
	return stem + "-summary" + ext
}

func writeSummary(path string, rep summary.Report) error {
	raw, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return fmt.Errorf("encode summary: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write summary %s: %w", path, err)
	}
	return nil
}

func printSummary(w io.Writer, rep summary.Report) error {
	if _, err := fmt.Fprintln(w, "---"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "passRate: %g (%d/%d)\n", rep.PassRate, rep.Passed, rep.Attempts); err != nil {
		return err
	}
	if rep.AvgTurns != nil {
		if _, err := fmt.Fprintf(w, "avgTurns: %g\n", *rep.AvgTurns); err != nil {
			return err
		}
	}
	if rep.AvgCostUSD != nil {
		if _, err := fmt.Fprintf(w, "avgCostUSD: %g\n", *rep.AvgCostUSD); err != nil {
			return err
		}
	}
	if rep.AvgDurationMs != nil {
		if _, err := fmt.Fprintf(w, "avgDurationMs: %g\n", *rep.AvgDurationMs); err != nil {
			return err
		}
	}
	return nil
}

// printVerdict prints PASS/FAIL (and failure lines) without affecting exit status.
func printVerdict(w io.Writer, v checker.Verdict) error {
	if v.Passed {
		_, err := fmt.Fprintln(w, "PASS")
		return err
	}
	if _, err := fmt.Fprintln(w, "FAIL"); err != nil {
		return err
	}
	for _, f := range v.Failures {
		if _, err := fmt.Fprintf(w, "  %s: %s\n", f.Path, f.Reason); err != nil {
			return err
		}
	}
	return nil
}

// reportVerdict prints PASS/FAIL and returns an error when the check failed.
func reportVerdict(w io.Writer, v checker.Verdict) error {
	if err := printVerdict(w, v); err != nil {
		return err
	}
	if !v.Passed {
		return fmt.Errorf("check failed")
	}
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
