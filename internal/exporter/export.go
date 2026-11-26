package exporter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/eraygdev/reta-toolkit/internal/ui"
)

var includeExt = map[string]bool{
	".py": true, ".pyi": true, ".js": true, ".jsx": true, ".mjs": true, ".cjs": true,
	".ts": true, ".tsx": true, ".vue": true, ".svelte": true, ".astro": true,
	".go": true, ".rs": true, ".java": true, ".kt": true, ".kts": true, ".scala": true,
	".groovy": true, ".rb": true, ".php": true, ".cs": true, ".fs": true, ".vb": true,
	".cpp": true, ".cc": true, ".cxx": true, ".c": true, ".h": true, ".hpp": true, ".hxx": true, ".hh": true,
	".swift": true, ".m": true, ".mm": true, ".lua": true, ".pl": true, ".r": true, ".R": true,
	".jl": true, ".dart": true, ".ex": true, ".exs": true, ".erl": true,
	".sh": true, ".bash": true, ".zsh": true, ".fish": true, ".ps1": true, ".psm1": true,
	".json": true, ".jsonc": true, ".json5": true, ".yml": true, ".yaml": true, ".toml": true,
	".ini": true, ".cfg": true, ".conf": true, ".config": true, ".xml": true, ".plist": true, ".properties": true,
	".mk": true, ".mak": true, ".cmake": true, ".gradle": true,
	".md": true, ".markdown": true, ".mdx": true, ".txt": true, ".rst": true, ".adoc": true, ".asciidoc": true,
	".html": true, ".htm": true, ".xhtml": true, ".css": true, ".scss": true, ".sass": true, ".less": true,
	".sql": true, ".graphql": true, ".gql": true, ".proto": true, ".prisma": true,
	".service": true, ".socket": true, ".timer": true, ".target": true, ".mount": true,
	".nginx": true, ".apache": true, ".htaccess": true,
	".nvmrc": true, ".python-version": true, ".ruby-version": true, ".node-version": true,
	".tool-versions": true, ".terraform-version": true,
}

var includeNames = map[string]bool{
	"Dockerfile": true, "Containerfile": true, "Makefile": true, "makefile": true, "GNUmakefile": true,
	"Justfile": true, "justfile": true, "Taskfile.yml": true, "Taskfile.yaml": true, "Vagrantfile": true,
	"Brewfile": true, "Gemfile": true, "Rakefile": true, "requirements.txt": true, "requirements-dev.txt": true,
	"pyproject.toml": true, "Pipfile": true, "setup.py": true, "setup.cfg": true, "tox.ini": true,
	".python-version": true, "package.json": true, "tsconfig.json": true, "jsconfig.json": true,
	".npmrc": true, ".nvmrc": true, ".yarnrc": true, "Cargo.toml": true, "go.mod": true,
	"LICENSE": true, "LICENCE": true, "COPYING": true, "NOTICE": true, "VERSION": true, "CHANGELOG": true,
	"AUTHORS": true, "MAINTAINERS": true, ".env.example": true, ".env.sample": true, ".env.template": true,
	".gitkeep": true, ".keep": true, ".pre-commit-config.yaml": true, ".editorconfig": true, ".prettierrc": true,
	".prettierignore": true, ".eslintrc": true, ".eslintrc.js": true, ".eslintrc.json": true, ".stylelintrc": true,
	"docker-compose.yml": true, "docker-compose.yaml": true, "compose.yml": true, "compose.yaml": true,
	"Procfile": true, "fly.toml": true, "netlify.toml": true, "vercel.json": true,
	".dockerignore": true, ".gitignore": true, ".gitattributes": true, ".gitmodules": true,
}

var excludeNames = map[string]bool{
	".env": true, ".env.local": true, ".env.production": true, ".env.prod": true,
	".env.dev": true, ".env.test": true, ".env.staging": true, ".env.development": true,
	"package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true, "bun.lockb": true,
	"poetry.lock": true, "Pipfile.lock": true, "uv.lock": true, "Cargo.lock": true,
	"composer.lock": true, "Gemfile.lock": true, "go.sum": true,
	"project_export.md": true, "project_export.txt": true,
	".DS_Store": true, "Thumbs.db": true, "desktop.ini": true,
}

var hiddenDirs = map[string]bool{
	".git": true, ".hg": true, ".svn": true, ".vscode": true, ".idea": true, ".fleet": true, ".nova": true,
	"__pycache__": true, "venv": true, ".venv": true, "env": true, ".env": true,
	".pytest_cache": true, ".mypy_cache": true, ".ruff_cache": true, ".tox": true, ".nox": true,
	".eggs": true, ".coverage": true, "build": true, "dist": true, "htmlcov": true, "coverage": true,
	".nyc_output": true, "node_modules": true, ".next": true, ".nuxt": true, ".svelte-kit": true,
	".astro": true, "out": true, ".turbo": true, ".parcel-cache": true, ".vercel": true, ".netlify": true,
	".cache": true, "playwright-report": true, "target": true, "vendor": true, ".gradle": true,
	"gradle": true, ".m2": true, "Pods": true,
}

var summaryDirs = map[string]bool{
	"data": true, "datasets": true, "raw": true, "processed": true, "models": true, "checkpoints": true,
	"weights": true, "assets": true, "media": true, "static": true, "backups": true, "exports": true,
	"reports": true, "logs": true, "notebooks": true, "mlruns": true, "mlartifacts": true,
	"faster-whisper-tiny": true, "piper-data": true, "tmp": true, "temp": true,
}

var excludeExt = map[string]bool{
	".pyc": true, ".pyo": true, ".pyd": true, ".so": true, ".dll": true, ".dylib": true, ".exe": true,
	".o": true, ".a": true, ".lib": true, ".obj": true, ".class": true, ".jar": true, ".war": true, ".ear": true,
	".dsym": true, ".map": true, ".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true,
	".webp": true, ".tiff": true, ".svg": true, ".ico": true, ".heic": true, ".raw": true,
	".mp4": true, ".mov": true, ".avi": true, ".webm": true, ".mkv": true, ".flv": true, ".wmv": true,
	".mp3": true, ".wav": true, ".flac": true, ".ogg": true, ".m4a": true, ".aac": true, ".wma": true,
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".eot": true,
	".pt": true, ".pth": true, ".onnx": true, ".pkl": true, ".joblib": true, ".h5": true, ".keras": true,
	".tflite": true, ".pb": true, ".ckpt": true, ".safetensors": true, ".gguf": true, ".ggml": true,
	".csv": true, ".tsv": true, ".parquet": true, ".feather": true, ".arff": true, ".npy": true, ".npz": true,
	".bin": true, ".dat": true, ".hdf5": true, ".zip": true, ".tar": true, ".gz": true, ".bz2": true,
	".xz": true, ".7z": true, ".rar": true, ".zst": true, ".pdf": true, ".docx": true, ".xlsx": true,
	".pptx": true, ".odt": true, ".ods": true, ".db": true, ".sqlite": true, ".sqlite3": true, ".mdb": true,
	".pem": true, ".key": true, ".crt": true, ".cer": true, ".p12": true, ".pfx": true, ".der": true,
	".lock": true, ".log": true, ".bak": true, ".tmp": true, ".swp": true, ".swo": true, ".orig": true, ".rej": true,
}

var langMap = map[string]string{
	".py": "python", ".pyi": "python", ".js": "javascript", ".jsx": "jsx", ".mjs": "javascript", ".cjs": "javascript",
	".ts": "typescript", ".tsx": "tsx", ".vue": "vue", ".svelte": "svelte", ".astro": "astro",
	".go": "go", ".rs": "rust", ".java": "java", ".kt": "kotlin", ".kts": "kotlin", ".scala": "scala",
	".groovy": "groovy", ".rb": "ruby", ".php": "php", ".cs": "csharp", ".fs": "fsharp", ".vb": "vbnet",
	".cpp": "cpp", ".cc": "cpp", ".cxx": "cpp", ".c": "c", ".h": "c", ".hpp": "cpp", ".hxx": "cpp", ".hh": "cpp",
	".swift": "swift", ".m": "objectivec", ".mm": "objectivec", ".lua": "lua", ".pl": "perl", ".r": "r", ".R": "r",
	".jl": "julia", ".dart": "dart", ".ex": "elixir", ".exs": "elixir", ".erl": "erlang",
	".sh": "bash", ".bash": "bash", ".zsh": "bash", ".fish": "fish", ".ps1": "powershell", ".psm1": "powershell",
	".json": "json", ".jsonc": "jsonc", ".json5": "json5", ".yml": "yaml", ".yaml": "yaml", ".toml": "toml",
	".ini": "ini", ".cfg": "ini", ".conf": "nginx", ".config": "xml", ".properties": "properties",
	".xml": "xml", ".plist": "xml", ".mk": "makefile", ".mak": "makefile", ".cmake": "cmake", ".gradle": "groovy",
	".md": "markdown", ".markdown": "markdown", ".mdx": "mdx", ".txt": "text", ".rst": "rst",
	".adoc": "asciidoc", ".asciidoc": "asciidoc", ".html": "html", ".htm": "html", ".xhtml": "html",
	".css": "css", ".scss": "scss", ".sass": "sass", ".less": "less", ".sql": "sql",
	".graphql": "graphql", ".gql": "graphql", ".proto": "protobuf", ".prisma": "prisma",
	".service": "ini", ".socket": "ini", ".timer": "ini", ".target": "ini", ".mount": "ini",
}

type sizedPath struct {
	Path string
	Size int64
}

type validFile struct {
	Path    string
	Content string
}

func printHelp() {
	fmt.Println("Usage: reta export [options]")
	fmt.Println("")
	fmt.Println("Export project source files into a single markdown file for AI context.")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -r, --root DIR        Project root directory (default: .)")
	fmt.Println("  -d, --dir DIR         Output directory (default: exports)")
	fmt.Println("  -o, --output FILE     Explicit output markdown path")
	fmt.Println("      --max-size SIZE   Max file size to include (default: 2M)")
	fmt.Println("      --include GLOB    Glob pattern to filter files (repeatable)")
	fmt.Println("      --warn-mb NUM     Warn if total exported MB exceeds NUM (default: 0.8)")
	fmt.Println("      --clean           Remove existing export file before regenerating")
	fmt.Println("      --clean-exports   Remove all files inside exports/ (with confirmation)")
	fmt.Println("  -h, --help            Show this help message")
}

func humanSize(n float64) string {
	for _, unit := range []string{"B", "KB", "MB", "GB"} {
		if n < 1024 {
			return fmt.Sprintf("%.1f %s", n, unit)
		}
		n /= 1024
	}
	return fmt.Sprintf("%.1f TB", n)
}

func formatCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
		if len(s) > rem {
			b.WriteByte(',')
		}
	}
	for i := rem; i < len(s); i += 3 {
		if i > rem {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func parseSize(s string) (int64, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return 0, nil
	}
	mult := map[byte]float64{
		'K': 1024,
		'M': 1024 * 1024,
		'G': 1024 * 1024 * 1024,
		'B': 1,
	}
	last := s[len(s)-1]
	if m, ok := mult[last]; ok {
		v, err := strconv.ParseFloat(s[:len(s)-1], 64)
		if err != nil {
			return 0, err
		}
		return int64(v * m), nil
	}
	return strconv.ParseInt(s, 10, 64)
}

// fnmatch converts a Python-style fnmatch glob (where * matches across '/') to a regexp.
func fnmatch(pattern, text string) bool {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteByte('.')
		case '[':
			j := i + 1
			if j < len(pattern) && (pattern[j] == '!' || pattern[j] == '^') {
				j++
			}
			if j < len(pattern) && pattern[j] == ']' {
				j++
			}
			for j < len(pattern) && pattern[j] != ']' {
				j++
			}
			if j >= len(pattern) {
				b.WriteString(`\[`)
			} else {
				stuff := pattern[i+1 : j]
				stuff = strings.ReplaceAll(stuff, `\`, `\\`)
				if strings.HasPrefix(stuff, "!") {
					stuff = "^" + stuff[1:]
				}
				b.WriteByte('[')
				b.WriteString(stuff)
				b.WriteByte(']')
				i = j
			}
		default:
			b.WriteString(regexp.QuoteMeta(pattern[i : i+1]))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return false
	}
	return re.MatchString(text)
}

func loadExportignore(root string) []string {
	data, err := os.ReadFile(filepath.Join(root, ".exportignore"))
	if err != nil {
		return nil
	}
	var rules []string
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimLeft(line, "/")
		if line != "" {
			rules = append(rules, line)
		}
	}
	return rules
}

func matchesIgnore(relPosix string, isDir bool, patterns []string) bool {
	relClean := strings.TrimRight(relPosix, "/")
	baseName := filepath.Base(filepath.FromSlash(relClean))
	for _, pat := range patterns {
		if strings.HasSuffix(pat, "/") {
			if !isDir {
				continue
			}
			base := strings.TrimRight(pat, "/")
			if relClean == base || strings.HasPrefix(relClean, base+"/") {
				return true
			}
		} else if fnmatch(pat, relClean) {
			return true
		} else if !strings.Contains(pat, "/") && fnmatch(pat, baseName) {
			return true
		}
	}
	return false
}

func samePath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func isExcludedFile(path string) bool {
	name := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(name))
	return excludeNames[name] || excludeExt[ext]
}

func shouldIncludeFile(path string) bool {
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	extLower := strings.ToLower(ext)

	if excludeNames[name] || excludeExt[extLower] {
		return false
	}
	if includeNames[name] {
		return true
	}
	if strings.HasPrefix(name, ".env.") {
		return name == ".env.example"
	}
	if includeExt[ext] || includeExt[extLower] {
		return true
	}
	if ext == "" {
		f, err := os.Open(path)
		if err != nil {
			return false
		}
		defer f.Close()
		buf := make([]byte, 2)
		n, _ := f.Read(buf)
		return n == 2 && buf[0] == '#' && buf[1] == '!'
	}
	return false
}

func dirTotalSize(root string) int64 {
	var total int64
	stack := []string{root}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		entries, err := os.ReadDir(cur)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			full := filepath.Join(cur, entry.Name())
			if entry.IsDir() {
				stack = append(stack, full)
			} else if entry.Type().IsRegular() {
				if info, err := entry.Info(); err == nil {
					total += info.Size()
				}
			}
		}
	}
	return total
}

func scanProject(root string, include, exportignore []string, skipAbs string) ([]string, []sizedPath, []sizedPath) {
	var files []string
	var skipped []sizedPath
	var summaries []sizedPath

	stack := []string{root}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		entries, err := os.ReadDir(cur)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			name := entry.Name()
			full := filepath.Join(cur, name)
			rel, err := filepath.Rel(root, full)
			if err != nil {
				continue
			}
			relPosix := filepath.ToSlash(rel)

			if entry.IsDir() {
				if name != ".github" {
					if summaryDirs[name] {
						if !matchesIgnore(relPosix, true, exportignore) {
							summaries = append(summaries, sizedPath{Path: full, Size: dirTotalSize(full)})
						}
						continue
					}
					if hiddenDirs[name] || strings.HasPrefix(name, ".") {
						continue
					}
				}
				if matchesIgnore(relPosix, true, exportignore) {
					continue
				}
				stack = append(stack, full)
				continue
			}

			if !entry.Type().IsRegular() {
				continue
			}
			absFull, _ := filepath.Abs(full)
			if resolved, err := filepath.EvalSymlinks(absFull); err == nil {
				absFull = resolved
			}
			if samePath(absFull, skipAbs) {
				continue
			}
			if matchesIgnore(relPosix, false, exportignore) {
				continue
			}
			if isExcludedFile(full) {
				if info, err := entry.Info(); err == nil {
					skipped = append(skipped, sizedPath{Path: full, Size: info.Size()})
				}
				continue
			}
			if len(include) > 0 {
				matched := false
				for _, pat := range include {
					normPat := strings.ReplaceAll(pat, `\`, "/")
					if fnmatch(normPat, relPosix) {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}
			if shouldIncludeFile(full) {
				files = append(files, full)
			}
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i]) < strings.ToLower(files[j])
	})
	sort.Slice(skipped, func(i, j int) bool {
		return strings.ToLower(skipped[i].Path) < strings.ToLower(skipped[j].Path)
	})
	sort.Slice(summaries, func(i, j int) bool {
		return strings.ToLower(summaries[i].Path) < strings.ToLower(summaries[j].Path)
	})
	return files, skipped, summaries
}

func resolveAbs(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = filepath.Clean(p)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

func resolveOutputFromDir(root, exportDir string) string {
	val := strings.TrimSpace(exportDir)
	if val == "" {
		val = "exports"
	}
	if val == "." {
		return resolveAbs(filepath.Join(root, "project_export.md"))
	}
	if strings.HasPrefix(val, "~/") || strings.HasPrefix(val, `~\`) || val == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			val = filepath.Join(home, strings.TrimPrefix(val, "~"))
		}
	}
	if filepath.IsAbs(val) {
		return resolveAbs(filepath.Join(val, "project_export.md"))
	}
	return resolveAbs(filepath.Join(root, val, "project_export.md"))
}

func isDangerousPath(p string) bool {
	clean := filepath.Clean(p)
	resolved := resolveAbs(p)
	if home, err := os.UserHomeDir(); err == nil {
		homeClean := filepath.Clean(home)
		homeResolved := resolveAbs(home)
		if runtime.GOOS == "windows" {
			if strings.EqualFold(clean, homeClean) || strings.EqualFold(resolved, homeResolved) {
				return true
			}
		} else if clean == homeClean || resolved == homeResolved {
			return true
		}
	}
	if runtime.GOOS == "windows" {
		vol := filepath.VolumeName(clean)
		if clean == vol+`\` || clean == vol || clean == `\` {
			return true
		}
		return false
	}
	dangerous := map[string]bool{
		"/": true, "/home": true, "/usr": true, "/etc": true, "/var": true,
		"/boot": true, "/sys": true, "/proc": true, "/root": true,
	}
	if runtime.GOOS == "darwin" {
		dangerous["/Users"] = true
		dangerous["/System"] = true
		dangerous["/Applications"] = true
	}
	return dangerous[clean]
}

func dirFileStats(root string) (int, int64) {
	count := 0
	var totalBytes int64
	stack := []string{root}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		entries, err := os.ReadDir(cur)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			full := filepath.Join(cur, entry.Name())
			if entry.IsDir() {
				stack = append(stack, full)
			} else if entry.Type().IsRegular() {
				count++
				if info, err := entry.Info(); err == nil {
					totalBytes += info.Size()
				}
			}
		}
	}
	return count, totalBytes
}

func cleanExportsDir(root, targetDir string) int {
	targetResolved := resolveAbs(targetDir)
	rootResolved := resolveAbs(root)

	if filepath.Base(rootResolved) == "exports" {
		fmt.Fprintf(os.Stderr, "Refusing to bulk-delete: project root directory '%s' is named 'exports'.\n", rootResolved)
		return 1
	}
	if targetResolved == rootResolved {
		fmt.Fprintln(os.Stderr, "Refusing to bulk-delete project root directory. Use 'make clean-export' to remove only project_export.md.")
		return 1
	}
	if filepath.Base(targetResolved) != "exports" {
		fmt.Fprintf(os.Stderr, "Refusing to bulk-delete '%s': only directories named 'exports' can be bulk-cleaned.\n", targetResolved)
		return 1
	}
	if isDangerousPath(targetResolved) {
		fmt.Fprintf(os.Stderr, "Refusing to clean protected path: %s\n", targetResolved)
		return 1
	}
	info, err := os.Stat(targetResolved)
	if err != nil || !info.IsDir() {
		fmt.Printf("Nothing to clean (%s does not exist).\n", targetResolved)
		return 0
	}
	children, err := os.ReadDir(targetResolved)
	if err != nil || len(children) == 0 {
		fmt.Printf("Nothing to clean (%s is already empty).\n", targetResolved)
		return 0
	}

	fileCount, totalBytes := dirFileStats(targetResolved)
	topDirs := 0
	for _, c := range children {
		if c.IsDir() {
			topDirs++
		}
	}
	detail := fmt.Sprintf("%d file(s)", fileCount)
	if topDirs > 0 && fileCount == 0 {
		detail = fmt.Sprintf("%d empty folder(s)", topDirs)
	}

	fmt.Printf("About to delete %s (%s) inside: %s/\n", detail, humanSize(float64(totalBytes)), targetResolved)
	if !ui.ConfirmYes(os.Stdout, "Continue? [y/N] ") {
		fmt.Println("Cancelled.")
		return 0
	}

	for _, item := range children {
		itemPath := filepath.Join(targetResolved, item.Name())
		if err := os.RemoveAll(itemPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing %s: %v\n", itemPath, err)
			return 1
		}
	}
	fmt.Printf("Cleaned %s (%s) from %s/.\n", detail, humanSize(float64(totalBytes)), targetResolved)
	return 0
}

func langFor(path string) string {
	name := filepath.Base(path)
	switch name {
	case "Dockerfile", "Containerfile":
		return "dockerfile"
	case "Makefile", "makefile", "GNUmakefile", "Justfile", "justfile":
		return "makefile"
	case ".env.example", ".env.sample", ".env.template", ".gitignore", ".dockerignore",
		".editorconfig", ".nvmrc", ".python-version", ".ruby-version", ".node-version", ".tool-versions":
		return "bash"
	case "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml":
		return "yaml"
	}
	ext := filepath.Ext(name)
	if v, ok := langMap[ext]; ok {
		return v
	}
	return langMap[strings.ToLower(ext)]
}

func decodeText(raw []byte) string {
	if utf8.Valid(raw) {
		return string(raw)
	}
	// Fallback to Latin-1 byte-to-rune mapping when input is not UTF-8.
	runes := make([]rune, len(raw))
	for i, b := range raw {
		runes[i] = rune(b)
	}
	return string(runes)
}

type tokenEstimate struct {
	ChatGPT int64
	Claude  int64
	Gemini  int64
}

func isEmojiModifier(r rune) bool {
	return r == 0x200D ||
		(r >= 0xFE00 && r <= 0xFE0F) ||
		(r >= 0x1F3FB && r <= 0x1F3FF)
}

func classifyWordRune(s string, idx int) (ok bool, size int, isUnderscore, isUpper, isLatinExt, isOtherScript bool) {
	b := s[idx]
	if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') {
		return true, 1, false, b >= 'A' && b <= 'Z', false, false
	}
	if b == '_' {
		return true, 1, true, false, false, false
	}
	if b >= 0x80 {
		r, sz := utf8.DecodeRuneInString(s[idx:])
		if sz <= 0 {
			sz = 1
		}
		if unicode.IsLetter(r) {
			if r <= 0x024F {
				return true, sz, false, unicode.IsUpper(r), true, false
			}
			return true, sz, false, unicode.IsUpper(r), false, true
		}
	}
	return false, 0, false, false, false, false
}

func segmentCost(segLen, latinExt, otherScript int, isFirstSeg, afterCamel bool) (dg, dm, dc float64) {
	if segLen == 0 {
		return 0, 0, 0
	}
	latinLen := segLen - otherScript
	if latinLen > 0 {
		base := 1.0
		if !isFirstSeg {
			base = 0.85
		}
		if latinLen > 5 {
			base += float64(latinLen-5) / 4.8
		}
		dg += base + float64(latinExt)*0.12
		if isFirstSeg {
			dm += base*1.04 + float64(latinExt)*0.08
			dc += base*1.12 + float64(latinExt)*0.32
		} else if afterCamel {
			dm += (base+0.07)*1.04 + float64(latinExt)*0.08
			dc += (base+0.25)*1.10 + float64(latinExt)*0.32
		} else {
			dm += base*1.05 + float64(latinExt)*0.08
			dc += (base+0.12)*1.12 + float64(latinExt)*0.32
		}
	}
	if otherScript > 0 {
		dg += float64(otherScript) * 0.85
		dm += float64(otherScript) * 0.75
		dc += float64(otherScript) * 1.15
	}
	return dg, dm, dc
}

func estimateTokens(text string) tokenEstimate {
	if len(text) == 0 {
		return tokenEstimate{}
	}

	var gpt, claude, gemini float64
	n := len(text)
	for i := 0; i < n; {
		b := text[i]

		// Whitespace and newlines
		if b == '\n' || b == '\r' || b == ' ' || b == '\t' {
			start := i
			newlines := 0
			for i < n && (text[i] == ' ' || text[i] == '\t' || text[i] == '\n' || text[i] == '\r') {
				if text[i] == '\n' {
					newlines++
				}
				i++
			}
			wsLen := i - start
			if newlines == 0 && wsLen == 1 && b == ' ' {
				if i < n {
					if ok, _, isUnder, _, _, _ := classifyWordRune(text, i); ok && !isUnder {
						continue
					}
					if text[i] >= '0' && text[i] <= '9' {
						gpt += 1.0
						gemini += 1.0
						claude += 1.0
						continue
					}
					gpt += 0.25
					gemini += 0.55
					claude += 0.85
					continue
				}
				gpt += 1.0
				gemini += 1.0
				claude += 1.0
				continue
			}

			extraWS := wsLen - newlines
			gpt += float64(newlines/2)*1.0 + float64(newlines%2)*0.35 + float64(extraWS)/4.0
			gemini += float64(newlines)*1.0 + float64(extraWS)/2.4
			claude += float64(newlines)*1.0 + float64(extraWS)/2.4
			continue
		}

		// Digit runs: OpenAI groups up to 3 digits; Gemini and Claude tokenize 1 digit per token.
		if b >= '0' && b <= '9' {
			start := i
			for i < n && text[i] >= '0' && text[i] <= '9' {
				i++
			}
			dLen := i - start
			gpt += float64((dLen + 2) / 3)
			gemini += float64(dLen)
			claude += float64(dLen)
			continue
		}

		// Words and code identifiers (snake_case, camelCase, Latin-Extended, other scripts)
		if ok, sz, firstUnder, firstUpper, firstLatinExt, firstOther := classifyWordRune(text, i); ok {
			segLen := 0
			segLatinExt := 0
			segOther := 0
			letterSegs := 0
			underscores := 0
			afterCamel := false
			prevLower := false

			if firstUnder {
				underscores++
			} else {
				segLen = 1
				if firstLatinExt {
					segLatinExt = 1
				}
				if firstOther {
					segOther = 1
				}
				prevLower = !firstUpper
			}
			i += sz

			for i < n {
				okNext, nextSz, nextUnder, nextUpper, nextLatinExt, nextOther := classifyWordRune(text, i)
				if !okNext {
					break
				}
				if nextUnder {
					if segLen > 0 {
						dg, dm, dc := segmentCost(segLen, segLatinExt, segOther, letterSegs == 0, afterCamel)
						gpt += dg
						gemini += dm
						claude += dc
						letterSegs++
						segLen, segLatinExt, segOther = 0, 0, 0
						afterCamel = false
					}
					underscores++
					gpt += 0.35
					gemini += 0.70
					claude += 0.90
					prevLower = false
					i += nextSz
					continue
				}
				if prevLower && nextUpper && segLen > 0 {
					dg, dm, dc := segmentCost(segLen, segLatinExt, segOther, letterSegs == 0, afterCamel)
					gpt += dg
					gemini += dm
					claude += dc
					letterSegs++
					segLen, segLatinExt, segOther = 0, 0, 0
					afterCamel = true
				}
				segLen++
				if nextLatinExt {
					segLatinExt++
				}
				if nextOther {
					segOther++
				}
				prevLower = !nextUpper
				i += nextSz
			}

			if segLen > 0 {
				dg, dm, dc := segmentCost(segLen, segLatinExt, segOther, letterSegs == 0, afterCamel)
				gpt += dg
				gemini += dm
				claude += dc
				letterSegs++
			}
			if letterSegs == 0 && underscores > 0 {
				gpt += float64(underscores) * 0.90
				gemini += float64(underscores) * 0.95
				claude += float64(underscores) * 1.00
			} else if firstUnder && letterSegs > 0 {
				gpt += 0.45
				gemini += 0.75
				claude += 0.90
			}
			continue
		}

		// Non-ASCII symbols and emoji clusters (ZWJ, variation selectors, Fitzpatrick skin tones)
		if b >= 0x80 {
			r, size := utf8.DecodeRuneInString(text[i:])
			if size <= 0 {
				size = 1
			}
			i += size
			if isEmojiModifier(r) {
				gpt += 0.5
				gemini += 0.5
				claude += 0.5
				continue
			}
			if r > 0x024F {
				gpt += 1.5
				gemini += 1.4
				claude += 1.8
				for i < n {
					nr, nsz := utf8.DecodeRuneInString(text[i:])
					if nsz <= 0 {
						break
					}
					if isEmojiModifier(nr) {
						i += nsz
						gpt += 0.5
						gemini += 0.5
						claude += 0.5
						if nr == 0x200D && i < n {
							jr, jsz := utf8.DecodeRuneInString(text[i:])
							if jsz > 0 && jr >= 0x80 && !unicode.IsSpace(jr) {
								i += jsz
								gpt += 0.5
								gemini += 0.5
								claude += 0.5
							}
						}
						continue
					}
					break
				}
				continue
			}
			gpt += 1.0
			gemini += 1.0
			claude += 1.15
			continue
		}

		// ASCII punctuation / operator runs
		start := i
		for i < n {
			c := text[i]
			if c >= 0x80 || c == ' ' || c == '\t' || c == '\n' || c == '\r' ||
				(c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
				break
			}
			i++
		}
		pLen := i - start
		if pLen == 1 {
			gpt += 0.85
			gemini += 0.95
			claude += 1.05
		} else {
			gpt += 0.45 + float64(pLen)*0.35
			gemini += 0.60 + float64(pLen)*0.55
			claude += 0.75 + float64(pLen)*0.85
		}
	}

	return tokenEstimate{
		ChatGPT: int64(gpt + 0.5),
		Claude:  int64(claude + 0.5),
		Gemini:  int64(gemini + 0.5),
	}
}

func Run(args []string) int {
	rootArg := "."
	dirArg := "exports"
	outputArg := ""
	maxSizeStr := "2M"
	var includeArgs []string
	warnMB := 0.8
	cleanFlag := false
	cleanExportsFlag := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-h", "--help":
			printHelp()
			return 0
		case "--clean":
			cleanFlag = true
		case "--clean-exports":
			cleanExportsFlag = true
		case "-r", "--root":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: %s requires an argument.\n", arg)
				return 2
			}
			i++
			rootArg = args[i]
		case "-d", "--dir":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: %s requires an argument.\n", arg)
				return 2
			}
			i++
			dirArg = args[i]
		case "-o", "--output":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: %s requires an argument.\n", arg)
				return 2
			}
			i++
			outputArg = args[i]
		case "--max-size":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --max-size requires an argument.")
				return 2
			}
			i++
			maxSizeStr = args[i]
		case "--include":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --include requires an argument.")
				return 2
			}
			i++
			includeArgs = append(includeArgs, args[i])
		case "--warn-mb":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --warn-mb requires an argument.")
				return 2
			}
			i++
			v, err := strconv.ParseFloat(args[i], 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid --warn-mb value: %s\n", args[i])
				return 2
			}
			warnMB = v
		default:
			if strings.HasPrefix(arg, "--root=") {
				rootArg = strings.TrimPrefix(arg, "--root=")
			} else if strings.HasPrefix(arg, "--dir=") {
				dirArg = strings.TrimPrefix(arg, "--dir=")
			} else if strings.HasPrefix(arg, "--output=") {
				outputArg = strings.TrimPrefix(arg, "--output=")
			} else if strings.HasPrefix(arg, "--max-size=") {
				maxSizeStr = strings.TrimPrefix(arg, "--max-size=")
			} else if strings.HasPrefix(arg, "--include=") {
				includeArgs = append(includeArgs, strings.TrimPrefix(arg, "--include="))
			} else if strings.HasPrefix(arg, "--warn-mb=") {
				v, err := strconv.ParseFloat(strings.TrimPrefix(arg, "--warn-mb="), 64)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: invalid --warn-mb value.\n")
					return 2
				}
				warnMB = v
			} else {
				fmt.Fprintf(os.Stderr, "Error: unknown option: %s\n", arg)
				return 2
			}
		}
	}

	root := resolveAbs(rootArg)
	var output string
	if outputArg != "" {
		output = resolveAbs(outputArg)
	} else {
		output = resolveOutputFromDir(root, dirArg)
	}
	maxSize, err := parseSize(maxSizeStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid --max-size '%s': %v\n", maxSizeStr, err)
		return 2
	}

	if cleanExportsFlag {
		return cleanExportsDir(root, filepath.Dir(output))
	}

	if isDangerousPath(root) {
		fmt.Printf("Refusing to scan protected path: %s\n", root)
		return 1
	}
	if _, err := os.Stat(root); err != nil {
		fmt.Printf("Root not found: %s\n", root)
		return 1
	}

	if cleanFlag {
		if info, err := os.Stat(output); err == nil && info.Mode().IsRegular() {
			if err := os.Remove(output); err == nil {
				fmt.Printf("Removed old export: %s\n", output)
			}
		}
	}

	exportignore := loadExportignore(root)

	fmt.Printf("Scanning: %s\n", root)
	fmt.Printf("Output:   %s\n", output)
	fmt.Printf("Max size: %s\n", humanSize(float64(maxSize)))
	if len(exportignore) > 0 {
		fmt.Printf("Ignore:   %d rules from .exportignore\n", len(exportignore))
	}
	fmt.Println()

	files, skipped, summaries := scanProject(root, includeArgs, exportignore, output)
	if len(files) == 0 {
		fmt.Println("No matching files.")
		return 1
	}

	fmt.Printf("Found %d files (%d skipped, %d dirs summarized). Writing...\n\n", len(files), len(skipped), len(summaries))

	var valid []validFile
	var oversized []sizedPath
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(root, f)
		if info.Size() > maxSize {
			oversized = append(oversized, sizedPath{Path: f, Size: info.Size()})
			fmt.Printf("  skipped (size): %s\n", filepath.ToSlash(rel))
			continue
		}
		raw, err := os.ReadFile(f)
		if err != nil {
			fmt.Printf("  skipped (read): %s\n", filepath.ToSlash(rel))
			continue
		}
		valid = append(valid, validFile{Path: f, Content: decodeText(raw)})
	}

	skipped = append(skipped, oversized...)
	sort.Slice(skipped, func(i, j int) bool {
		return strings.ToLower(skipped[i].Path) < strings.ToLower(skipped[j].Path)
	})

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output dir: %v\n", err)
		return 1
	}

	var out bytes.Buffer
	out.WriteString(fmt.Sprintf("# Project Export — %s\n\n", filepath.Base(root)))
	out.WriteString(fmt.Sprintf("**Root:** `%s`\n\n", root))
	out.WriteString(fmt.Sprintf("**Files:** %d\n\n", len(files)))
	out.WriteString("---\n\n## Contents\n\n")

	for _, vf := range valid {
		rel, _ := filepath.Rel(root, vf.Path)
		out.WriteString(fmt.Sprintf("- `%s`\n", filepath.ToSlash(rel)))
	}
	for _, sp := range skipped {
		rel, _ := filepath.Rel(root, sp.Path)
		out.WriteString(fmt.Sprintf("- `%s` (%s, content omitted)\n", filepath.ToSlash(rel), humanSize(float64(sp.Size))))
	}
	for _, sm := range summaries {
		rel, _ := filepath.Rel(root, sm.Path)
		out.WriteString(fmt.Sprintf("- `%s/` (%s, content omitted)\n", filepath.ToSlash(rel), humanSize(float64(sm.Size))))
	}
	out.WriteString("\n---\n\n")

	var total int64
	written := 0
	for _, vf := range valid {
		rel, _ := filepath.Rel(root, vf.Path)
		size := int64(len(vf.Content))
		total += size
		written++

		out.WriteString(fmt.Sprintf("## `%s`\n\n", filepath.ToSlash(rel)))
		out.WriteString(fmt.Sprintf("Size: %s bytes\n\n", formatCommas(size)))
		if strings.TrimSpace(vf.Content) == "" {
			out.WriteString("_(empty file)_\n\n")
		}
		fence := "```"
		for strings.Contains(vf.Content, fence) {
			fence += "`"
		}
		out.WriteString(fmt.Sprintf("%s%s\n", fence, langFor(vf.Path)))
		out.WriteString(vf.Content)
		if !strings.HasSuffix(vf.Content, "\n") {
			out.WriteByte('\n')
		}
		out.WriteString(fmt.Sprintf("%s\n\n---\n\n", fence))
	}

	for _, sp := range skipped {
		rel, _ := filepath.Rel(root, sp.Path)
		out.WriteString(fmt.Sprintf("## `%s`\n\n", filepath.ToSlash(rel)))
		out.WriteString(fmt.Sprintf("Size: %s bytes\n\n", formatCommas(sp.Size)))
		out.WriteString("_(Content was not included by export-context)_\n\n---\n\n")
	}
	for _, sm := range summaries {
		rel, _ := filepath.Rel(root, sm.Path)
		out.WriteString(fmt.Sprintf("## `%s/`\n\n", filepath.ToSlash(rel)))
		out.WriteString(fmt.Sprintf("Size: %s bytes\n\n", formatCommas(sm.Size)))
		out.WriteString("_(Directory content was not included by export-context)_\n\n---\n\n")
	}

	if err := os.WriteFile(output, out.Bytes(), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", output, err)
		return 1
	}

	sizeMB := float64(total) / (1024 * 1024)
	est := estimateTokens(out.String())
	fmt.Println("\nDone.")
	fmt.Printf("Files:  %d included, %d skipped, %d dirs summarized, Total: %s\n",
		written, len(skipped), len(summaries), humanSize(float64(total)))
	fmt.Printf("Tokens: ~%s ChatGPT (o200k) | ~%s Gemini | ~%s Claude 4.7+ (est.)\n",
		formatCommas(est.ChatGPT), formatCommas(est.Gemini), formatCommas(est.Claude))
	fmt.Printf("Output: %s\n", output)

	if sizeMB > warnMB {
		fmt.Printf("\nWarning: %.2f MB exceeds %.1f MB threshold.\n", sizeMB, warnMB)
		fmt.Println("Add .exportignore rules or narrow with --include.")
	}
	return 0
}
