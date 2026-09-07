package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lnatpunblhna/mxd-login-shell/internal/bridge"
	"github.com/lnatpunblhna/mxd-login-shell/internal/handoff"
	"github.com/lnatpunblhna/mxd-login-shell/internal/launcher"
	"github.com/lnatpunblhna/mxd-login-shell/internal/shim"
)

func main() {
	bridgeURL := flag.String("bridge", "http://127.0.0.1:17979", "CMS079 LoginBridge base URL (localhost only)")
	user := flag.String("user", "", "account username (optional; prompts if empty)")
	pass := flag.String("pass", "", "account password (optional; prompts if empty)")
	clientPath := flag.String("client", "", "path to MapleStory.exe (or its folder); default env MXD_CLIENT")
	direct := flag.Bool("direct", false, "launch at real channel host:port instead of local Scheme A shim (stock client will show its own login UI)")
	noLaunch := flag.Bool("no-launch", false, "after select, only write handoff.json — do not start shim/client")
	flag.Parse()

	resolvedClient := launcher.ResolvePath(*clientPath)

	client := bridge.NewClient(*bridgeURL)
	health, err := client.Health()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bridge unreachable at %s: %v\n", *bridgeURL, err)
		fmt.Fprintln(os.Stderr, "Checkout MapleStory (login-bridge merged), rebuild maple.jar, start the server, then retry.")
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
		if strings.Contains(err.Error(), "already_logged_in") {
			fmt.Fprintln(os.Stderr, "account was online; unlocked — retrying login once")
			login, err = client.Login(username, password)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "login request failed: %v\n", err)
			os.Exit(1)
		}
	}
	if !login.OK {
		if login.Error == "already_logged_in" {
			fmt.Fprintln(os.Stderr, "account was online; unlocked — retrying login once")
			login, err = client.Login(username, password)
			if err != nil {
				fmt.Fprintf(os.Stderr, "login request failed: %v\n", err)
				os.Exit(1)
			}
		}
		if !login.OK {
			fmt.Fprintf(os.Stderr, "login rejected: %s\n", login.Error)
			os.Exit(1)
		}
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
	picked := chars.Characters[idx]
	ch := w.Channels[0]
	sel, err := client.Select(picked.ID, ch.ID)
	if err != nil || !sel.OK {
		fmt.Fprintf(os.Stderr, "select failed: %v %s\n", err, selectErr(sel))
		os.Exit(1)
	}

	charID := sel.CharacterID
	if charID == 0 {
		charID = picked.ID
	}
	channelID := sel.Channel
	if channelID == 0 {
		channelID = ch.ID
	}

	fmt.Printf("handoff ready → channel %s:%d charId=%d (authIp=%s)\n", sel.Host, sel.Port, charID, sel.AuthIP)
	fmt.Println("note: putLoginAuth IP must match the stock client's outbound IP (authIp above).")
	fmt.Print(handoff.DocRemaining())

	info := handoff.Info{
		Host:      sel.Host,
		Port:      sel.Port,
		CharID:    charID,
		AuthIP:    sel.AuthIP,
		Channel:   channelID,
		Scheme:    "A",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	var shimSrv *shim.Server
	launchIP := sel.Host
	launchPort := sel.Port

	if !*noLaunch && !*direct {
		shimSrv, err = shim.Start(shim.Config{
			ChannelHost: sel.Host,
			ChannelPort: sel.Port,
			CharID:      charID,
			AuthIP:      sel.AuthIP,
			Log:         os.Stderr,
			AccountID:   login.AccountID,
			AccountName: username,
			Gender:      byte(login.Gender),
			GM:          login.GM != 0,
			CharName:    picked.Name,
			CharLevel:   byte(picked.Level),
			CharJob:     uint16(picked.Job),
			WorldName:   w.Name,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "shim start failed: %v\n", err)
			os.Exit(1)
		}
		defer shimSrv.Close()
		launchIP = "127.0.0.1"
		launchPort = shimSrv.Port()
		info.ShimAddr = shimSrv.Addr()
		info.Note = "Phase2 shim: Hello + MapleAES/Shanda + fake login + encrypted SERVER_IP handoff."
		fmt.Printf("Scheme A Phase2: launching client at local shim %s (real channel %s:%d)\n", shimSrv.Addr(), sel.Host, sel.Port)
	} else if *direct {
		info.Scheme = "direct"
		info.Note = "Direct launch at channel. Stock MapleStory.exe still runs its own login UI — putLoginAuth alone is not enough."
		fmt.Printf("direct: launching client at channel %s:%d (own login UI expected)\n", launchIP, launchPort)
	}

	if resolvedClient != "" {
		path, err := handoff.WriteBesideClient(resolvedClient, info)
		if err != nil {
			fmt.Fprintf(os.Stderr, "handoff.json write failed: %v\n", err)
		} else {
			fmt.Printf("wrote %s\n", path)
		}
	} else {
		path, err := handoff.WriteBesideClient(".", info)
		if err != nil {
			fmt.Fprintf(os.Stderr, "handoff.json write failed: %v\n", err)
		} else {
			fmt.Printf("wrote %s (no -client; cwd)\n", path)
		}
	}

	if *noLaunch {
		fmt.Println("no-launch: skipping MapleStory.exe")
		return
	}

	if resolvedClient == "" {
		fmt.Fprintln(os.Stderr, "no client path: pass -client C:\\path\\to\\MapleStory.exe or set MXD_CLIENT")
		fmt.Fprintln(os.Stderr, "handoff is ready; start the exe manually when you have a path")
		if shimSrv != nil {
			fmt.Print("shim running — press Enter to stop… ")
			_, _ = in.ReadString('\n')
		}
		return
	}

	proc, err := launcher.Start(resolvedClient, launchIP, launchPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "launcher failed: %v\n", err)
		os.Exit(1)
	}
	_ = proc

	if shimSrv != nil {
		fmt.Print("client launched against Phase2 shim — press Enter to stop shim… ")
		_, _ = in.ReadString('\n')
	}
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
