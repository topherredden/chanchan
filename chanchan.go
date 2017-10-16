package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
    "log"
)

var Token string
var adminID string

func init() {
    log.Printf("Starting up...")

	flag.StringVar(&Token, "t", "", "Bot Token")
    flag.StringVar(&adminID, "admin", "", "AdminID")
	flag.Parse()

}

func main() {
	BotOpen(Token, adminID)

	KanjiCommands()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

    BotClose()
}

