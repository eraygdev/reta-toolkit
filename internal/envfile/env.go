package envfile

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/eraygdev/reta-toolkit/internal/ui"
)

func printEnvHelp() {
	c := ui.Colors()
	fmt.Printf("%sreta env%s — inspect .env against .env.example\n", c.Bold, c.Reset)
	fmt.Println("")
	fmt.Println("Usage: reta env <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Printf("  %scheck%s    List keys missing or empty in .env\n", c.Cyan, c.Reset)
	fmt.Printf("  %slist%s     List keys in .env with masked values\n", c.Cyan, c.Reset)
	fmt.Printf("           Use %s--show%s to reveal values\n", c.Bold, c.Reset)
	fmt.Printf("  %sdiff%s     Compare keys between .env and .env.example\n", c.Cyan, c.Reset)
	fmt.Printf("  %ssync%s     Append missing keys from .env.example into .env\n", c.Cyan, c.Reset)
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Printf("  %s--env%s      Env file (default: .env)\n", c.Bold, c.Reset)
	fmt.Printf("  %s--example%s  Example file (default: .env.example)\n", c.Bold, c.Reset)
}

func dieMissing(path string, hints []string) {
	c := ui.Colors()
	fmt.Fprintf(os.Stderr, "%sError:%s %s not found.\n", c.Red, c.Reset, path)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Next steps:")
	for _, h := range hints {
		fmt.Fprintf(os.Stderr, "  %s%s%s\n", c.Bold, h, c.Reset)
	}
}

func cmdCheck(example, env parsedEnv) int {
	c := ui.Colors()
	var missing, empty []string
	for _, k := range example.Order {
		v, ok := env.Values[k]
		if !ok {
			missing = append(missing, k)
		} else if v == "" {
			empty = append(empty, k)
		}
	}

	if len(missing) == 0 && len(empty) == 0 {
		fmt.Printf("%sOK%s .env is in sync with .env.example (%d keys)\n", c.Green, c.Reset, len(example.Order))
		return 0
	}

	if len(missing) > 0 {
		fmt.Printf("%sMissing%s in .env (%d):\n", c.Red, c.Reset, len(missing))
		for _, k := range missing {
			fmt.Printf("  %s%s%s\n", c.Cyan, k, c.Reset)
		}
	}
	if len(empty) > 0 {
		if len(missing) > 0 {
			fmt.Println("")
		}
		fmt.Printf("%sEmpty%s values in .env (%d):\n", c.Yellow, c.Reset, len(empty))
		for _, k := range empty {
			fmt.Printf("  %s%s%s\n", c.Cyan, k, c.Reset)
		}
	}

	fmt.Println("")
	fmt.Printf("%sHint:%s Fill these in .env, or ignore if not used.\n", c.Dim, c.Reset)
	return 1
}

func cmdList(env parsedEnv, show bool) int {
	c := ui.Colors()
	if len(env.Values) == 0 {
		fmt.Println("No keys found in .env")
		return 1
	}

	keys := make([]string, 0, len(env.Values))
	width := 0
	for k := range env.Values {
		keys = append(keys, k)
		if len(k) > width {
			width = len(k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := env.Values[k]
		var masked string
		switch {
		case v == "":
			masked = fmt.Sprintf("%s(empty)%s", c.Dim, c.Reset)
		case show:
			masked = v
		case len(v) > 3:
			masked = "***"
		default:
			masked = "*"
		}
		fmt.Printf("  %s%-*s%s  %s\n", c.Cyan, width, k, c.Reset, masked)
	}
	return 0
}

func cmdDiff(example, env parsedEnv) int {
	c := ui.Colors()
	var onlyExample, onlyEnv, common []string
	for k := range example.Values {
		if _, ok := env.Values[k]; ok {
			common = append(common, k)
		} else {
			onlyExample = append(onlyExample, k)
		}
	}
	for k := range env.Values {
		if _, ok := example.Values[k]; !ok {
			onlyEnv = append(onlyEnv, k)
		}
	}
	sort.Strings(onlyExample)
	sort.Strings(onlyEnv)

	if len(onlyExample) == 0 && len(onlyEnv) == 0 {
		fmt.Printf("%sOK%s .env and .env.example have identical keys (%d)\n", c.Green, c.Reset, len(common))
		return 0
	}
	if len(onlyExample) > 0 {
		fmt.Printf("%sOnly in .env.example%s (%d):\n", c.Red, c.Reset, len(onlyExample))
		for _, k := range onlyExample {
			fmt.Printf("  - %s%s%s\n", c.Cyan, k, c.Reset)
		}
	}
	if len(onlyEnv) > 0 {
		if len(onlyExample) > 0 {
			fmt.Println("")
		}
		fmt.Printf("%sOnly in .env%s (%d):\n", c.Yellow, c.Reset, len(onlyEnv))
		for _, k := range onlyEnv {
			fmt.Printf("  + %s%s%s\n", c.Cyan, k, c.Reset)
		}
	}
	return 1
}

func cmdSync(example, env parsedEnv, envPath string, dryRun bool) int {
	c := ui.Colors()
	var missing []string
	for _, k := range example.Order {
		if _, ok := env.Values[k]; !ok {
			missing = append(missing, k)
		}
	}

	if len(missing) == 0 {
		fmt.Printf("%sOK%s .env already has all keys from .env.example (%d keys)\n", c.Green, c.Reset, len(example.Order))
		return 0
	}

	tty := isStdinTTY()
	newLines := make([]string, 0, len(missing))
	for _, k := range missing {
		val := example.RawValues[k]
		isQuoted := len(val) >= 2 && val[0] == val[len(val)-1] && (val[0] == '\'' || val[0] == '"')
		marker := val
		if !isQuoted {
			marker = strings.TrimSpace(strings.SplitN(val, "#", 2)[0])
		}

		var resolved string
		switch marker {
		case "GENERATE_SECRET":
			sec, err := generateSecret()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%sError:%s %v\n", c.Red, c.Reset, err)
				return 1
			}
			resolved = sec
			fmt.Printf("  %sgenerated%s %s%s%s\n", c.Green, c.Reset, c.Cyan, k, c.Reset)
		case "AUTO_HASH":
			if !tty {
				resolved = ""
				fmt.Printf("  %sskip%s %s (non-interactive)\n", c.Yellow, c.Reset, k)
			} else {
				pw, err := readPasswordPrompt(fmt.Sprintf("  password for %s: ", k))
				if err != nil || pw == "" {
					resolved = ""
					fmt.Printf("  %sskip%s %s (empty password)\n", c.Yellow, c.Reset, k)
				} else {
					h, err := hashPassword(pw)
					if err != nil {
						fmt.Fprintf(os.Stderr, "%sError:%s %v\n", c.Red, c.Reset, err)
						return 1
					}
					resolved = h
					fmt.Printf("  %shashed%s %s%s%s\n", c.Green, c.Reset, c.Cyan, k, c.Reset)
				}
			}
		case "":
			if tty {
				resolved = readLinePrompt(fmt.Sprintf("  %s %s(press Enter to skip)%s: ", k, c.Dim, c.Reset))
			}
			if resolved != "" {
				fmt.Printf("  %sset%s %s%s%s\n", c.Green, c.Reset, c.Cyan, k, c.Reset)
			} else {
				fmt.Printf("  %sskip%s %s\n", c.Yellow, c.Reset, k)
			}
		default:
			resolved = val
			fmt.Printf("  %sadded%s %s%s%s\n", c.Green, c.Reset, c.Cyan, k, c.Reset)
		}
		newLines = append(newLines, fmt.Sprintf("%s=%s", k, resolved))
	}

	if dryRun {
		fmt.Printf("\n%s--- dry run, would append to %s: ---%s\n", c.Dim, envPath, c.Reset)
		for _, ln := range newLines {
			fmt.Println(ln)
		}
		fmt.Printf("%s--- end ---%s\n", c.Dim, c.Reset)
		return 0
	}

	existingBytes, err := os.ReadFile(envPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError:%s %v\n", c.Red, c.Reset, err)
		return 1
	}
	existing := string(existingBytes)
	prefix := ""
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		prefix = "\n"
	}
	if err := os.WriteFile(envPath, []byte(existing+prefix+strings.Join(newLines, "\n")+"\n"), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "%sError:%s %v\n", c.Red, c.Reset, err)
		return 1
	}
	_ = os.Chmod(envPath, 0o600)

	fmt.Println()
	fmt.Printf("%sdone%s synced %d key(s) into %s\n", c.Green, c.Reset, len(newLines), envPath)
	return 0
}

func RunEnv(args []string) int {
	envPath := ".env"
	examplePath := ".env.example"
	cmd := ""
	show := false
	dryRun := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-h", "--help":
			printEnvHelp()
			return 0
		case "--env":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --env requires a file argument.")
				return 2
			}
			i++
			envPath = args[i]
		case "--example":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --example requires a file argument.")
				return 2
			}
			i++
			examplePath = args[i]
		case "--show":
			show = true
		case "-n", "--dry-run":
			dryRun = true
		default:
			if strings.HasPrefix(arg, "--env=") {
				envPath = strings.TrimPrefix(arg, "--env=")
			} else if strings.HasPrefix(arg, "--example=") {
				examplePath = strings.TrimPrefix(arg, "--example=")
			} else if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(os.Stderr, "Error: unknown option: %s\n", arg)
				return 2
			} else if cmd == "" {
				cmd = arg
			} else {
				fmt.Fprintf(os.Stderr, "Error: unexpected argument: %s\n", arg)
				return 2
			}
		}
	}

	if cmd == "" {
		printEnvHelp()
		return 0
	}

	if cmd != "check" && cmd != "list" && cmd != "diff" && cmd != "sync" {
		fmt.Fprintf(os.Stderr, "Error: unknown subcommand '%s'.\n", cmd)
		return 2
	}

	if _, err := os.Stat(examplePath); err != nil {
		dieMissing(examplePath, []string{
			"reta init        Scaffold .env.example in this directory",
			"reta init-env    Then generate .env from it",
		})
		return 1
	}

	if _, err := os.Stat(envPath); err != nil {
		dieMissing(envPath, []string{
			"reta init-env    Generate .env from .env.example",
		})
		return 1
	}

	example, err := parseEnvFile(examplePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	env, err := parseEnvFile(envPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	c := ui.Colors()
	if len(env.Duplicates) > 0 {
		fmt.Printf("%sWarning:%s duplicate keys in %s (%d): %s\n", c.Yellow, c.Reset, envPath, len(env.Duplicates), strings.Join(env.Duplicates, ", "))
		fmt.Printf("%sDuplicate keys cause inconsistent behavior across tools (docker, dotenv, bash). Keep only one definition per key.%s\n\n", c.Dim, c.Reset)
	}
	if len(example.Duplicates) > 0 {
		fmt.Printf("%sWarning:%s duplicate keys in %s (%d): %s\n\n", c.Yellow, c.Reset, examplePath, len(example.Duplicates), strings.Join(example.Duplicates, ", "))
	}

	fmt.Printf("%sreta env %s%s\n", c.Bold, cmd, c.Reset)
	fmt.Printf("  env:     %s\n", envPath)
	fmt.Printf("  example: %s\n", examplePath)
	fmt.Println()

	switch cmd {
	case "check":
		code := cmdCheck(example, env)
		if code == 0 {
			ui.NextSteps([]string{
				"reta env diff      Confirm .env.example keys match .env",
				"reta env list      Preview values with masking",
			})
		} else {
			ui.NextSteps([]string{
				"edit .env          Fill in the missing/empty keys above",
				"reta env check     Re-check after editing",
			})
		}
		return code
	case "list":
		code := cmdList(env, show)
		if code == 0 {
			ui.NextSteps([]string{
				"reta env check     Verify all keys are set",
				"reta env diff      Compare against .env.example",
			})
		}
		return code
	case "diff":
		code := cmdDiff(example, env)
		if code == 0 {
			ui.NextSteps([]string{
				"reta env check     Ensure values are filled",
				"reta env list      Preview current values",
			})
		} else {
			ui.NextSteps([]string{
				"reta env sync      Append missing keys into .env",
				"reta env check     Re-verify after syncing",
			})
		}
		return code
	case "sync":
		code := cmdSync(example, env, envPath, dryRun)
		if code == 0 && !dryRun {
			ui.NextSteps([]string{
				"reta env check     Verify all keys are filled",
				"reta env list      Preview keys with masked values",
			})
		}
		return code
	}
	return 1
}
