package cli

import (
	"fmt"
	"strings"
)

// ExitError represents an error associated with a specific CLI exit code.
type ExitError struct {
	Code    int
	Message string
	Err     error
}

func (e *ExitError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit code %d", e.Code)
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

// Exit returns an ExitError with the specified exit code and formatted message.
func Exit(code int, format string, a ...any) error {
	msg := ""
	if format != "" || len(a) > 0 {
		msg = fmt.Sprintf(format, a...)
	}
	return &ExitError{
		Code:    code,
		Message: msg,
	}
}

// Wrap returns an ExitError wrapping an underlying error with a code and optional message.
func Wrap(code int, err error, format string, a ...any) error {
	msg := ""
	if format != "" || len(a) > 0 {
		msg = fmt.Sprintf(format, a...)
	}
	return &ExitError{
		Code:    code,
		Message: msg,
		Err:     err,
	}
}

// Parser provides lightweight POSIX/GNU style CLI flag parsing without external dependencies.
type Parser struct {
	boolFlags   map[string]*bool
	stringFlags map[string]*string
	stringLists map[string]*[]string
	aliases     map[string]string
	args        []string
}

// NewParser initializes a clean CLI argument parser.
func NewParser() *Parser {
	return &Parser{
		boolFlags:   make(map[string]*bool),
		stringFlags: make(map[string]*string),
		stringLists: make(map[string]*[]string),
		aliases:     make(map[string]string),
	}
}

// Bool registers a boolean flag with an optional short alias.
func (p *Parser) Bool(target *bool, name string, short string) {
	p.boolFlags[name] = target
	if short != "" {
		p.aliases[short] = name
	}
}

// String registers a string flag with an optional short alias.
func (p *Parser) String(target *string, name string, short string) {
	p.stringFlags[name] = target
	if short != "" {
		p.aliases[short] = name
	}
}

// StringSlice registers a repeatable string slice flag with an optional short alias.
func (p *Parser) StringSlice(target *[]string, name string, short string) {
	p.stringLists[name] = target
	if short != "" {
		p.aliases[short] = name
	}
}

// Parse processes arguments, updates bound flag pointers, and returns non-flag positional arguments.
func (p *Parser) Parse(args []string) ([]string, error) {
	var positional []string
	n := len(args)

	for i := 0; i < n; i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}

		if strings.HasPrefix(arg, "--") {
			name := strings.TrimPrefix(arg, "--")
			var val string
			hasVal := false
			if eqIdx := strings.IndexByte(name, '='); eqIdx >= 0 {
				val = name[eqIdx+1:]
				name = name[:eqIdx]
				hasVal = true
			}

			if bTarget, ok := p.boolFlags[name]; ok {
				if hasVal {
					return nil, Exit(2, "flag --%s does not take a value", name)
				}
				*bTarget = true
				continue
			}

			if sTarget, ok := p.stringFlags[name]; ok {
				if hasVal {
					*sTarget = val
					continue
				}
				if i+1 >= n {
					return nil, Exit(2, "flag --%s requires an argument", name)
				}
				i++
				*sTarget = args[i]
				continue
			}

			if listTarget, ok := p.stringLists[name]; ok {
				if hasVal {
					*listTarget = append(*listTarget, val)
					continue
				}
				if i+1 >= n {
					return nil, Exit(2, "flag --%s requires an argument", name)
				}
				i++
				*listTarget = append(*listTarget, args[i])
				continue
			}

			return nil, Exit(2, "unknown option: %s", arg)
		}

		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			short := strings.TrimPrefix(arg, "-")
			canonical, hasAlias := p.aliases[short]
			if !hasAlias {
				canonical = short
			}

			if bTarget, ok := p.boolFlags[canonical]; ok {
				*bTarget = true
				continue
			}

			if sTarget, ok := p.stringFlags[canonical]; ok {
				if i+1 >= n {
					return nil, Exit(2, "option %s requires an argument", arg)
				}
				i++
				*sTarget = args[i]
				continue
			}

			if listTarget, ok := p.stringLists[canonical]; ok {
				if i+1 >= n {
					return nil, Exit(2, "option %s requires an argument", arg)
				}
				i++
				*listTarget = append(*listTarget, args[i])
				continue
			}

			return nil, Exit(2, "unknown option: %s", arg)
		}

		positional = append(positional, arg)
	}

	p.args = positional
	return positional, nil
}
