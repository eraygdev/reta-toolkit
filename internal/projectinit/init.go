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
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func Run(args []string) int {
	targetArg := "."
	targetSet := false
	convertMakefile := false

	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			printHelp()
			return 0
		case "--convert":
			convertMakefile = true
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(os.Stderr, "Error: unknown option: %s\n", arg)
				return 1
			}
			if !targetSet {
				targetArg = arg
				targetSet = true
			} else {
				fmt.Fprintf(os.Stderr, "Error: unexpected argument: %s\n", arg)
				return 1
			}
		}
	}

	info, err := os.Stat(targetArg)
	if err != nil || !info.IsDir() {
		fmt.Printf("Error: target directory does not exist: %s\n", targetArg)
		return 1
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
		fmt.Fprintf(os.Stderr, "Error reading embedded templates: %v\n", err)
		return 1
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
						fmt.Fprintf(os.Stderr, "Error converting Makefile: %v\n", err)
						return 1
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
					fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", dst, err)
					return 1
				}
				fmt.Printf("  ok    %s (dynamic)\n", name)
			} else {
				tplBytes, err := fs.ReadFile(reta.TemplatesFS, "templates/"+name)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading template %s: %v\n", name, err)
					return 1
				}
				if err := os.WriteFile(dst, tplBytes, 0o644); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", dst, err)
					return 1
				}
				fmt.Printf("  ok    %s\n", name)
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
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", signPath, err)
			return 1
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
	return 0
}
