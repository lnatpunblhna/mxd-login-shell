package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lnatpunblhna/mxd-login-shell/internal/bridge"
)

func main() {
	bridgeURL := flag.String("bridge", "http://127.0.0.1:17979", "CMS079 LoginBridge base URL (localhost only)")
	flag.Parse()

	client := bridge.NewClient(*bridgeURL)
	health, err := client.Health()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bridge unreachable at %s: %v\n", *bridgeURL, err)
		fmt.Fprintln(os.Stderr, "Start MapleStory LoginBridge first, then retry.")
		os.Exit(1)
	}
	fmt.Printf("bridge ok: %+v\n", health)
	fmt.Println("UI login/world/char select: TODO — next milestone")
}
