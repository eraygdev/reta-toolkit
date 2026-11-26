package envfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eraygdev/reta-toolkit/internal/ui"
)

func printInitEnvHelp() {
	fmt.Println("Usage: reta init-env [-i INPUT] [-o OUTPUT] [-f] [-n]")
	fmt.Println("")
	fmt.Println("Generate .env from .env.example. Resolves special markers.")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -i, --input FILE   Template file (default: .env.example)")
	fmt.Println("  -o, --output FILE  Output file (default: .env)")
	fmt.Println("  -f, --force        Overwrite output if it exists")
	fmt.Println("  -n, --dry-run      Show result without writing")
	fmt.Println("  -h, --help         Show this help message")
}

func RunInitEnv(args []string) int {
	inputPath := ".env.example"
	outputPath := ".env"
	force := false
	dryRun := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-h", "--help":
			printInitEnvHelp()
			return 0
		case "-f", "--force":
			force = true
		case "-n", "--dry-run":
			dryRun = true
		case "-i", "--input":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: %s requires a file argument.\n", arg)
				return 2
			}
			i++
			inputPath = args[i]
		case "-o", "--output":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: %s requires a file argument.\n", arg)
				return 2
			}
			i++
			outputPath = args[i]
		default:
			if strings.HasPrefix(arg, "--input=") {
				inputPath = strings.TrimPrefix(arg, "--input=")
			} else if strings.HasPrefix(arg, "--output=") {
				outputPath = strings.TrimPrefix(arg, "--output=")
			} else {
				fmt.Fprintf(os.Stderr, "Error: unknown option: %s\n", arg)
				return 2
			}
		}
	}

	c := ui.Colors()
	if _, err := os.Stat(inputPath); err != nil {
		cwd, _ := os.Getwd()
		fmt.Fprintf(os.Stderr, "%sError:%s %s not found in %s\n", c.Red, c.Reset, inputPath, cwd)
		return 1
	}

	if _, err := os.Stat(outputPath); err == nil && !force {
		fmt.Fprintf(os.Stderr, "%sError:%s %s already exists. Use %s--force%s to overwrite.\n", c.Red, c.Reset, outputPath, c.Bold, c.Reset)
		return 1
	}

	rawData, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError:%s %v\n", c.Red, c.Reset, err)
		return 1
	}

	fmt.Printf("%sreta init-env%s\n", c.Bold, c.Reset)
	fmt.Printf("  source: %s\n", inputPath)
	fmt.Printf("  output: %s\n", outputPath)
	fmt.Println()

	text := strings.TrimSuffix(strings.ReplaceAll(string(rawData), "\r\n", "\n"), "\n")
	rawLines := strings.Split(text, "\n")
	outputLines := make([]string, 0, len(rawLines))
	secretCount, hashCount, promptCount, skipCount := 0, 0, 0, 0
	tty := isStdinTTY()

	for _, rawLine := range rawLines {
		stripped := strings.TrimSpace(rawLine)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			outputLines = append(outputLines, rawLine)
			continue
		}

		if strings.Contains(rawLine, "=") {
			exportPrefix := ""
			lineBody := rawLine
			if strings.HasPrefix(stripped, "export ") {
				idx := strings.Index(rawLine, "export ")
				exportPrefix = rawLine[:idx+7]
				lineBody = rawLine[idx+7:]
			}

			key, value, _ := strings.Cut(lineBody, "=")
			key = strings.TrimRight(key, " \t")
			val := strings.TrimSpace(value)
			marker := strings.TrimSpace(strings.SplitN(val, "#", 2)[0])

			switch marker {
			case "GENERATE_SECRET":
				newVal, err := generateSecret()
				if err != nil {
					fmt.Fprintf(os.Stderr, "%sError:%s %v\n", c.Red, c.Reset, err)
					return 1
				}
				outputLines = append(outputLines, fmt.Sprintf("%s%s=%s", exportPrefix, key, newVal))
				secretCount++
				fmt.Printf("  %sgenerated%s %s%s%s\n", c.Green, c.Reset, c.Cyan, key, c.Reset)
			case "AUTO_HASH":
				if !tty {
					fmt.Printf("  %sskip%s %s (non-interactive)\n", c.Yellow, c.Reset, key)
					outputLines = append(outputLines, fmt.Sprintf("%s%s=", exportPrefix, key))
					skipCount++
				} else {
					pw, err := readPasswordPrompt(fmt.Sprintf("  password for %s: ", key))
					if err != nil || pw == "" {
						fmt.Printf("  %sskip%s %s (empty password)\n", c.Yellow, c.Reset, key)
						outputLines = append(outputLines, fmt.Sprintf("%s%s=", exportPrefix, key))
						skipCount++
					} else {
						h, err := hashPassword(pw)
						if err != nil {
							fmt.Fprintf(os.Stderr, "%sError:%s %v\n", c.Red, c.Reset, err)
							return 1
						}
						outputLines = append(outputLines, fmt.Sprintf("%s%s=%s", exportPrefix, key, h))
						hashCount++
						fmt.Printf("  %shashed%s %s%s%s\n", c.Green, c.Reset, c.Cyan, key, c.Reset)
					}
				}
			case "":
				userVal := ""
				if tty {
					userVal = readLinePrompt(fmt.Sprintf("  %s %s(press Enter to skip)%s: ", key, c.Dim, c.Reset))
				}
				outputLines = append(outputLines, fmt.Sprintf("%s%s=%s", exportPrefix, key, userVal))
				promptCount++
				if userVal != "" {
					fmt.Printf("  %sset%s %s%s%s\n", c.Green, c.Reset, c.Cyan, key, c.Reset)
				} else {
					fmt.Printf("  %sskip%s %s\n", c.Yellow, c.Reset, key)
				}
			default:
				outputLines = append(outputLines, rawLine)
			}
		} else {
			outputLines = append(outputLines, rawLine)
		}
	}

	result := strings.Join(outputLines, "\n") + "\n"
	if dryRun {
		fmt.Printf("\n%s--- dry run, would write: ---%s\n", c.Dim, c.Reset)
		fmt.Print(result)
		fmt.Printf("%s--- end ---%s\n", c.Dim, c.Reset)
		return 0
	}

	if dir := filepath.Dir(outputPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "%sError:%s %v\n", c.Red, c.Reset, err)
			return 1
		}
	}
	if err := os.WriteFile(outputPath, []byte(result), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "%sError:%s %v\n", c.Red, c.Reset, err)
		return 1
	}
	_ = os.Chmod(outputPath, 0o600)

	fmt.Println()
	fmt.Printf("%sdone%s %s\n", c.Green, c.Reset, outputPath)
	fmt.Printf("  %d secret(s), %d hash(es), %d prompt(s), %d skipped\n", secretCount, hashCount, promptCount, skipCount)

	if giData, err := os.ReadFile(".gitignore"); err == nil {
		ignored := false
		patterns := map[string]bool{
			".env": true, "/.env": true, ".env*": true, "/.env*": true,
			"*.env": true, "**/.env": true, "**/.env*": true,
		}
		for _, raw := range strings.Split(string(giData), "\n") {
			pat := strings.TrimSpace(raw)
			if pat == "" || strings.HasPrefix(pat, "#") {
				continue
			}
			if patterns[pat] {
				ignored = true
				break
			}
		}
		if !ignored {
			fmt.Printf("\n%swarn%s .env is not in .gitignore — add it before committing!\n", c.Yellow, c.Reset)
		}
	}

	steps := []string{
		"reta env check     Verify all keys are filled",
		"reta env list      Preview keys with masked values",
	}
	if _, err := os.Stat("Makefile"); err == nil {
		steps = append(steps, "make help          List available make targets")
	}
	ui.NextSteps(steps)
	return 0
}
