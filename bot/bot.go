package bot

import (
	"fmt"

	"github.com/asdine/storm"
	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	DB *storm.DB
	DG *discordgo.Session

	commands map[string]*cmdDesc
	aliases  map[string]string
}

type cmdDesc struct {
	Namespace string
	Handler   CmdHandler
}

type CmdHandler func(b *Bot, cmd *Command) error

type Command struct {
	Cmd  string
	Args string

	*discordgo.MessageCreate
}

func New(dbPath, token string) (*Bot, error) {
	db, err := storm.Open(dbPath)
	if err != nil {
		return nil, err
	}

	dg, err := discordgo.New(token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		DB:       db,
		DG:       dg,
		commands: make(map[string]*cmdDesc),
		aliases:  make(map[string]string),
	}

	dg.AddHandler(bot.messageHandler)

	return bot, nil
}

func (b *Bot) Start() error {
	err := b.DG.Open()
	if err != nil {
		return err
	}

	return nil
}

func (b *Bot) Stop() error {
	var cerr error

	err := b.DB.Close()
	if err != nil && cerr == nil {
		cerr = err
	}

	err = b.DG.Close()
	if err != nil && cerr == nil {
		cerr = err
	}

	return cerr
}

func (b *Bot) Messagef(channelID, format string, args ...interface{}) error {
	message := fmt.Sprintf(format, args...)
	return b.Message(channelID, message)
}

func (b *Bot) Message(channelID, message string) error {
	_, err := b.DG.ChannelMessageSend(channelID, message)
	return err
}
