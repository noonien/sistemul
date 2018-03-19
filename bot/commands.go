package bot

import (
	"fmt"
	"log"
	"runtime/debug"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) RegisterCmd(ns, cmd string, handler CmdHandler) {
	cmd = strings.ToLower(cmd)

	_, ok := b.commands[cmd]
	if ok {
		log.Printf("command: %s is already registered", cmd)
		return
	}

	_, ok = b.aliases[cmd]
	if ok {
		log.Printf("command: %s is already an alias", cmd)
		return
	}

	b.commands[cmd] = &cmdDesc{
		Namespace: ns,
		Handler:   handler,
	}
}

func (b *Bot) RemoveCmd(cmd string) {
	delete(b.commands, cmd)
}

func (b *Bot) RegisterAlias(alias, cmd string) {
	_, ok := b.aliases[alias]
	if ok {
		log.Printf("alias: %s was already registered", alias)
		return
	}

	_, ok = b.commands[alias]
	if ok {
		log.Printf("alias: %s is already a command", alias)
		return
	}

	b.aliases[alias] = cmd
}

func (b *Bot) RemoveAlias(alias string) {
	delete(b.aliases, alias)
}

func (b *Bot) messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	content := strings.TrimSpace(m.Content)

	prefix := "!"
	if !strings.HasPrefix(content, prefix) {
		return
	}
	content = content[len(prefix):]

	cmdline := strings.SplitN(content, " ", 2)
	if len(cmdline) == 0 {
		return
	}

	var cmd Command
	cmd.MessageCreate = m

	cmd.Cmd = strings.ToLower(cmdline[0])
	if cmd.Cmd == "" {
		return
	}
	if len(cmdline) == 2 {
		cmd.Args = strings.TrimSpace(cmdline[1])
	}

	desc, ok := b.commands[cmd.Cmd]
	if !ok {
		aliasCmd, ok := b.aliases[cmd.Cmd]
		if !ok {
			return
		}

		aliasCmdline := strings.SplitN(aliasCmd, " ", 2)

		cmd.Cmd = strings.ToLower(aliasCmdline[0])
		desc, ok = b.commands[cmd.Cmd]
		if !ok {
			return
		}

		if len(aliasCmdline) == 2 {
			args := aliasCmdline[1]
			if len(cmd.Args) > 0 {
				args += " " + cmd.Args
			}

			cmd.Args = args
		}
	}

	if desc.Handler == nil {
		return
	}

	var err error
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				debug.PrintStack()
				rerr, ok := rec.(error)
				if ok {
					err = rerr
				} else {
					err = fmt.Errorf("panic: %v", rec)
				}
			}
		}()
		err = desc.Handler(b, &cmd)
	}()
	if err != nil {
		b.DG.ChannelMessageSend(cmd.ChannelID, "`oops, something bad happened`")
		log.Printf("error while executing `%s %s`: %s", cmd.Cmd, cmd.Args, err.Error())
	}
}
