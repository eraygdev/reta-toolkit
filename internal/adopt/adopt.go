package adopt

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	reta "github.com/eraygdev/reta-toolkit"
	"github.com/eraygdev/reta-toolkit/internal/cli"
	"github.com/eraygdev/reta-toolkit/internal/ui"
)

const HIBlock = "hi:  ## Say hello\n\t@echo \"Hi👋\"\n"

const fallbackHeader = `# Project Makefile (powered by ~/.reta/lib/base.mk)
#
# Rules & usage:
#   - Inherited targets: ` + "`up`" + `, ` + "`down`" + `, ` + "`logs`" + `, ` + "`clean`" + `, ` + "`export`" + `, ` + "`git-sync`" + `, ...
#   - Do NOT define ` + "`help:`" + ` — ` + "`base.mk`" + ` generates a combined ` + "`help`" + ` for all targets.
#   - Override any other base target (e.g. ` + "`clean`" + `, ` + "`up`" + `) by defining it below
#     ` + "`include $(RETA_BASE)`" + `. (Converted Makefiles comment out colliding rules
#     first; uncomment any of them to restore your custom behavior.)
#   - Section heading: ` + "`# === Section Name ===`" + ` (shown as a bold yellow header).
#   - Documented target: ` + "`name:  ## Short description`" + ` (targets without ` + "`##`" + `
#     still run normally, but stay hidden from ` + "`make help`" + `).

RETA_BASE := $(HOME)/.reta/lib/base.mk
ifeq ($(wildcard $(RETA_BASE)),)
$(info ❌ [reta] reta toolkit is not installed.)
$(info Install: git clone https://github.com/eraygdev/reta-toolkit.git ~/.reta && ~/.reta/install.sh)
$(error $(RETA_BASE) missing)
endif
include $(RETA_BASE)
`

var (
	targetLineBaseRE   = regexp.MustCompile(`^([A-Za-z0-9_./%-]+(?:\s+[A-Za-z0-9_./%-]+)*)\s*:(.*)$`)
	assignRE           = regexp.MustCompile(`^[A-Za-z0-9_.-]+\s*(?:\?=|:=|\+=|!=|=)`)
	decoratedHeadingRE = regexp.MustCompile(`^#\s*(?:===+|---|###|\*\*\*|═══+|───+|━━━+)\s*(.+?)\s*(?:===+|---|###|\*\*\*|═══+|───+|━━━+)\s*$`)
	bannerLineRE       = regexp.MustCompile(`^#\s*[=\-*#═─━~_]{4,}\s*$`)
	helpEchoCmdRE      = regexp.MustCompile(`^\t\s*@?echo\s+["']\s*(make\s+)?([A-Za-z0-9_.-]+)(?:\s{2,}|\s+[-:]\s+)(.+?)["']\s*$`)
	helpPrintfCmdRE    = regexp.MustCompile(`^\t\s*@?printf\s+["'][^"']*["']\s+["']([A-Za-z0-9_.-]+)["']\s+["'](.+?)["']\s*$`)
	condStartRE        = regexp.MustCompile(`^(?:ifeq|ifneq|ifdef|ifndef)\b`)
	condEndRE          = regexp.MustCompile(`^endif\b`)
	defineStartRE      = regexp.MustCompile(`^(?:override\s+)?define\b`)
	baseMatrixRE       = regexp.MustCompile(`^([a-zA-Z0-9_.-]+):(?:[^=][^#]*)?## `)
	halfDecoratedRE    = regexp.MustCompile(`^(?:===+|---|###|\*\*\*)\s*(.+)$`)
)

var redundantDefaults = map[string]bool{
	"SHELL := /bin/bash":    true,
	"SHELL = /bin/bash":     true,
	".DEFAULT_GOAL := help": true,
	".DEFAULT_GOAL = help":  true,
}

var directives = map[string]bool{
	".PHONY":                true,
	".DEFAULT_GOAL":         true,
	".SILENT":               true,
	".IGNORE":               true,
	".EXPORT_ALL_VARIABLES": true,
	".PRECIOUS":             true,
	".SECONDARY":            true,
}

type block struct {
	Kind  string
	Lines []string
}

type Report struct {
	Headings         []string
	CommentedTargets []string
	AnnotatedTargets []string
	HiddenTargets    []string
	EditHere         []string
}

type Options struct {
	DryRun      bool
	NoBackup    bool
	NoHints     bool
	SummaryOnly bool
}

func matchTargetLine(line string) (targetsPart, rest string, ok bool) {
	m := targetLineBaseRE.FindStringSubmatch(line)
	if len(m) != 3 {
		return "", "", false
	}
	if len(m[2]) > 0 && (m[2][0] == '=' || m[2][0] == ':') {
		return "", "", false
	}
	return m[1], m[2], true
}

func GenerateBaseMatrix(baseMK string) string {
	var targets []string
	for _, raw := range strings.Split(strings.ReplaceAll(baseMK, "\r\n", "\n"), "\n") {
		if m := baseMatrixRE.FindStringSubmatch(raw); len(m) == 2 {
			tname := m[1]
			if tname != "" && !strings.HasPrefix(tname, ".") && tname != "help" {
				targets = append(targets, tname)
			}
		}
	}
	if len(targets) == 0 {
		return ""
	}
	const cols = 5
	maxLen := 0
	for _, t := range targets {
		if len(t) > maxLen {
			maxLen = len(t)
		}
	}
	maxLen += 4

	lines := []string{"# Inherited base targets (matrix view):"}
	var row []string
	for _, t := range targets {
		pad := maxLen - len(t)
		if pad < 0 {
			pad = 0
		}
		row = append(row, t+strings.Repeat(" ", pad))
		if len(row) == cols {
			lines = append(lines, "#   "+strings.TrimRight(strings.Join(row, ""), " "))
			row = nil
		}
	}
	if len(row) > 0 {
		lines = append(lines, "#   "+strings.TrimRight(strings.Join(row, ""), " "))
	}
	return strings.Join(lines, "\n")
}

func LoadTemplateHeader(forAdopt, includeHI bool) string {
	matrix := GenerateBaseMatrix(reta.BaseMK)
	wantHI := forAdopt && includeHI

	tplBytes, err := fs.ReadFile(reta.TemplatesFS, "templates/Makefile")
	if err != nil {
		base := fallbackHeader
		if matrix != "" {
			base += "\n\n" + matrix
		}
		if wantHI {
			return strings.TrimRight(base, "\n") + "\n\n" + HIBlock
		}
		return base
	}

	content := strings.ReplaceAll(string(tplBytes), "\r\n", "\n")
	if strings.Contains(content, HIBlock) {
		content = strings.Replace(content, HIBlock, "", 1)
	}

	hiTrimmed := strings.TrimRight(HIBlock, "\n")
	if strings.Contains(content, "# === BASE_MATRIX ===") {
		matrixBlock := strings.TrimSpace(matrix)
		if wantHI {
			replacement := hiTrimmed
			if matrixBlock != "" {
				replacement = matrixBlock + "\n\n" + hiTrimmed
			}
			content = strings.ReplaceAll(content, "# === BASE_MATRIX ===", replacement)
		} else {
			content = strings.ReplaceAll(content, "# === BASE_MATRIX ===", matrixBlock)
		}
	} else if matrix != "" && strings.Contains(content, "include $(RETA_BASE)") {
		extra := "\n\n" + strings.TrimSpace(matrix)
		if wantHI {
			extra += "\n\n" + hiTrimmed
		}
		content = strings.ReplaceAll(content, "include $(RETA_BASE)", "include $(RETA_BASE)"+extra)
	} else if wantHI {
		content = strings.ReplaceAll(content, "include $(RETA_BASE)", "include $(RETA_BASE)\n\n"+hiTrimmed)
	}

	return strings.TrimRight(content, "\n \t") + "\n"
}

func LoadBaseTargets(baseMK string) map[string]bool {
	targets := map[string]bool{"help": true}
	for _, line := range strings.Split(strings.ReplaceAll(baseMK, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		targetsPart, _, ok := matchTargetLine(line)
		if !ok {
			continue
		}
		for _, name := range strings.Fields(targetsPart) {
			if !directives[name] && !strings.HasPrefix(name, ".") {
				targets[name] = true
			}
		}
	}
	return targets
}

func AlreadyUsesReta(text string) bool {
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		s := strings.TrimSpace(raw)
		if strings.HasPrefix(s, "#") {
			continue
		}
		if strings.Contains(s, "RETA_BASE") || (strings.Contains(s, "base.mk") && strings.Contains(s, "include")) {
			return true
		}
	}
	return false
}

func splitBlocks(lines []string) []block {
	var blocks []block
	n := len(lines)
	for i := 0; i < n; {
		line := lines[i]
		stripped := strings.TrimSpace(line)

		if stripped == "" {
			blocks = append(blocks, block{Kind: "blank", Lines: []string{line}})
			i++
			continue
		}

		if condStartRE.MatchString(stripped) {
			chunk := []string{line}
			depth := 1
			i++
			for i < n && depth > 0 {
				cur := lines[i]
				cs := strings.TrimSpace(cur)
				if condStartRE.MatchString(cs) {
					depth++
				} else if condEndRE.MatchString(cs) {
					depth--
				}
				chunk = append(chunk, cur)
				i++
			}
			blocks = append(blocks, block{Kind: "directive", Lines: chunk})
			continue
		}

		if defineStartRE.MatchString(stripped) {
			chunk := []string{line}
			i++
			for i < n {
				cur := lines[i]
				chunk = append(chunk, cur)
				i++
				if strings.TrimSpace(cur) == "endef" {
					break
				}
			}
			blocks = append(blocks, block{Kind: "directive", Lines: chunk})
			continue
		}

		if strings.HasPrefix(stripped, "#") {
			blocks = append(blocks, block{Kind: "comment", Lines: []string{line}})
			i++
			continue
		}

		if !strings.HasPrefix(line, "\t") && assignRE.MatchString(line) {
			chunk := []string{line}
			for strings.HasSuffix(strings.TrimRight(chunk[len(chunk)-1], " \t"), `\`) && i+1 < n {
				i++
				chunk = append(chunk, lines[i])
			}
			blocks = append(blocks, block{Kind: "directive", Lines: chunk})
			i++
			continue
		}

		if !strings.HasPrefix(line, "\t") {
			if targetsPart, _, ok := matchTargetLine(line); ok {
				fields := strings.Fields(targetsPart)
				firstTok := ""
				if len(fields) > 0 {
					firstTok = fields[0]
				}
				if directives[firstTok] || strings.HasPrefix(firstTok, ".") {
					chunk := []string{line}
					for strings.HasSuffix(strings.TrimRight(chunk[len(chunk)-1], " \t"), `\`) && i+1 < n {
						i++
						chunk = append(chunk, lines[i])
					}
					blocks = append(blocks, block{Kind: "directive", Lines: chunk})
					i++
					continue
				}

				chunk := []string{line}
				for strings.HasSuffix(strings.TrimRight(chunk[len(chunk)-1], " \t"), `\`) && i+1 < n {
					i++
					chunk = append(chunk, lines[i])
				}
				i++

				for i < n {
					nxt := lines[i]
					if strings.HasPrefix(nxt, "\t") {
						chunk = append(chunk, nxt)
						for strings.HasSuffix(strings.TrimRight(chunk[len(chunk)-1], " \t"), `\`) && i+1 < n {
							i++
							chunk = append(chunk, lines[i])
						}
						i++
					} else if strings.TrimSpace(nxt) == "" {
						look := i + 1
						for look < n && strings.TrimSpace(lines[look]) == "" {
							look++
						}
						if look < n && strings.HasPrefix(lines[look], "\t") {
							chunk = append(chunk, nxt)
							i++
						} else {
							break
						}
					} else {
						break
					}
				}
				blocks = append(blocks, block{Kind: "target", Lines: chunk})
				continue
			}
		}

		blocks = append(blocks, block{Kind: "raw", Lines: []string{line}})
		i++
	}
	return blocks
}

func collectTargetNames(blocks []block) map[string]bool {
	names := make(map[string]bool)
	for _, b := range blocks {
		if b.Kind != "target" || len(b.Lines) == 0 {
			continue
		}
		if targetsPart, _, ok := matchTargetLine(b.Lines[0]); ok {
			for _, name := range strings.Fields(targetsPart) {
				names[name] = true
			}
		}
	}
	return names
}

func extractHelpDescriptions(blocks []block) map[string]string {
	known := collectTargetNames(blocks)
	descs := make(map[string]string)
	for _, b := range blocks {
		if b.Kind != "target" || len(b.Lines) == 0 {
			continue
		}
		targetsPart, _, ok := matchTargetLine(b.Lines[0])
		if !ok {
			continue
		}
		hasHelp := false
		for _, n := range strings.Fields(targetsPart) {
			if n == "help" {
				hasHelp = true
				break
			}
		}
		if !hasHelp {
			continue
		}
		for _, line := range b.Lines[1:] {
			if m := helpEchoCmdRE.FindStringSubmatch(line); len(m) == 4 {
				hasMakePrefix := m[1] != ""
				tname := strings.TrimSpace(m[2])
				desc := strings.TrimRight(strings.TrimSpace(m[3]), `"'`)
				if !hasMakePrefix && !known[tname] {
					continue
				}
				if tname != "" && desc != "" {
					if _, exists := descs[tname]; !exists {
						descs[tname] = desc
					}
				}
				continue
			}
			if m := helpPrintfCmdRE.FindStringSubmatch(line); len(m) == 3 {
				tname := strings.TrimSpace(m[1])
				desc := strings.TrimRight(strings.TrimSpace(m[2]), `"'`)
				if tname != "" && desc != "" {
					if _, exists := descs[tname]; !exists {
						descs[tname] = desc
					}
				}
			}
		}
	}
	return descs
}

func nextNonBlankKind(blocks []block, idx int) string {
	for j := idx + 1; j < len(blocks); j++ {
		if blocks[j].Kind == "blank" {
			continue
		}
		if blocks[j].Kind == "comment" && bannerLineRE.MatchString(strings.TrimSpace(blocks[j].Lines[0])) {
			continue
		}
		return blocks[j].Kind
	}
	return ""
}

func classifyHeading(blocks []block, idx int) (status, title string) {
	line := strings.TrimSpace(blocks[idx].Lines[0])
	if bannerLineRE.MatchString(line) {
		return "banner", ""
	}
	if m := decoratedHeadingRE.FindStringSubmatch(line); len(m) == 2 {
		t := strings.TrimSpace(m[1])
		if t != "" && !strings.HasPrefix(strings.ToLower(t), "edit_here") {
			return "heading", t
		}
	}

	text := strings.TrimSpace(strings.TrimLeft(line, "#"))
	if text == "" {
		return "normal", ""
	}

	prevIsBanner := idx > 0 && blocks[idx-1].Kind == "comment" && bannerLineRE.MatchString(strings.TrimSpace(blocks[idx-1].Lines[0]))
	prevIsBlankOrBanner := idx == 0 || blocks[idx-1].Kind == "blank" || prevIsBanner
	nextIsBanner := idx+1 < len(blocks) && blocks[idx+1].Kind == "comment" && bannerLineRE.MatchString(strings.TrimSpace(blocks[idx+1].Lines[0]))

	runeLen := utf8.RuneCountInString(text)
	if prevIsBanner && nextIsBanner && runeLen <= 100 {
		return "heading", text
	}
	if runeLen > 48 {
		return "normal", ""
	}

	nxt := nextNonBlankKind(blocks, idx)
	words := strings.Fields(text)
	hasLetter := strings.IndexFunc(text, unicode.IsLetter) >= 0
	looksUpper := text == strings.ToUpper(text) && hasLetter && len(words) <= 5 && !strings.Contains(text, "=") && !strings.Contains(text, ":")

	if looksUpper && prevIsBlankOrBanner && nxt == "target" {
		return "heading", text
	}
	if looksUpper && prevIsBlankOrBanner {
		return "uncertain", text
	}

	if m := halfDecoratedRE.FindStringSubmatch(text); len(m) == 2 {
		cand := strings.TrimSpace(m[1])
		if cand != "" {
			return "uncertain", cand
		}
	}

	allTitleCase := len(words) >= 1 && len(words) <= 3
	if allTitleCase {
		for _, w := range words {
			r, _ := utf8.DecodeRuneInString(w)
			if r == utf8.RuneError || !unicode.IsLetter(r) || !unicode.IsUpper(r) {
				allTitleCase = false
				break
			}
		}
	}
	if allTitleCase && !strings.HasSuffix(text, ".") && !strings.Contains(text, "=") && !strings.Contains(text, ":") && prevIsBlankOrBanner && nxt == "target" {
		return "uncertain", text
	}
	return "normal", ""
}

func adjacentCommentDesc(blocks []block, targetIdx int) string {
	var comments []string
	for j := targetIdx - 1; j >= 0 && blocks[j].Kind == "comment"; j-- {
		status, _ := classifyHeading(blocks, j)
		if status != "normal" {
			return ""
		}
		text := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(blocks[j].Lines[0]), "#"))
		comments = append(comments, text)
	}
	if len(comments) == 0 {
		return ""
	}
	for i, j := 0, len(comments)-1; i < j; i, j = i+1, j-1 {
		comments[i], comments[j] = comments[j], comments[i]
	}
	for _, text := range comments {
		if text == "" {
			continue
		}
		low := strings.ToLower(text)
		if strings.HasPrefix(low, "kullanim:") || strings.HasPrefix(low, "usage:") || strings.HasPrefix(low, "[reta]") || strings.HasPrefix(low, "edit_here") {
			continue
		}
		if utf8.RuneCountInString(text) <= 80 {
			return text
		}
	}
	return ""
}

func convertTargetHeader(headerLine string, helpDescs map[string]string, fallbackDesc string) (string, string) {
	targetsPart, rest, ok := matchTargetLine(headerLine)
	if !ok {
		return "existing", headerLine
	}
	if strings.Contains(rest, "##") {
		return "existing", headerLine
	}
	if idx := strings.Index(rest, "#"); idx >= 0 {
		before := rest[:idx]
		comment := strings.TrimSpace(rest[idx+1:])
		if comment != "" {
			return "annotated", fmt.Sprintf("%s:%s  ## %s", targetsPart, strings.TrimRight(before, " \t"), comment)
		}
		return "existing", headerLine
	}

	names := strings.Fields(targetsPart)
	if len(names) == 1 {
		if desc, ok := helpDescs[names[0]]; ok && desc != "" {
			sep := "  "
			if strings.HasSuffix(rest, " ") {
				sep = " "
			}
			return "annotated", fmt.Sprintf("%s:%s%s## %s", targetsPart, strings.TrimRight(rest, " \t"), sep, desc)
		}
	}

	cleanRest := strings.TrimRight(strings.SplitN(rest, "#", 2)[0], " \t")
	sep := "  "
	if strings.HasSuffix(cleanRest, " ") {
		sep = " "
	}
	if fallbackDesc != "" {
		return "annotated", fmt.Sprintf("%s:%s%s## %s", targetsPart, cleanRest, sep, fallbackDesc)
	}
	return "placeholder", fmt.Sprintf("%s:%s%s## EDIT_HERE: add description", targetsPart, cleanRest, sep)
}

func ConvertMakefile(content string, baseTargets map[string]bool, headerBlock string) (string, Report) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(strings.TrimSuffix(normalized, "\n"), "\n")
	blocks := splitBlocks(lines)
	helpDescs := extractHelpDescriptions(blocks)

	if collectTargetNames(blocks)["hi"] && strings.Contains(headerBlock, strings.TrimRight(HIBlock, "\n")) {
		headerBlock = strings.ReplaceAll(headerBlock, HIBlock, "")
		headerBlock = strings.ReplaceAll(headerBlock, strings.TrimRight(HIBlock, "\n"), "")
	}

	var report Report
	outLines := []string{strings.TrimRight(headerBlock, "\n"), ""}

	skipLeadingBlanks := true
	for i := 0; i < len(blocks); i++ {
		b := blocks[i]
		if skipLeadingBlanks && b.Kind == "blank" {
			continue
		}
		skipLeadingBlanks = false

		switch b.Kind {
		case "comment":
			status, title := classifyHeading(blocks, i)
			switch status {
			case "banner":
				prevH := false
				if i-1 >= 0 && blocks[i-1].Kind == "comment" {
					st, _ := classifyHeading(blocks, i-1)
					prevH = st == "heading"
				}
				nextH := false
				if i+1 < len(blocks) && blocks[i+1].Kind == "comment" {
					st, _ := classifyHeading(blocks, i+1)
					nextH = st == "heading"
				}
				if prevH || nextH {
					continue
				}
				outLines = append(outLines, b.Lines...)
			case "heading":
				outLines = append(outLines, fmt.Sprintf("# === %s ===", title))
				report.Headings = append(report.Headings, title)
			case "uncertain":
				outLines = append(outLines, fmt.Sprintf("# EDIT_HERE: verify if '%s' is a section heading -> # === %s ===", title, title))
				outLines = append(outLines, b.Lines...)
				report.EditHere = append(report.EditHere, "Possible section heading: "+title)
			default:
				outLines = append(outLines, b.Lines...)
			}

		case "target":
			targetsPart, _, ok := matchTargetLine(b.Lines[0])
			var names []string
			if ok {
				names = strings.Fields(targetsPart)
			}
			var colliding []string
			for _, n := range names {
				if baseTargets[n] {
					colliding = append(colliding, n)
				}
			}

			if len(colliding) > 0 {
				joined := strings.Join(colliding, ", ")
				if len(colliding) == 1 && colliding[0] == "help" {
					outLines = append(outLines, "# [reta] Commented out old 'help' target (base.mk provides combined help)")
				} else {
					outLines = append(outLines, fmt.Sprintf("# [reta] Commented out '%s' (provided by base.mk; uncomment to override)", joined))
				}
				for _, ln := range b.Lines {
					if ln != "" {
						outLines = append(outLines, "# "+ln)
					} else {
						outLines = append(outLines, "#")
					}
				}
				report.CommentedTargets = append(report.CommentedTargets, colliding...)
			} else {
				fallbackDesc := adjacentCommentDesc(blocks, i)
				status, newHead := convertTargetHeader(b.Lines[0], helpDescs, fallbackDesc)
				if status == "annotated" && len(names) > 0 {
					report.AnnotatedTargets = append(report.AnnotatedTargets, names[0])
				} else if status == "placeholder" && len(names) > 0 {
					report.HiddenTargets = append(report.HiddenTargets, names[0])
					report.EditHere = append(report.EditHere, "Missing description on target: "+names[0])
				}
				outLines = append(outLines, newHead)
				outLines = append(outLines, b.Lines[1:]...)
			}

		case "directive":
			if len(b.Lines) == 1 {
				norm := strings.Join(strings.Fields(b.Lines[0]), " ")
				if redundantDefaults[norm] {
					outLines = append(outLines, fmt.Sprintf("# [reta] Commented redundant '%s' (defined in base.mk)", norm))
					outLines = append(outLines, "# "+b.Lines[0])
					continue
				}
			}
			outLines = append(outLines, b.Lines...)

		default:
			outLines = append(outLines, b.Lines...)
		}
	}

	bodyLines := outLines[1:]
	var boundaryIndices []int
	for idx, ln := range bodyLines {
		if (strings.HasPrefix(ln, "# === ") && strings.HasSuffix(ln, " ===")) || strings.HasPrefix(ln, "# EDIT_HERE: verify if ") {
			boundaryIndices = append(boundaryIndices, idx)
		}
	}

	for pos, hIdx := range boundaryIndices {
		lnHead := bodyLines[hIdx]
		if !(strings.HasPrefix(lnHead, "# === ") && strings.HasSuffix(lnHead, " ===")) {
			continue
		}
		title := strings.TrimSpace(lnHead[6 : len(lnHead)-4])
		endIdx := len(bodyLines)
		if pos+1 < len(boundaryIndices) {
			endIdx = boundaryIndices[pos+1]
		}
		sectionSlice := bodyLines[hIdx+1 : endIdx]

		isHelpSection := false
		if strings.ToUpper(title) == "HELP" {
			for _, ln := range sectionSlice {
				if strings.HasPrefix(ln, "# [reta] Commented out old 'help' target") {
					isHelpSection = true
					break
				}
			}
		}

		hasActiveTarget := false
		if !isHelpSection {
			for _, ln := range sectionSlice {
				if strings.TrimSpace(ln) == "" || strings.HasPrefix(ln, "#") || strings.HasPrefix(ln, "\t") {
					continue
				}
				if assignRE.MatchString(ln) {
					continue
				}
				if tp, _, ok := matchTargetLine(ln); ok {
					fields := strings.Fields(tp)
					if len(fields) > 0 && !directives[fields[0]] && !strings.HasPrefix(fields[0], ".") {
						hasActiveTarget = true
						break
					}
				}
			}
		}

		if !hasActiveTarget {
			bodyLines[hIdx] = "# " + title
			for idx, h := range report.Headings {
				if h == title {
					report.Headings = append(report.Headings[:idx], report.Headings[idx+1:]...)
					break
				}
			}
		}
	}

	combined := append([]string{outLines[0]}, bodyLines...)
	var cleaned []string
	blankRun := 0
	for _, ln := range combined {
		if strings.TrimSpace(ln) == "" {
			blankRun++
			if blankRun <= 2 {
				cleaned = append(cleaned, "")
			}
		} else {
			blankRun = 0
			cleaned = append(cleaned, ln)
		}
	}

	result := strings.TrimRight(strings.Join(cleaned, "\n"), "\n \t") + "\n"
	return result, report
}

func fmtList(items []string, limit int) string {
	if len(items) <= limit {
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%s (+%d more)", strings.Join(items[:limit], ", "), len(items)-limit)
}

func AdoptFile(targetPath string, opts Options) (bool, Report, error) {
	c := ui.Colors()
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return false, Report{}, err
	}
	original := string(data)
	if AlreadyUsesReta(original) {
		if !opts.SummaryOnly {
			fmt.Printf("  %sskip  %s (already uses base.mk)%s\n", c.Dim, filepath.Base(targetPath), c.Reset)
		}
		return false, Report{}, nil
	}

	baseTargets := LoadBaseTargets(reta.BaseMK)
	normalized := strings.ReplaceAll(original, "\r\n", "\n")
	blocks := splitBlocks(strings.Split(strings.TrimSuffix(normalized, "\n"), "\n"))
	hasCustomHI := collectTargetNames(blocks)["hi"]
	headerBlock := LoadTemplateHeader(true, !hasCustomHI)

	converted, report := ConvertMakefile(original, baseTargets, headerBlock)
	name := filepath.Base(targetPath)

	if opts.DryRun {
		fmt.Printf("%s--- dry run: %s ---%s\n", c.Dim, targetPath, c.Reset)
		fmt.Print(converted)
		fmt.Printf("%s--- end ---%s\n", c.Dim, c.Reset)
	} else {
		backupPath := targetPath + ".bak"
		if !opts.NoBackup {
			if err := os.WriteFile(backupPath, data, 0o644); err != nil {
				return false, report, err
			}
		}
		if err := os.WriteFile(targetPath, []byte(converted), 0o644); err != nil {
			return false, report, err
		}
		if !opts.SummaryOnly {
			fmt.Printf("  %sconv  %s%s\n", c.Green, name, c.Reset)
			if !opts.NoBackup {
				fmt.Printf("  %sbackup %s%s\n", c.Dim, filepath.Base(backupPath), c.Reset)
			}
		}
	}

	if !opts.SummaryOnly {
		fmt.Println("")
	}
	if len(report.Headings) > 0 {
		fmt.Printf("  %sHeadings:%s          %s\n", c.Cyan, c.Reset, fmtList(report.Headings, 5))
	}
	if len(report.AnnotatedTargets) > 0 {
		fmt.Printf("  %sAdded '##': %s       %s\n", c.Cyan, c.Reset, fmtList(report.AnnotatedTargets, 5))
	}
	if len(report.HiddenTargets) > 0 {
		fmt.Printf("  %sNeeds description:%s %s %s(marked with EDIT_HERE)%s\n", c.Yellow, c.Reset, fmtList(report.HiddenTargets, 5), c.Dim, c.Reset)
	}
	if len(report.CommentedTargets) > 0 {
		fmt.Printf("  %sCommented out:%s     %s %s(already in base.mk; uncomment to override)%s\n", c.Yellow, c.Reset, fmtList(report.CommentedTargets, 5), c.Dim, c.Reset)
	}
	if len(report.EditHere) > 0 {
		fmt.Printf("  %sReview EDIT_HERE:%s  %d item(s) marked in %s\n", c.Yellow, c.Reset, len(report.EditHere), name)
	}

	if !opts.NoHints {
		steps := []string{"make help          Verify converted targets and headings"}
		if len(report.EditHere) > 0 {
			steps = append(steps, fmt.Sprintf("grep EDIT_HERE %s   Review marked items", name))
		}
		if !opts.DryRun && !opts.NoBackup {
			steps = append(steps, fmt.Sprintf("rm %s.bak     Remove backup once verified", name))
		}
		ui.NextSteps(steps)
	}
	return true, report, nil
}

func printHelp() {
	fmt.Println("Usage: reta adopt-makefile [TARGET] [options]")
	fmt.Println("")
	fmt.Println("Convert an existing Makefile to use ~/.reta/lib/base.mk.")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -n, --dry-run       Preview without modifying")
	fmt.Println("      --no-backup     Skip creating Makefile.bak")
	fmt.Println("      --no-hints      Suppress Next steps output")
	fmt.Println("      --summary-only  Print only conversion summary")
	fmt.Println("  -h, --help          Show this help message")
}

func Run(args []string) error {
	parser := cli.NewParser()
	var showHelp bool
	var opts Options

	parser.Bool(&showHelp, "help", "h")
	parser.Bool(&opts.DryRun, "dry-run", "n")
	parser.Bool(&opts.NoBackup, "no-backup", "")
	parser.Bool(&opts.NoHints, "no-hints", "")
	parser.Bool(&opts.SummaryOnly, "summary-only", "")

	pos, err := parser.Parse(args)
	if err != nil {
		return err
	}
	if showHelp {
		printHelp()
		return nil
	}

	target := "Makefile"
	if len(pos) == 1 {
		target = pos[0]
	} else if len(pos) > 1 {
		return cli.Exit(2, "unexpected argument: %s", pos[1])
	}

	info, err := os.Stat(target)
	if err != nil || !info.Mode().IsRegular() {
		return cli.Exit(1, "%s not found.", target)
	}

	if _, _, err := AdoptFile(target, opts); err != nil {
		return cli.Wrap(1, err, "")
	}
	return nil
}
