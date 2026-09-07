package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lnatpunblhna/mxd-login-shell/internal/bridge"
)

func main() {
	bridgeURL := flag.String("bridge", "http://127.0.0.1:17979", "CMS079 LoginBridge base URL (localhost only)")
	user := flag.String("user", "", "account username (optional; prompts if empty)")
	pass := flag.String("pass", "", "account password (optional; prompts if empty)")
	flag.Parse()

	client := bridge.NewClient(*bridgeURL)
	health, err := client.Health()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bridge unreachable at %s: %v\n", *bridgeURL, err)
		fmt.Fprintln(os.Stderr, "Checkout feat/login-bridge, rebuild maple.jar, start the server, then retry.")
		os.Exit(1)
	}
	fmt.Printf("bridge ok: %+v\n", health)

	username := strings.TrimSpace(*user)
	password := *pass
	in := bufio.NewReader(os.Stdin)
	if username == "" {
		fmt.Print("username: ")
		line, _ := in.ReadString('\n')
		username = strings.TrimSpace(line)
	}
	if password == "" {
		fmt.Print("password: ")
		line, _ := in.ReadString('\n')
		password = strings.TrimSpace(line)
	}

	login, err := client.Login(username, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "login request failed: %v\n", err)
		os.Exit(1)
	}
	if !login.OK {
		fmt.Fprintf(os.Stderr, "login rejected: %s\n", login.Error)
		os.Exit(1)
	}
	fmt.Printf("logged in accountId=%d gm=%d\n", login.AccountID, login.GM)

	worlds, err := client.Worlds()
	if err != nil || !worlds.OK {
		fmt.Fprintf(os.Stderr, "worlds failed: %v %s\n", err, worldsErr(worlds))
		os.Exit(1)
	}
	if len(worlds.Worlds) == 0 || len(worlds.Worlds[0].Channels) == 0 {
		fmt.Fprintln(os.Stderr, "no worlds/channels from bridge")
		os.Exit(1)
	}
	w := worlds.Worlds[0]
	fmt.Printf("world %d channels:\n", w.ID)
	for _, ch := range w.Channels {
		fmt.Printf("  ch%d load=%d %s:%d\n", ch.ID, ch.Load, ch.Host, ch.Port)
	}

	chars, err := client.Characters(w.ID)
	if err != nil || !chars.OK {
		fmt.Fprintf(os.Stderr, "characters failed: %v %s\n", err, charsErr(chars))
		os.Exit(1)
	}
	if len(chars.Characters) == 0 {
		fmt.Fprintln(os.Stderr, "no characters; create one with the stock client first")
		os.Exit(1)
	}
	for i, c := range chars.Characters {
		fmt.Printf("  [%d] id=%d name=%s lv=%d job=%d\n", i, c.ID, c.Name, c.Level, c.Job)
	}

	fmt.Print("pick character index: ")
	idxLine, _ := in.ReadString('\n')
	idx, err := strconv.Atoi(strings.TrimSpace(idxLine))
	if err != nil || idx < 0 || idx >= len(chars.Characters) {
		fmt.Fprintln(os.Stderr, "bad index")
		os.Exit(1)
	}
	ch := w.Channels[0]
	sel, err := client.Select(chars.Characters[idx].ID, ch.ID)
	if err != nil || !sel.OK {
		fmt.Fprintf(os.Stderr, "select failed: %v %s\n", err, selectErr(sel))
		os.Exit(1)
	}
	fmt.Printf("handoff ready → %s:%d (authIp=%s)\n", sel.Host, sel.Port, sel.AuthIP)
	fmt.Println("next: launch stock CMS079 client into that channel (launcher TODO)")
}

func worldsErr(w *bridge.WorldsResponse) string {
	if w == nil {
		return ""
	}
	return w.Error
}

func charsErr(c *bridge.CharactersResponse) string {
	if c == nil {
		return ""
	}
	return c.Error
}

func selectErr(s *bridge.SelectResponse) string {
	if s == nil {
		return ""
	}
	return s.Error
}
