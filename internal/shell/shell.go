package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/eraygdev/reta-toolkit/internal/cli"
)

var allCommands = []string{
	"init", "init-env", "env", "config", "hook", "update", "uninstall",
	"clean", "search", "check-secrets", "export", "adopt-makefile", "port",
	"doctor", "completion", "shell-init", "help",
}

func detectShell() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	sh := filepath.Base(os.Getenv("SHELL"))
	if sh == "zsh" {
		return "zsh"
	}
	return "bash"
}

func RunShellInit(args []string) error {
	parser := cli.NewParser()
	var showHelp bool
	parser.Bool(&showHelp, "help", "h")

	pos, err := parser.Parse(args)
	if err != nil {
		return err
	}
	if showHelp {
		fmt.Println("Usage: reta shell-init [bash|zsh|powershell]")
		fmt.Println("")
		fmt.Println("Print shell integration code for 'rcd' project jumping.")
		return nil
	}

	target := ""
	if len(pos) == 1 {
		switch pos[0] {
		case "bash", "zsh", "powershell":
			target = pos[0]
		default:
			return cli.Exit(2, "unsupported shell '%s' (choose bash, zsh, powershell).", pos[0])
		}
	} else if len(pos) > 1 {
		return cli.Exit(2, "unexpected argument: %s", pos[1])
	}

	if target == "" {
		target = detectShell()
	}

	switch target {
	case "powershell":
		fmt.Print(`function rcd {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Query)
    $userHome = if ($env:USERPROFILE) { $env:USERPROFILE } elseif ($env:HOME) { $env:HOME } else { $HOME }
    $jumpFile = Join-Path $userHome ".cache\reta\last_path"
    if (Test-Path $jumpFile) { Remove-Item $jumpFile -Force }
    & reta search @Query
    if ($LASTEXITCODE -ne 0) { return }
    if (Test-Path $jumpFile) {
        $target = (Get-Content $jumpFile -Raw).Trim()
        Remove-Item $jumpFile -Force
        if ($target -and (Test-Path $target -PathType Container)) {
            Set-Location $target
        }
    }
}
`)
	default:
		fmt.Print(`rcd() {
    if ! command -v reta >/dev/null 2>&1; then
        printf "❌ [reta] reta toolkit is not installed.\n" >&2
        return 1
    fi
    local jump_file="$HOME/.cache/reta/last_path"
    rm -f "$jump_file"
    reta search "$@" || return $?
    if [ -f "$jump_file" ]; then
        local target
        target="$(cat "$jump_file")"
        rm -f "$jump_file"
        if [ -n "$target" ] && [ -d "$target" ]; then
            cd "$target" || return $?
        fi
    fi
}
`)
	}
	return nil
}

func RunCompletion(args []string) error {
	parser := cli.NewParser()
	var showHelp bool
	parser.Bool(&showHelp, "help", "h")

	pos, err := parser.Parse(args)
	if err != nil {
		return err
	}
	if showHelp {
		fmt.Println("Usage: reta completion [bash|zsh|powershell]")
		fmt.Println("")
		fmt.Println("Generate shell completion script.")
		return nil
	}

	target := ""
	if len(pos) == 1 {
		switch pos[0] {
		case "bash", "zsh", "powershell":
			target = pos[0]
		default:
			return cli.Exit(2, "unsupported shell '%s' (choose bash, zsh, powershell).", pos[0])
		}
	} else if len(pos) > 1 {
		return cli.Exit(2, "unexpected argument: %s", pos[1])
	}

	if target == "" {
		target = detectShell()
	}

	cmdList := strings.Join(allCommands, " ")
	switch target {
	case "bash":
		fmt.Printf(`_reta_completions() {
    local cur prev
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    local cmds="%s"

    if [ "$COMP_CWORD" -eq 1 ]; then
        COMPREPLY=($(compgen -W "$cmds --version --help -v -h" -- "$cur"))
        return
    fi

    case "${COMP_WORDS[1]}" in
        help)
            [ "$COMP_CWORD" -eq 2 ] && COMPREPLY=($(compgen -W "$cmds" -- "$cur"))
            ;;
        init)
            COMPREPLY=($(compgen -d -W "--convert -h --help" -- "$cur"))
            ;;
        init-env)
            case "$prev" in
                -i|--input|-o|--output) COMPREPLY=($(compgen -f -- "$cur")) ;;
                *) COMPREPLY=($(compgen -W "-i --input -o --output -f --force -n --dry-run -h --help" -- "$cur")) ;;
            esac
            ;;
        env)
            if [ "$prev" = "--env" ] || [ "$prev" = "--example" ]; then
                COMPREPLY=($(compgen -f -- "$cur"))
            elif [ "$COMP_CWORD" -eq 2 ]; then
                COMPREPLY=($(compgen -W "check list diff sync --env --example -h --help" -- "$cur"))
            else
                COMPREPLY=($(compgen -W "--show -n --dry-run --env --example -h --help" -- "$cur"))
            fi
            ;;
        config)
            if [ "$COMP_CWORD" -eq 2 ]; then
                COMPREPLY=($(compgen -W "agents-file agents-template hints learn search-depth --reset -h --help" -- "$cur"))
            elif [ "$COMP_CWORD" -eq 3 ]; then
                case "${COMP_WORDS[2]}" in
                    hints|learn) COMPREPLY=($(compgen -W "on off" -- "$cur")) ;;
                    search-depth) COMPREPLY=($(compgen -W "4" -- "$cur")) ;;
                    agents-file) COMPREPLY=($(compgen -W "AGENTS.md" -- "$cur")) ;;
                    agents-template) COMPREPLY=($(compgen -f -- "$cur")) ;;
                esac
            fi
            ;;
        port)
            [ "$COMP_CWORD" -eq 2 ] && COMPREPLY=($(compgen -W "list kill docker ps help -h --help" -- "$cur"))
            ;;
        hook)
            [ "$COMP_CWORD" -eq 2 ] && COMPREPLY=($(compgen -W "install remove status -h --help" -- "$cur"))
            ;;
        completion|shell-init)
            [ "$COMP_CWORD" -eq 2 ] && COMPREPLY=($(compgen -W "bash zsh powershell -h --help" -- "$cur"))
            ;;
    esac
}
complete -F _reta_completions reta
`, cmdList)

	case "zsh":
		fmt.Printf(`#compdef reta
_reta() {
    local -a cmds
    cmds=(%s)
    _arguments -C \
        '(-h --help)'{-h,--help}'[show help]' \
        '(-v --version)'{-v,--version}'[show version]' \
        '1:command:->cmds' \
        '*::arg:->args'

    case $state in
        cmds) _describe 'command' cmds ;;
        args)
            case "$words[1]" in
                help) _describe 'command' cmds ;;
                init) _arguments '--convert[convert Makefile]' '1:directory:_directories' ;;
                env) _values 'env' check list diff sync --show -n --dry-run --env --example ;;
                config) _values 'config' agents-file agents-template hints learn search-depth --reset ;;
                port) _values 'port' list kill docker ps help ;;
                hook) _values 'hook' install remove status ;;
                completion|shell-init) _values 'shell' bash zsh powershell ;;
            esac
            ;;
    esac
}
_reta "$@"
`, cmdList)

	case "powershell":
		quoted := make([]string, len(allCommands))
		for i, c := range allCommands {
			quoted[i] = "'" + c + "'"
		}
		fmt.Printf(`Register-ArgumentCompleter -Native -CommandName reta -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    $cmds = @(%s)
    $cmds | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
    }
}
`, strings.Join(quoted, ", "))
	}
	return nil
}
