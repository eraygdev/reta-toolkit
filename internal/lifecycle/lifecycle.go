package lifecycle

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	reta "github.com/eraygdev/reta-toolkit"
	"github.com/eraygdev/reta-toolkit/internal/doctor"
	"github.com/eraygdev/reta-toolkit/internal/ui"
)

const latestReleaseAPI = "https://api.github.com/repos/eraygdev/reta-toolkit/releases/latest"

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func expectedAssetName() string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("reta-%s-%s%s", runtime.GOOS, runtime.GOARCH, ext)
}

func replaceBinary(exePath string, newBytes []byte) error {
	newPath := exePath + ".new"
	if err := os.WriteFile(newPath, newBytes, 0o755); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		oldPath := exePath + ".old"
		_ = os.Remove(oldPath)
		if err := os.Rename(exePath, oldPath); err != nil {
			_ = os.Remove(newPath)
			return err
		}
		if err := os.Rename(newPath, exePath); err != nil {
			_ = os.Rename(oldPath, exePath)
			return err
		}
		_ = os.Remove(oldPath)
		return nil
	}
	if err := os.Rename(newPath, exePath); err != nil {
		_ = os.Remove(newPath)
		return err
	}
	return nil
}

func updateFromGitRepo(retaRoot string) int {
	c := ui.Colors()
	oldVer := strings.TrimSpace(reta.VersionRaw)
	headCmd := exec.Command("git", "-C", retaRoot, "rev-parse", "HEAD")
	oldHeadBytes, _ := headCmd.Output()
	oldHead := strings.TrimSpace(string(oldHeadBytes))

	fmt.Println("Pulling from origin...")
	pullCmd := exec.Command("git", "-C", retaRoot, "pull", "--rebase")
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		fmt.Println("")
		stOut, _ := exec.Command("git", "-C", retaRoot, "status", "--porcelain").Output()
		if strings.TrimSpace(string(stOut)) != "" {
			fmt.Fprintf(os.Stderr, "  %sERR%s   You have uncommitted changes in ~/.reta.\n", c.Red, c.Reset)
			fmt.Fprintln(os.Stderr, "         git pull --rebase needs a clean working tree.")
		} else {
			fmt.Fprintf(os.Stderr, "  %sERR%s   git pull --rebase failed. Run manually: cd ~/.reta && git pull --rebase\n", c.Red, c.Reset)
		}
		return 1
	}

	newHeadBytes, _ := exec.Command("git", "-C", retaRoot, "rev-parse", "HEAD").Output()
	newHead := strings.TrimSpace(string(newHeadBytes))
	newVerBytes, err := os.ReadFile(filepath.Join(retaRoot, "VERSION"))
	newVer := oldVer
	if err == nil {
		newVer = strings.TrimSpace(string(newVerBytes))
	}

	fmt.Println("")
	if oldHead != "" && oldHead == newHead {
		fmt.Printf("  %sOK%s    Already up to date (%s).\n", c.Green, c.Reset, newVer)
	} else {
		fmt.Println("Changes:")
		diffCmd := exec.Command("git", "-C", retaRoot, "diff", "--stat", oldHead, newHead)
		if diffOut, err := diffCmd.Output(); err == nil {
			for _, ln := range strings.Split(strings.TrimSpace(string(diffOut)), "\n") {
				if ln != "" {
					fmt.Printf("  %s\n", ln)
				}
			}
		}
		fmt.Println("")
		if oldVer != newVer {
			fmt.Printf("  %sOK%s    Version: %s -> %s\n", c.Green, c.Reset, oldVer, newVer)
		} else {
			ui.PrintDim(os.Stdout, fmt.Sprintf("Version: %s (unchanged)", newVer))
		}
	}

	if _, err := exec.LookPath("go"); err == nil {
		if exePath, exeErr := os.Executable(); exeErr == nil {
			if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
				exePath = resolved
			}
			fmt.Println("\nRebuilding Go binary...")
			tmpOut := exePath + ".new"
			buildCmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", tmpOut, "./cmd/reta")
			buildCmd.Dir = retaRoot
			buildCmd.Env = append(os.Environ(), "CGO_ENABLED=0")
			buildCmd.Stdout = os.Stdout
			buildCmd.Stderr = os.Stderr
			if err := buildCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "  %sERR%s   Failed to rebuild binary: %v\n", c.Red, c.Reset, err)
				return 1
			}
			data, rErr := os.ReadFile(tmpOut)
			_ = os.Remove(tmpOut)
			if rErr != nil {
				fmt.Fprintf(os.Stderr, "  %sERR%s   Failed to read built binary: %v\n", c.Red, c.Reset, rErr)
				return 1
			}
			if repErr := replaceBinary(exePath, data); repErr != nil {
				fmt.Fprintf(os.Stderr, "  %sERR%s   Failed to replace binary: %v\n", c.Red, c.Reset, repErr)
				return 1
			}
			fmt.Printf("  %sOK%s    Updated binary at %s\n", c.Green, c.Reset, exePath)
		}
	} else {
		fmt.Printf("\n  %sWARN%s  'go' not found on PATH; source updated, but binary was not rebuilt.\n", c.Yellow, c.Reset)
		fmt.Println("         Run 'reta update --binary' to update the binary.")
	}
	return 0
}

func updateFromRelease(retaRoot string) int {
	c := ui.Colors()
	curVer := strings.TrimPrefix(strings.TrimSpace(reta.VersionRaw), "v")
	client := &http.Client{Timeout: 20 * time.Second}

	req, _ := http.NewRequest("GET", latestReleaseAPI, nil)
	req.Header.Set("User-Agent", "reta-cli")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %sERR%s   Failed to check latest release: %v\n", c.Red, c.Reset, err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Fprintf(os.Stderr, "  %sWARN%s  No published GitHub Releases found yet.\n", c.Yellow, c.Reset)
		return 1
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "  %sERR%s   GitHub API returned status %d\n", c.Red, c.Reset, resp.StatusCode)
		return 1
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		fmt.Fprintf(os.Stderr, "  %sERR%s   Invalid release metadata: %v\n", c.Red, c.Reset, err)
		return 1
	}

	latestVer := strings.TrimPrefix(strings.TrimSpace(rel.TagName), "v")
	if latestVer == "" || latestVer == curVer {
		fmt.Printf("  %sOK%s    Already up to date (%s).\n", c.Green, c.Reset, curVer)
		_ = doctor.EnsureBaseMK(retaRoot)
		return 0
	}

	wantAsset := expectedAssetName()
	downloadURL := ""
	for _, a := range rel.Assets {
		if a.Name == wantAsset {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		fmt.Fprintf(os.Stderr, "  %sERR%s   Release %s has no binary for %s.\n", c.Red, c.Reset, rel.TagName, wantAsset)
		return 1
	}

	fmt.Printf("Downloading %s (%s)...\n", rel.TagName, wantAsset)
	dlResp, err := client.Get(downloadURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %sERR%s   Failed to download binary.\n", c.Red, c.Reset)
		return 1
	}
	defer dlResp.Body.Close()
	if dlResp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "  %sERR%s   Failed to download binary (status %d).\n", c.Red, c.Reset, dlResp.StatusCode)
		return 1
	}

	binBytes, err := io.ReadAll(dlResp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %sERR%s   Download interrupted: %v\n", c.Red, c.Reset, err)
		return 1
	}

	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %sERR%s   Cannot determine executable path: %v\n", c.Red, c.Reset, err)
		return 1
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}
	if err := replaceBinary(exePath, binBytes); err != nil {
		fmt.Fprintf(os.Stderr, "  %sERR%s   Failed to replace %s: %v\n", c.Red, c.Reset, exePath, err)
		return 1
	}

	_ = doctor.EnsureBaseMK(retaRoot)
	fmt.Printf("  %sOK%s    Updated reta: %s -> %s\n", c.Green, c.Reset, curVer, latestVer)
	return 0
}

func RunUpdate(args []string) int {
	forceBinary := false
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			fmt.Println("Usage: reta update [--binary]")
			fmt.Println("")
			fmt.Println("Pull latest changes in ~/.reta (git install) or update from GitHub Releases.")
			return 0
		case "--binary":
			forceBinary = true
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown option: %s\n", arg)
			return 1
		}
	}

	home, _ := os.UserHomeDir()
	retaRoot := filepath.Join(home, ".reta")
	if !forceBinary {
		if info, err := os.Stat(filepath.Join(retaRoot, ".git")); err == nil && info.IsDir() {
			return updateFromGitRepo(retaRoot)
		}
	}
	return updateFromRelease(retaRoot)
}

func RunUninstall(args []string) int {
	keepSource := false
	assumeYes := false

	for _, arg := range args {
		switch arg {
		case "--keep", "--keep-source":
			keepSource = true
		case "-y", "--yes":
			assumeYes = true
		case "-h", "--help":
			fmt.Println("Usage: reta uninstall [--keep] [-y]")
			fmt.Println("")
			fmt.Println("  --keep       Remove only the PATH binary/symlink; keep ~/.reta")
			fmt.Println("  -y, --yes    Skip confirmation prompts")
			return 0
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown option: %s\n", arg)
			return 1
		}
	}

	c := ui.Colors()
	home, _ := os.UserHomeDir()
	retaRoot := filepath.Join(home, ".reta")

	fmt.Println("reta uninstall")
	fmt.Println("")

	if runtime.GOOS == "windows" {
		binPath := filepath.Join(os.Getenv("LOCALAPPDATA"), "reta", "bin", "reta.exe")
		if _, err := os.Stat(binPath); err == nil {
			if err := os.Remove(binPath); err != nil {
				oldPath := binPath + ".old"
				_ = os.Remove(oldPath)
				if rErr := os.Rename(binPath, oldPath); rErr == nil {
					fmt.Printf("  %sWARN%s  %s is in use; renamed to %s (delete after closing terminal)\n", c.Yellow, c.Reset, binPath, oldPath)
				} else {
					fmt.Printf("  %sWARN%s  Could not remove %s (%v); delete it manually\n", c.Yellow, c.Reset, binPath, err)
				}
			} else {
				fmt.Printf("  %sOK%s    removed %s\n", c.Green, c.Reset, binPath)
			}
		}
	} else {
		targetBin := filepath.Join(home, ".local", "bin", "reta")
		if fi, err := os.Lstat(targetBin); err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				t, _ := filepath.EvalSymlinks(targetBin)
				expected, _ := filepath.EvalSymlinks(filepath.Join(retaRoot, "bin", "reta"))
				if t == expected || strings.HasPrefix(t, retaRoot) {
					_ = os.Remove(targetBin)
					fmt.Printf("  %sOK%s    removed %s\n", c.Green, c.Reset, targetBin)
				} else {
					fmt.Printf("  %sWARN%s  %s points to %s, leaving it alone\n", c.Yellow, c.Reset, targetBin, t)
				}
			} else if fi.Mode().IsRegular() {
				_ = os.Remove(targetBin)
				fmt.Printf("  %sOK%s    removed %s\n", c.Green, c.Reset, targetBin)
			}
		} else {
			ui.PrintDim(os.Stdout, fmt.Sprintf("%s not found (nothing to remove)", targetBin))
		}

		for _, comp := range []string{
			filepath.Join(home, ".local", "share", "bash-completion", "completions", "reta"),
			filepath.Join(home, ".zfunc", "_reta"),
		} {
			if _, err := os.Lstat(comp); err == nil {
				_ = os.Remove(comp)
				fmt.Printf("  %sOK%s    removed %s\n", c.Green, c.Reset, comp)
			}
		}
	}

	if keepSource {
		fmt.Printf("  %sOK%s    kept %s (--keep)\n", c.Green, c.Reset, retaRoot)
		return 0
	}

	_ = os.RemoveAll(filepath.Join(home, ".cache", "reta"))

	fi, err := os.Lstat(retaRoot)
	if err != nil {
		ui.PrintDim(os.Stdout, fmt.Sprintf("%s not found", retaRoot))
		return 0
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		target, _ := os.Readlink(retaRoot)
		fmt.Printf("  %sOK%s    %s is a symlink -> %s\n", c.Green, c.Reset, retaRoot, target)
		_ = os.Remove(retaRoot)
		fmt.Printf("  %sOK%s    removed symlink (source directory kept intact)\n", c.Green, c.Reset)
		return 0
	}

	_, hasBin := os.Stat(filepath.Join(retaRoot, "bin", "reta"))
	_, hasBaseMK := os.Stat(filepath.Join(retaRoot, "lib", "base.mk"))
	if hasBin != nil && hasBaseMK != nil {
		fmt.Fprintf(os.Stderr, "  %sERR%s   %s does not look like a reta installation.\n", c.Red, c.Reset, retaRoot)
		return 1
	}

	if !assumeYes {
		fmt.Printf("\nAbout to delete: %s\n", retaRoot)
		if !ui.ConfirmYes(os.Stdout, "Continue? [y/N] ") {
			ui.PrintDim(os.Stdout, "Cancelled.")
			return 0
		}
	}

	if err := os.RemoveAll(retaRoot); err != nil {
		fmt.Fprintf(os.Stderr, "  %sERR%s   %v\n", c.Red, c.Reset, err)
		return 1
	}
	fmt.Printf("  %sOK%s    removed %s\n", c.Green, c.Reset, retaRoot)
	return 0
}
