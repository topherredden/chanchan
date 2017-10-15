package main

import (
	"github.com/bwmarrin/discordgo"
	"log"
	"fmt"
	"strings"
)

type BotCommandState struct {
	cmd string
	args []string
	argText string
	authorID string
	channelID string
	argCursor int
}

func (state *BotCommandState) ParseInt(v *int)(err error) {

}

func (state *BotCommandState) ParseUInt(v *uint)(err error) {

}

func (state *BotCommandState) ParseFloat(v *float64)(err error) {

}

func (state *BotCommandState) ParseString(v *string)(err error) {

}

func (state *BotCommandState) ParseText(v *string)(err error) {

}

type BotCmdHandler func(*BotCommandState)(error)

type BotCommand struct {
	handler BotCmdHandler
	helpText string
	adminOnly bool
}

var dg *discordgo.Session
var cmds map[string]*BotCommand
var adminIDs []string

func BotOpen(token string) (err error) {

	// Create Discord Session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageHandler)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Println("Bot is now running.")
}

func BotClose() {
	dg.Close()
}

func BotAddCommand(command string, handler BotCmdHandler, helpText string, adminOnly bool) (err error) {
	if _, ok := cmds[command]; ok {
		err = fmt.Errorf("duplicate command: %s", command)
		return
	}

	cmds[command] = &BotCommand {
		handler:handler,
		helpText:helpText,
		adminOnly:adminOnly,
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Find the channel that the message came from.
	c, err := s.State.Channel(m.ChannelID)
	if err != nil {
		// Could not find channel.
		return
	}

	// Ignore messages if not DM
	if c.Type != discordgo.ChannelTypeDM {
		return
	}

	// Split command args
	command := strings.Fields(m.Content)
	if !(len(command) >0) {
		return
	}

	// Check if key exists, if not just ignore
	if _, ok := cmds[command[0]]; !ok {
		return
	}

	// Join args into single text
	argText := strings.Join(command[1:], " ")

	// Compose command info and call handler
	state := &BotCommandState {
		cmd: command[0],
		args: command[1:],
		argText: argText,
		authorID: m.Author.ID,
		channelID: m.ChannelID,
		argCursor: 0,
	}

	cmds[command[0]].handler(state)
}


