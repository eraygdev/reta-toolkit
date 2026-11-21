#!/usr/bin/env bash
# Bash completion for reta.
#
# Installed automatically by install.sh into:
#   ~/.local/share/bash-completion/completions/reta
#
# Manual install for ~/.bashrc:
#   source ~/.reta/lib/completion.bash

_reta_completions() {
    local cur prev
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    local RETA_LIBEXEC="$HOME/.reta/libexec"

    local -a cmds=()
    if [ -d "$RETA_LIBEXEC" ]; then
        local f
        for f in "$RETA_LIBEXEC"/reta-*; do
            [ -x "$f" ] || continue
            cmds+=("$(basename "$f" | sed 's/^reta-//')")
        done
    fi

    if [ "$COMP_CWORD" -eq 1 ]; then
        COMPREPLY=($(compgen -W "${cmds[*]} help --version --help -v -h" -- "$cur"))
        return
    fi

    case "${COMP_WORDS[1]}" in
        help)
            if [ "$COMP_CWORD" -eq 2 ]; then
                COMPREPLY=($(compgen -W "${cmds[*]}" -- "$cur"))
            fi
            ;;
        init)
            if [ "$COMP_CWORD" -eq 2 ]; then
                COMPREPLY=($(compgen -d -- "$cur"))
            fi
            ;;
        env)
            if [ "$prev" = "--env" ] || [ "$prev" = "--example" ]; then
                COMPREPLY=($(compgen -f -- "$cur"))
            elif [ "$COMP_CWORD" -eq 2 ]; then
                COMPREPLY=($(compgen -W "check list diff sync --env --example -h --help" -- "$cur"))
            else
                case " ${COMP_WORDS[*]} " in
                    *" list "*)
                        COMPREPLY=($(compgen -W "--show --env --example -h --help" -- "$cur"))
                        ;;
                    *" sync "*)
                        COMPREPLY=($(compgen -W "-n --dry-run --env --example -h --help" -- "$cur"))
                        ;;
                    *)
                        COMPREPLY=($(compgen -W "--env --example -h --help" -- "$cur"))
                        ;;
                esac
            fi
            ;;
        port)
            if [ "$COMP_CWORD" -eq 2 ]; then
                COMPREPLY=($(compgen -W "list kill help -h --help" -- "$cur"))
            fi
            ;;
        config)
            if [ "$COMP_CWORD" -eq 2 ]; then
                COMPREPLY=($(compgen -W "hints --reset -h --help" -- "$cur"))
            elif [ "$COMP_CWORD" -eq 3 ] && [ "${COMP_WORDS[2]}" = "hints" ]; then
                COMPREPLY=($(compgen -W "on off" -- "$cur"))
            fi
            ;;
        uninstall)
            COMPREPLY=($(compgen -W "--keep --keep-source -y --yes -h --help" -- "$cur"))
            ;;
        init-env)
            case "$prev" in
                -i|--input|-o|--output)
                    COMPREPLY=($(compgen -f -- "$cur"))
                    ;;
                *)
                    COMPREPLY=($(compgen -W "-i --input -o --output -f --force -n --dry-run -h --help" -- "$cur"))
                    ;;
            esac
            ;;
    esac
}

complete -F _reta_completions reta