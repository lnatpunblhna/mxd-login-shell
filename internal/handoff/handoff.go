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
		info.Note = "Phase2: shim Hello + MapleAES/Shanda + fake login + encrypted SERVER_IP (0x0B) -> real channel."
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

// DocRemaining returns the human-readable Phase-2 status note.
func DocRemaining() string {
	return fmt.Sprintf(`Scheme A Phase2 wired (see internal/maplecrypto + internal/shim):
  - MapleAESOFB + MapleCustomEncryption (Shanda) ported
  - Fake login: LOGIN_STATUS / SERVERLIST / SERVERSTATUS / CHARLIST stub
  - Encrypted SERVER_IP 0x0B on CHAR_SELECT
  Still verify: authIp matches client outbound IP; live client may need extra opcode replies
`)
}