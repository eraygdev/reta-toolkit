#compdef reta
# Zsh completion for reta.
#
# Installed automatically by install.sh into ~/.zfunc/_reta.
# Requires ~/.zshrc to contain:
#   fpath+=~/.zfunc
#   autoload -Uz compinit && compinit

_reta() {
    local -a cmds
    local libexec="$HOME/.reta/libexec"
    local f

    if [ -d "$libexec" ]; then
        for f in "$libexec"/reta-*; do
            [ -x "$f" ] || continue
            cmds+=("$(basename "$f" | sed 's/^reta-//')")
        done
    fi
    cmds+=('help')

    _arguments -C \
        '(-h --help)'{-h,--help}'[show help]' \
        '(-v --version)'{-v,--version}'[show version]' \
        '1:command:->cmds' \
        '*::arg:->args'

    case $state in
        cmds)
            _describe 'command' cmds
            ;;
        args)
            case "$words[1]" in
                help)
                    if [ "$CURRENT" -eq 2 ]; then
                        _describe 'command' cmds
                    fi
                    ;;
                init)
                    if [ "$CURRENT" -eq 2 ]; then
                        _directories
                    fi
                    ;;
                init-env)
                    _arguments \
                        '(-i --input)'{-i,--input}'[template file]:file:_files' \
                        '(-o --output)'{-o,--output}'[output file]:file:_files' \
                        '(-f --force)'{-f,--force}'[overwrite output if it exists]' \
                        '(-n --dry-run)'{-n,--dry-run}'[show result without writing]' \
                        '(-h --help)'{-h,--help}'[show help]'
                    ;;
                env)
                    if [ "$CURRENT" -eq 2 ]; then
                        _values 'env command' check list diff sync --env --example -h --help
                    else
                        _values 'option' --show -n --dry-run --env --example -h --help
                    fi
                    ;;
                port)
                    if [ "$CURRENT" -eq 2 ]; then
                        _values 'port command' list kill help -h --help
                    fi
                    ;;
                config)
                    if [ "$CURRENT" -eq 2 ]; then
                        _values 'config key' hints '--reset' -h --help
                    elif [ "$CURRENT" -eq 3 ] && [ "$words[2]" = "hints" ]; then
                        _values 'value' on off
                    fi
                    ;;
                uninstall)
                    _values 'option' --keep --keep-source -y --yes -h --help
                    ;;
            esac
            ;;
    esac
}

_reta "$@"