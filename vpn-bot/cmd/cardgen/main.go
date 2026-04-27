package main

import (
	"log"
	"os"
	"path/filepath"

	"freeride/vpn-bot/services/media"
)

func main() {
	if err := os.MkdirAll("img", 0o755); err != nil {
		log.Fatal(err)
	}

	cards := map[string]string{
		"vpn":      "VPN",
		"telegram": "Telegram Proxy",
		"client":   "VPN Client",
	}

	for fileName, title := range cards {
		b, err := media.RenderTitleCard(title)
		if err != nil {
			log.Fatalf("render %s: %v", fileName, err)
		}
		dst := filepath.Join("img", fileName+".png")
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			log.Fatalf("write %s: %v", dst, err)
		}
		log.Printf("generated %s", dst)
	}
}
