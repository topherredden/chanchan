package bot

import (
	"github.com/bwmarrin/discordgo"
	"log"
	"fmt"
	"strings"
	"strconv"
)

type BotCommandState struct {
	cmd string
	args []string
	ArgText string
	AuthorID string
	channelID string
	argCursor int
	botCommand *BotCommand
}

func (state *BotCommandState) NextArg()(arg string, err error) {
	if state.argCursor >= len(state.args) {
		err = fmt.Errorf("not enough command arguments")
		return
	}

	arg = state.args[state.argCursor]
	state.argCursor++
	return
}

func (state *BotCommandState) ParseInt(v *int64)(err error) {
	arg, err := state.NextArg()
	if err != nil {
		return
	}

	*v, err = strconv.ParseInt(arg, 10, 64)
	return
}

func (state *BotCommandState) ParseUInt(v *uint64)(err error) {
	arg, err := state.NextArg()
	if err != nil {
		return
	}

	*v, err = strconv.ParseUint(arg, 10, 64)
	return
}

func (state *BotCommandState) ParseFloat(v *float64)(err error) {
	arg, err := state.NextArg()
	if err != nil {
		return
	}

	*v, err = strconv.ParseFloat(arg, 64)
	return
}

func (state *BotCommandState) ParseString(v *string)(err error) {
	arg, err := state.NextArg()
	if err != nil {
		return
	}

	//*v, err = strconv.ParseInt(arg, 10, 64)
	*v = arg
	return
}

func (state *BotCommandState) ParseText(v *string)(err error) {
	if state.argCursor >= len(state.args) {
		err = fmt.Errorf("not enough command arguments")
		return
	}

	*v = state.ArgText
	return
}

func (state *BotCommandState) IsAdmin()(isAdmin bool, err error) {
	isAdmin, err = BotIsAdmin(state.AuthorID)
	return
}

func (state *BotCommandState) SendHelp()() {
	log.Printf("Channel: %s, Help: %s", state.channelID, state.botCommand.helpText)
	dg.ChannelMessageSend(state.channelID, state.botCommand.helpText)
}

func (state *BotCommandState) SendReply(reply string)() {
	dg.ChannelMessageSend(state.channelID, reply)
}

func (state *BotCommandState) SendChannel(channelID string, reply string)() {
	dg.ChannelMessageSend(channelID, reply)
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

func BotOpen(token string, adminID string) (err error) {

	// Create Discord Session
	dg, err = discordgo.New("Bot " + token)
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

	cmds = make(map[string]*BotCommand)
	adminIDs = append(adminIDs, adminID)

	return
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

	return
}

func BotIsAdmin(userID string)(isAdmin bool, err error){
	isAdmin = false

	for _, adminID := range adminIDs {
		if userID == adminID {
			isAdmin = true
			return
		}
	}

	return
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
		ArgText: argText,
		AuthorID: m.Author.ID,
		channelID: m.ChannelID,
		argCursor: 0,
		botCommand: cmds[command[0]],
	}

	cmds[command[0]].handler(state)
}


