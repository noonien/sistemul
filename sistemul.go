package main // import "github.com/noonien/sistemul"
import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/noonien/sistemul/bot"
	"github.com/noonien/sistemul/plugins"
)

var token = flag.String("token", "", "discord token")

func main() {
	bot, err := bot.New("sistemul.db", "Bot "+*token)
	if err != nil {
		panic(err)
	}

	plugins.RegisterRedditRoulette(bot)

	err = bot.Start()
	if err != nil {
		panic(err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Airhorn is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	bot.Stop()
}
