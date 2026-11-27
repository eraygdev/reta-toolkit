package projectinit

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	reta "github.com/eraygdev/reta-toolkit"
	"github.com/eraygdev/reta-toolkit/internal/adopt"
	"github.com/eraygdev/reta-toolkit/internal/cli"
	"github.com/eraygdev/reta-toolkit/internal/config"
	"github.com/eraygdev/reta-toolkit/internal/retasign"
	"github.com/eraygdev/reta-toolkit/internal/search"
	"github.com/eraygdev/reta-toolkit/internal/ui"
	"golang.org/x/term"
)

func printHelp() {
	fmt.Println("Usage: reta init [DIR] [--convert]")
	fmt.Println("")
	fmt.Println("  DIR          Target directory (default: current directory)")
	fmt.Println("  --convert    Convert existing Makefile to reta template without prompting")
	fmt.Println("")
	fmt.Println("Agent file settings (via 'reta config'):")
	fmt.Println("  reta config agents-file <name>       Rename file (default: AGENTS.md; \"\" disables)")
	fmt.Println("  reta config agents-template <path>   Fill file from a local prompt file")
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func Run(args []string) error {
	parser := cli.NewParser()
	var convertMakefile bool
	var showHelp bool
	parser.Bool(&convertMakefile, "convert", "")
	parser.Bool(&showHelp, "help", "h")

	pos, err := parser.Parse(args)
	if err != nil {
		return err
	}
	if showHelp {
		printHelp()
		return nil
	}

	targetArg := "."
	if len(pos) == 1 {
		targetArg = pos[0]
	} else if len(pos) > 1 {
		return cli.Exit(1, "unexpected argument: %s", pos[1])
	}

	info, err := os.Stat(targetArg)
	if err != nil || !info.IsDir() {
		return cli.Exit(1, "target directory does not exist: %s", targetArg)
	}
	targetDir, err := filepath.Abs(targetArg)
	if err != nil {
		targetDir = targetArg
	}
	if real, err := filepath.EvalSymlinks(targetDir); err == nil {
		targetDir = real
	}

	fmt.Println("reta init")
	fmt.Printf("Target: %s\n", targetDir)
	fmt.Println("")

	entries, err := fs.ReadDir(reta.TemplatesFS, "templates")
	if err != nil {
		return cli.Wrap(1, err, "reading embedded templates")
	}

	copied, skipped, converted := 0, 0, 0
	tty := isInteractive()
	reader := bufio.NewReader(os.Stdin)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		dst := filepath.Join(targetDir, name)

		if dstInfo, statErr := os.Stat(dst); statErr == nil {
			if name == "Makefile" && dstInfo.Mode().IsRegular() {
				existingData, _ := os.ReadFile(dst)
				if adopt.AlreadyUsesReta(string(existingData)) {
					fmt.Printf("  skip  %s (already uses base.mk)\n", name)
					skipped++
					continue
				}
				doConvert := convertMakefile
				if !doConvert && tty {
					fmt.Print("  Makefile already exists. Convert it to reta template? [y/N] ")
					ans, _ := reader.ReadString('\n')
					ans = strings.ToLower(strings.TrimSpace(ans))
					doConvert = ans == "y" || ans == "yes"
				}
				if doConvert {
					did, _, err := adopt.AdoptFile(dst, adopt.Options{SummaryOnly: true, NoHints: true})
					if err != nil {
						return cli.Wrap(1, err, "converting Makefile")
					}
					if did {
						fmt.Printf("  conv  %s (backup: %s.bak)\n", name, name)
						converted++
					} else {
						skipped++
					}
				} else {
					fmt.Printf("  skip  %s (exists)\n", name)
					skipped++
				}
			} else {
				fmt.Printf("  skip  %s (exists)\n", name)
				skipped++
			}
		} else {
			if name == "Makefile" {
				content := adopt.LoadTemplateHeader(false, false)
				if err := os.WriteFile(dst, []byte(content), 0o644); err != nil {
					return cli.Wrap(1, err, "writing %s", dst)
				}
				fmt.Printf("  ok    %s (dynamic)\n", name)
			} else {
				tplBytes, err := fs.ReadFile(reta.TemplatesFS, "templates/"+name)
				if err != nil {
					return cli.Wrap(1, err, "reading template %s", name)
				}
				if err := os.WriteFile(dst, tplBytes, 0o644); err != nil {
					return cli.Wrap(1, err, "writing %s", dst)
				}
				fmt.Printf("  ok    %s\n", name)
			}
			copied++
		}
	}

	agentsFile := config.AgentsFile()
	agentsTemplate := config.AgentsTemplate()
	if agentsFile != "" {
		agentsDst := filepath.Join(targetDir, filepath.FromSlash(agentsFile))
		if fileExists(agentsDst) {
			fmt.Printf("  skip  %s (exists)\n", agentsFile)
			skipped++
		} else {
			if err := os.MkdirAll(filepath.Dir(agentsDst), 0o755); err != nil {
				return cli.Wrap(1, err, "creating directory for %s", agentsDst)
			}
			var content []byte
			note := "(empty)"
			if agentsTemplate != "" {
				tplPath := config.ExpandPath(agentsTemplate)
				if data, readErr := os.ReadFile(tplPath); readErr == nil {
					content = data
					note = "(from template)"
				} else {
					fmt.Printf("  warn  %s (template not found: %s; created empty)\n", agentsFile, tplPath)
					note = ""
				}
			}
			if err := os.WriteFile(agentsDst, content, 0o644); err != nil {
				return cli.Wrap(1, err, "writing %s", agentsDst)
			}
			if note != "" {
				fmt.Printf("  ok    %s %s\n", agentsFile, note)
			}
			copied++
		}
	}

	signPath := filepath.Join(targetDir, retasign.SignFilename)
	if fileExists(signPath) {
		fmt.Printf("  skip  %s (exists)\n", retasign.SignFilename)
		skipped++
	} else {
		if _, err := retasign.WriteSignature(targetDir); err != nil {
			return cli.Wrap(1, err, "writing %s", signPath)
		}
		fmt.Printf("  ok    %s\n", retasign.SignFilename)
		copied++
	}

	fmt.Println("")
	if converted > 0 {
		fmt.Printf("Done: %d copied, %d converted, %d skipped.\n", copied, converted, skipped)
	} else {
		fmt.Printf("Done: %d copied, %d skipped.\n", copied, skipped)
	}

	search.RegisterDir(targetDir)

	var next []string
	if fileExists(filepath.Join(targetDir, ".env.example")) && !fileExists(filepath.Join(targetDir, ".env")) {
		next = append(next, "reta init-env      Generate .env from .env.example")
	}
	if agentsFile != "" && agentsTemplate == "" {
		next = append(next, fmt.Sprintf("reta config agents-template <file>  Auto-fill %s on init", agentsFile))
	}
	if !fileExists(filepath.Join(targetDir, ".git")) {
		next = append(next, "git init           Initialize a git repository")
	}
	mkPath := filepath.Join(targetDir, "Makefile")
	if fileExists(mkPath) {
		if converted > 0 {
			next = append(next, "make help                Verify converted targets and headings")
			if data, err := os.ReadFile(mkPath); err == nil && strings.Contains(string(data), "EDIT_HERE") {
				next = append(next, "grep EDIT_HERE Makefile  Review marked items")
			}
			if fileExists(filepath.Join(targetDir, "Makefile.bak")) {
				next = append(next, "rm Makefile.bak          Remove backup once verified")
			}
		} else {
			next = append(next, "make help          List available make targets")
		}
	}
	next = append(next, "rcd                Jump to any project instantly (type 'rcd')")

	fmt.Println("")
	if runtime.GOOS == "windows" {
		ui.PrintDim(os.Stdout, "Tip: reload your PowerShell profile ('. $PROFILE') if 'rcd' is not loaded yet")
	} else if strings.HasSuffix(os.Getenv("SHELL"), "zsh") {
		ui.PrintDim(os.Stdout, "Tip: execute 'source ~/.zshrc' if 'rcd' is not loaded yet")
	} else {
		ui.PrintDim(os.Stdout, "Tip: execute 'source ~/.bashrc' if 'rcd' is not loaded yet")
	}
	ui.NextSteps(next)
	return nil
}
