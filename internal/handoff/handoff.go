// Package handoff writes a side-car JSON next to the stock client so a future
// packet shim (or external tool) can finish Scheme A without re-selecting.
package handoff

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Info is written after LoginBridge /api/select succeeds.
type Info struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	CharID    int    `json:"charId"`
	AuthIP    string `json:"authIp"`
	Channel   int    `json:"channel"`
	ShimAddr  string `json:"shimAddr,omitempty"`
	Timestamp string `json:"timestamp"`
	Scheme    string `json:"scheme"`
	Note      string `json:"note"`
}

const FileName = "handoff.json"

// WriteBesideClient writes handoff.json next to MapleStory.exe (or under clientPath if it is a dir).
func WriteBesideClient(clientPath string, info Info) (string, error) {
	if info.Timestamp == "" {
		info.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if info.Scheme == "" {
		info.Scheme = "A"
	}
	if info.Note == "" {
		info.Note = "Phase1: launcher + stub shim. Remaining: MapleAESOFB + encrypted SERVER_IP (0x0B) after fake login so stock client skips its own UI."
	}

	dir, err := resolveClientDir(clientPath)
	if err != nil {
		// Fall back to cwd so select still leaves an artifact.
		dir, _ = os.Getwd()
	}
	path := filepath.Join(dir, FileName)
	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func resolveClientDir(clientPath string) (string, error) {
	clientPath = filepath.Clean(clientPath)
	st, err := os.Stat(clientPath)
	if err != nil {
		return "", err
	}
	if st.IsDir() {
		return clientPath, nil
	}
	return filepath.Dir(clientPath), nil
}

// DocRemaining returns the human-readable Phase-2 checklist.
func DocRemaining() string {
	return fmt.Sprintf(`Scheme A remaining work (see internal/shim):
  1. Port MapleAESOFB (AES-ECB OFB + funnyBytes IV rollover) from MapleStory src/tools/MapleAESOFB.java
  2. After Hello, decrypt client packets (header check + crypt)
  3. Minimal fake-login sequence until client will accept SERVER_IP:
     LOGIN_STATUS → SERVERLIST → CHARLIST (or the CMS079 subset your client expects)
  4. Encrypt+send getServerIP: opcode 0x0B, short 0, IPv4(host), short(port), int(charId), {1,0,0,0,0}
     (Java: MaplePacketCreator.getServerIP)
  5. Client then connects to real channel; putLoginAuth(charId, authIp, …) must match client outbound IP
`)
}
