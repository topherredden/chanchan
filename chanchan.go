package chanchan

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
    "log"
    "github.com/topherredden/chanchan/bot"
	"github.com/topherredden/chanchan/kanji"
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
	bot.BotOpen(Token, adminID)

	kanji.KanjiCommands()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

    bot.BotClose()
}

