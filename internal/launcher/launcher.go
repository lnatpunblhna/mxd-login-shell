// Package launcher starts the stock CMS079 MapleStory.exe after LoginBridge select.
// Windows-first: args are "<ip> <port>" matching the classic private-server launch line.
package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const EnvClient = "MXD_CLIENT"

// ResolvePath picks -client flag, else MXD_CLIENT, else empty.
func ResolvePath(flagPath string) string {
	p := strings.TrimSpace(flagPath)
	if p != "" {
		return p
	}
	return strings.TrimSpace(os.Getenv(EnvClient))
}

// Start launches MapleStory.exe (or Wine-compatible path) with "<ip> <port>".
// Working directory is the client folder so WZ/DLL resolution works on Windows.
func Start(clientPath, ip string, port int) (*os.Process, error) {
	clientPath = strings.TrimSpace(clientPath)
	if clientPath == "" {
		return nil, fmt.Errorf("client path empty: pass -client or set %s", EnvClient)
	}
	abs, err := filepath.Abs(clientPath)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("client not found: %w", err)
	}
	if st.IsDir() {
		abs = filepath.Join(abs, "MapleStory.exe")
		if _, err := os.Stat(abs); err != nil {
			return nil, fmt.Errorf("directory given but MapleStory.exe missing under %s", clientPath)
		}
	}
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return nil, fmt.Errorf("launch ip empty")
	}
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port %d", port)
	}

	cmd := exec.Command(abs, ip, strconv.Itoa(port))
	cmd.Dir = filepath.Dir(abs)
	// Inherit nothing sensitive; client is GUI.
	cmd.Stdout = nil
	cmd.Stderr = nil

	if runtime.GOOS != "windows" {
		fmt.Fprintf(os.Stderr, "launcher note: GOOS=%s — CMS079 MapleStory.exe is Windows-native; use Wine or run the shell on Windows.\n", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", abs, err)
	}
	fmt.Printf("launched %s %s %d (pid=%d)\n", abs, ip, port, cmd.Process.Pid)
	return cmd.Process, nil
}
