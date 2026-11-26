package search

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/eraygdev/reta-toolkit/internal/cli"
	"github.com/eraygdev/reta-toolkit/internal/config"
	"github.com/eraygdev/reta-toolkit/internal/retasign"
	"github.com/eraygdev/reta-toolkit/internal/ui"
)

var scanSkipDirs = map[string]bool{
	".git": true, ".hg": true, ".svn": true, ".cache": true, ".local": true, ".config": true,
	".Trash": true, "Trash": true, ".thumbnails": true, ".npm": true, ".yarn": true, ".pnpm-store": true,
	".cargo": true, ".rustup": true, ".go": true, ".m2": true, ".gradle": true, ".ivy2": true, ".sbt": true,
	".conda": true, ".pyenv": true, ".virtualenvs": true, ".rbenv": true, ".nvm": true, ".sdkman": true,
	".docker": true, ".kube": true, ".ssh": true, ".gnupg": true, ".pki": true, ".mozilla": true,
	".thunderbird": true, ".dbus": true, ".gvfs": true, ".var": true, "snap": true,
	"node_modules": true, "__pycache__": true, ".pytest_cache": true, ".mypy_cache": true,
	".ruff_cache": true, ".tox": true, ".nox": true, "venv": true, ".venv": true, "site-packages": true,
	"dist": true, "build": true, "target": true, "vendor": true, ".next": true, ".nuxt": true,
	".svelte-kit": true, ".astro": true, "coverage": true, "htmlcov": true, ".eggs": true,
	"exports": true, "backups": true, "logs": true, "tmp": true, "temp": true,
}

func printHelp() {
	fmt.Println("Usage: reta search [QUERY] [--register]")
	fmt.Println("")
	fmt.Println("Search and jump to reta projects (use 'rcd [query]').")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --register    Register current directory")
	fmt.Println("  -h, --help    Show this help message")
}

func registryFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "reta", "projects.txt")
}

func jumpFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "reta", "last_path")
}

func resolvePath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = filepath.Clean(p)
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real
	}
	return abs
}

func isSkippedPath(p string) bool {
	resolved := resolvePath(p)
	home, err := os.UserHomeDir()
	var parts []string
	if err == nil {
		homeResolved := resolvePath(home)
		if rel, err := filepath.Rel(homeResolved, resolved); err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
			parts = strings.Split(filepath.ToSlash(rel), "/")
		}
	}
	if parts == nil {
		slashed := filepath.ToSlash(resolved)
		trimmed := strings.TrimPrefix(slashed, "/")
		rawParts := strings.Split(trimmed, "/")
		if len(rawParts) > 2 && rawParts[0] == "private" && (rawParts[1] == "tmp" || rawParts[1] == "var") {
			parts = rawParts[2:]
		} else if len(rawParts) > 1 && (rawParts[0] == "tmp" || rawParts[0] == "var") {
			parts = rawParts[1:]
		} else {
			parts = rawParts
		}
	}
	for _, part := range parts {
		if scanSkipDirs[part] {
			return true
		}
	}
	return false
}

func saveProjects(paths []string) {
	rf := registryFile()
	if rf == "" {
		return
	}
	seen := make(map[string]bool)
	unique := make([]string, 0, len(paths))
	for _, p := range paths {
		rp := resolvePath(p)
		if !seen[rp] {
			seen[rp] = true
			unique = append(unique, rp)
		}
	}
	sort.Strings(unique)
	_ = os.MkdirAll(filepath.Dir(rf), 0o755)
	content := strings.Join(unique, "\n")
	if len(unique) > 0 {
		content += "\n"
	}
	_ = os.WriteFile(rf, []byte(content), 0o644)
}

func loadAndValidateCache(promptMissing bool) ([]string, map[string]string) {
	rf := registryFile()
	idMap := make(map[string]string)
	if rf == "" {
		return nil, idMap
	}
	data, err := os.ReadFile(rf)
	if err != nil {
		return nil, idMap
	}
	c := ui.Colors()
	rawLines := strings.Split(strings.TrimSuffix(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n"), "\n")
	if len(rawLines) == 1 && rawLines[0] == "" {
		rawLines = nil
	}

	var paths []string
	seen := make(map[string]bool)
	dirty := false

	for _, line := range rawLines {
		stripped := strings.TrimSpace(line)
		if stripped == "" {
			dirty = true
			continue
		}
		info, err := os.Stat(stripped)
		if err != nil || !info.IsDir() {
			dirty = true
			continue
		}
		resolved := resolvePath(stripped)
		if seen[resolved] || isSkippedPath(resolved) {
			dirty = true
			continue
		}

		sigID := retasign.ReadSignatureID(resolved)
		if sigID != "" {
			seen[resolved] = true
			paths = append(paths, resolved)
			idMap[resolved] = sigID
			continue
		}

		if !promptMissing {
			dirty = true
			continue
		}

		fmt.Fprintf(os.Stderr, "%sWarn:%s %s missing .retasign\n", c.Yellow, c.Reset, resolved)
		fmt.Fprint(os.Stderr, "[y/N] Add signature? (No removes from cache) ")
		ans, readErr := ui.ReadLine()
		if readErr != nil && ans == "" {
			fmt.Fprintln(os.Stderr, "")
			seen[resolved] = true
			paths = append(paths, resolved)
			idMap[resolved] = "rid?r"
			continue
		}
		ans = strings.ToLower(strings.TrimSpace(ans))
		if ans == "y" || ans == "yes" {
			newSig, wErr := retasign.WriteSignature(resolved)
			if wErr == nil {
				fmt.Fprintf(os.Stderr, "  %sOK%s    added .retasign (%s)\n", c.Green, c.Reset, newSig)
				seen[resolved] = true
				paths = append(paths, resolved)
				idMap[resolved] = newSig
			}
		} else {
			dirty = true
			fmt.Fprintf(os.Stderr, "  %sRemoved:%s %s\n", c.Dim, c.Reset, resolved)
		}
	}

	sort.Strings(paths)
	rawStripped := make([]string, 0, len(rawLines))
	for _, ln := range rawLines {
		rawStripped = append(rawStripped, strings.TrimSpace(ln))
	}
	same := len(paths) == len(rawStripped)
	if same {
		for i := range paths {
			if paths[i] != rawStripped[i] {
				same = false
				break
			}
		}
	}
	if dirty || !same {
		saveProjects(paths)
	}
	return paths, idMap
}

func scanHomeForProjects(root string, maxDepth int) map[string]string {
	found := make(map[string]string)
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() || maxDepth < 1 {
		return found
	}

	type frame struct {
		path  string
		depth int
	}
	stack := []frame{{path: root, depth: 0}}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		nextDepth := cur.depth + 1
		if nextDepth > maxDepth {
			continue
		}
		entries, err := os.ReadDir(cur.path)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			if scanSkipDirs[name] {
				continue
			}
			if cur.depth == 0 {
				if runtime.GOOS == "darwin" && name == "Library" {
					continue
				}
				if runtime.GOOS == "windows" && strings.EqualFold(name, "AppData") {
					continue
				}
			}
			if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
				continue
			}
			subPath := filepath.Join(cur.path, entry.Name())
			if sigID := retasign.ReadSignatureID(subPath); sigID != "" {
				found[resolvePath(subPath)] = sigID
			}
			if nextDepth < maxDepth {
				stack = append(stack, frame{path: subPath, depth: nextDepth})
			}
		}
	}
	return found
}

func mergeDiscovered(projects []string, idMap map[string]string, discovered map[string]string) ([]string, map[string]string) {
	changed := false
	mergedMap := make(map[string]string, len(idMap)+len(discovered))
	for k, v := range idMap {
		mergedMap[k] = v
	}
	for p, sig := range discovered {
		if _, exists := mergedMap[p]; !exists {
			changed = true
		}
		mergedMap[p] = sig
	}

	mergedPaths := make([]string, 0, len(mergedMap))
	for p := range mergedMap {
		mergedPaths = append(mergedPaths, p)
	}
	sort.Strings(mergedPaths)

	same := len(mergedPaths) == len(projects)
	if same {
		for i := range mergedPaths {
			if mergedPaths[i] != projects[i] {
				same = false
				break
			}
		}
	}
	if changed || !same {
		saveProjects(mergedPaths)
	}
	return mergedPaths, mergedMap
}

func RegisterDir(dir string) {
	projects, idMap := loadAndValidateCache(false)
	resolved := resolvePath(dir)
	if isSkippedPath(resolved) {
		return
	}
	sigID := retasign.ReadSignatureID(resolved)
	if sigID == "" {
		return
	}
	if _, exists := idMap[resolved]; exists {
		return
	}
	idMap[resolved] = sigID
	projects = append(projects, resolved)
	saveProjects(projects)
}

func removeFromCache(target string) {
	rf := registryFile()
	if rf == "" {
		return
	}
	data, err := os.ReadFile(rf)
	if err != nil {
		return
	}
	var kept []string
	for _, raw := range strings.Split(string(data), "\n") {
		s := strings.TrimSpace(raw)
		if s != "" && s != target {
			kept = append(kept, s)
		}
	}
	content := strings.Join(kept, "\n")
	if len(kept) > 0 {
		content += "\n"
	}
	_ = os.WriteFile(rf, []byte(content), 0o644)
}

func ensureInCache(target string) {
	rf := registryFile()
	if rf == "" {
		return
	}
	resolved := resolvePath(target)
	seen := make(map[string]bool)
	var lines []string
	if data, err := os.ReadFile(rf); err == nil {
		for _, raw := range strings.Split(string(data), "\n") {
			s := strings.TrimSpace(raw)
			if s != "" && !seen[s] {
				seen[s] = true
				lines = append(lines, s)
			}
		}
	}
	if !seen[resolved] {
		lines = append(lines, resolved)
		sort.Strings(lines)
		_ = os.MkdirAll(filepath.Dir(rf), 0o755)
		_ = os.WriteFile(rf, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
	}
}

func clearJump() {
	if jf := jumpFile(); jf != "" {
		_ = os.Remove(jf)
	}
}

func writeJump(target string) bool {
	c := ui.Colors()
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "%sError:%s Directory not found: %s\n", c.Red, c.Reset, target)
		removeFromCache(target)
		return false
	}
	resolved := resolvePath(target)
	if isSkippedPath(resolved) {
		fmt.Fprintf(os.Stderr, "%sError:%s Directory not found: %s\n", c.Red, c.Reset, target)
		removeFromCache(target)
		return false
	}
	if retasign.ReadSignatureID(resolved) == "" {
		fmt.Fprintf(os.Stderr, "%sWarn:%s %s missing .retasign\n", c.Yellow, c.Reset, resolved)
		fmt.Fprint(os.Stderr, "[y/N] Add signature? (No removes from cache) ")
		ans, readErr := ui.ReadLine()
		if readErr != nil && ans == "" {
			fmt.Fprintln(os.Stderr, "")
			return false
		}
		ans = strings.ToLower(strings.TrimSpace(ans))
		if ans == "y" || ans == "yes" {
			if sigID, err := retasign.WriteSignature(resolved); err == nil {
				fmt.Fprintf(os.Stderr, "  %sOK%s    added .retasign (%s)\n", c.Green, c.Reset, sigID)
			}
		} else {
			removeFromCache(resolved)
			fmt.Fprintf(os.Stderr, "  %sRemoved:%s %s\n", c.Dim, c.Reset, resolved)
			return false
		}
	}

	ensureInCache(resolved)
	jf := jumpFile()
	if jf != "" {
		_ = os.MkdirAll(filepath.Dir(jf), 0o755)
		_ = os.WriteFile(jf, []byte(resolved), 0o644)
	}
	fmt.Printf("Jumping to %s\n", resolved)
	return true
}

func formatDisplayPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	fullStr := filepath.Clean(p)
	for _, candidate := range []string{filepath.Clean(home), resolvePath(home)} {
		if fullStr == candidate {
			return "~"
		}
		prefix := candidate + string(filepath.Separator)
		if strings.HasPrefix(fullStr, prefix) {
			return "~/" + filepath.ToSlash(strings.TrimPrefix(fullStr, prefix))
		}
	}
	return fullStr
}

func printDepthHint() {
	ui.PrintDim(os.Stderr, "Missing a project? Increase depth: reta config search-depth <N>")
}

func printProjectList(projects []string, idMap map[string]string) {
	c := ui.Colors()
	counts := make(map[string]int)
	for _, p := range projects {
		counts[filepath.Base(p)]++
	}

	colWidth := 20
	for _, p := range projects {
		name := filepath.Base(p)
		w := utf8.RuneCountInString(name)
		if counts[name] > 1 {
			w += utf8.RuneCountInString(idMap[p])
		}
		if w > colWidth {
			colWidth = w
		}
	}

	fmt.Fprintf(os.Stderr, "%sProjects:%s\n", c.Bold, c.Reset)
	for i, p := range projects {
		name := filepath.Base(p)
		rel := formatDisplayPath(p)
		label := name
		visLen := utf8.RuneCountInString(name)
		if counts[name] > 1 {
			sigID := idMap[p]
			label = fmt.Sprintf("%s%s%s%s", name, c.Dim, sigID, c.Reset)
			visLen += utf8.RuneCountInString(sigID)
		}
		pad := colWidth - visLen
		if pad < 1 {
			pad = 1
		}
		fmt.Fprintf(os.Stderr, "  %s%2d)%s %s%s  %s%s%s\n", c.Cyan, i+1, c.Reset, label, strings.Repeat(" ", pad), c.Dim, rel, c.Reset)
	}
}

func resolveQuery(query string, projects []string) (string, []string) {
	if idx, err := strconv.Atoi(query); err == nil {
		pos := idx - 1
		if pos >= 0 && pos < len(projects) {
			return "jump", []string{projects[pos]}
		}
		return "none", nil
	}

	qLow := strings.ToLower(query)
	var exact, partial []string
	for _, p := range projects {
		baseLow := strings.ToLower(filepath.Base(p))
		if baseLow == qLow {
			exact = append(exact, p)
		} else if strings.Contains(baseLow, qLow) {
			partial = append(partial, p)
		}
	}
	if len(exact) >= 1 {
		return "jump", []string{exact[0]}
	}
	if len(partial) == 1 {
		return "jump", []string{partial[0]}
	}
	if len(partial) > 1 {
		return "menu", partial
	}
	return "none", nil
}

func Run(args []string) error {
	parser := cli.NewParser()
	var register bool
	var showHelp bool
	parser.Bool(&register, "register", "")
	parser.Bool(&showHelp, "help", "h")

	pos, err := parser.Parse(args)
	if err != nil {
		return err
	}
	if showHelp {
		printHelp()
		return nil
	}

	query := ""
	if len(pos) == 1 {
		query = pos[0]
	} else if len(pos) > 1 {
		return cli.Exit(2, "unexpected argument: %s", pos[1])
	}

	projects, idMap := loadAndValidateCache(!register)
	if cwd, err := os.Getwd(); err == nil {
		resolved := resolvePath(cwd)
		if !isSkippedPath(resolved) {
			if sigID := retasign.ReadSignatureID(resolved); sigID != "" {
				if _, exists := idMap[resolved]; !exists {
					idMap[resolved] = sigID
					projects = append(projects, resolved)
					sort.Strings(projects)
					saveProjects(projects)
				}
			}
		}
	}

	if register {
		return nil
	}

	clearJump()
	searchDepth := config.SearchDepth()
	home, _ := os.UserHomeDir()

	if query != "" {
		status, matched := resolveQuery(query, projects)
		if status == "jump" {
			if writeJump(matched[0]) {
				return nil
			}
			return cli.Exit(1, "")
		}
		if status == "none" {
			discovered := scanHomeForProjects(home, searchDepth)
			projects, idMap = mergeDiscovered(projects, idMap, discovered)
			status, matched = resolveQuery(query, projects)
			if status == "jump" {
				if writeJump(matched[0]) {
					return nil
				}
				return cli.Exit(1, "")
			}
		}
		if status == "menu" {
			projects = matched
		} else {
			fmt.Fprintf(os.Stderr, "No project matching '%s'.\n", query)
			printDepthHint()
			return cli.Exit(1, "")
		}
	} else {
		discovered := scanHomeForProjects(home, searchDepth)
		projects, idMap = mergeDiscovered(projects, idMap, discovered)
	}

	c := ui.Colors()
	if len(projects) == 0 {
		fmt.Fprintf(os.Stderr, "%sNo projects found. Run 'reta init' in a project.%s\n", c.Dim, c.Reset)
		printDepthHint()
		return cli.Exit(1, "")
	}

	printProjectList(projects, idMap)
	fmt.Fprintln(os.Stderr, "")
	printDepthHint()

	fmt.Fprintf(os.Stderr, "\n%sSelect (1-%d, or q):%s ", c.Bold, len(projects), c.Reset)
	choice, err := ui.ReadLine()
	if err != nil && choice == "" {
		fmt.Fprintln(os.Stderr, "")
		return nil
	}
	choice = strings.TrimSpace(choice)
	if choice == "" || strings.ToLower(choice) == "q" {
		return nil
	}
	if idx, err := strconv.Atoi(choice); err == nil {
		posIdx := idx - 1
		if posIdx >= 0 && posIdx < len(projects) {
			if writeJump(projects[posIdx]) {
				return nil
			}
			return cli.Exit(1, "")
		}
	}

	fmt.Fprintln(os.Stderr, "Invalid selection.")
	return cli.Exit(1, "")
}
