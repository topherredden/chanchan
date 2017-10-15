package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
    "log"
    "database/sql"
    "strconv"
    "strings"
    "time"

	"github.com/bwmarrin/discordgo"
    _ "github.com/mattn/go-sqlite3"
)

var DBFile string = "./cc.db"
//var MainChannelID string = "203238995117867008" //#japanesefromzero
var MainChannelID string = "365484608671842304" //#area51
var Token string
var adminID string

func init() {
    log.Printf("Starting up...")

	flag.StringVar(&Token, "t", "", "Bot Token")
    flag.StringVar(&adminID, "admin", "", "AdminID")
	flag.Parse()

    db, err := sql.Open("sqlite3", DBFile)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func isKanji(r rune) (bool) {
    if (r >= 0x4E00 && r <= 0x9FA5) || (r >= 0x3005 && r <= 0x3007) {
        return true
    }

    return false
}

func getUserGoal(id string) (int) {
    db, err := sql.Open("sqlite3", DBFile)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    sqlCmd := fmt.Sprintf("select kanjigoal from users where id=%s", id)
    row := db.QueryRow(sqlCmd)

    var goal int
    err = row.Scan(&goal)
    if err != nil {
        log.Fatal(err)
    }

    return goal
}

func getKanjiCount(id string) (int) {
    db, err := sql.Open("sqlite3", DBFile)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    sqlCmd := fmt.Sprintf("select count from checkins where id=%s", id)
    rows, err := db.Query(sqlCmd)
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    var totalCount int = 0
    for rows.Next() {
        var count int
        err = rows.Scan(&count)
        if err != nil {
            log.Fatal(err)
        }

        totalCount = totalCount + count
    }
    err = rows.Err()
    if err != nil {
        log.Fatal(err)
    }

    return totalCount
}

func getKanjiString(id string) (string) {
    db, err := sql.Open("sqlite3", DBFile)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    sqlCmd := fmt.Sprintf("select kanji from checkins where id=%s", id)
    rows, err := db.Query(sqlCmd)
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    var totalKanji string
    for rows.Next() {
        var kanji string
        err = rows.Scan(&kanji)
        if err != nil {
            log.Fatal(err)
        }

        totalKanji = fmt.Sprintf("%s%s", totalKanji, kanji)
    }
    err = rows.Err()
    if err != nil {
        log.Fatal(err)
    }

    return totalKanji
}

func isRegistered(id string) (bool) {
    db, err := sql.Open("sqlite3", DBFile)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    sqlCmd := fmt.Sprintf("select id from users where id=%s", id)
    row := db.QueryRow(sqlCmd)

    var count int
    row.Scan(&count)

    if count > 0 {
        return true
    }

    return false
}

func isAdmin(id string) (bool) {
    if id == adminID {
        return true
    }

    return false
}

func getUserCount() (int) {
    db, err := sql.Open("sqlite3", DBFile)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    var count int
    row := db.QueryRow("SELECT COUNT(*) FROM users")
	err = row.Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

    return count
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
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

    if m.Content == "!count" {
        if !(isAdmin(m.Author.ID)) {
            return
        }

        userCount := getUserCount()

        s.ChannelMessageSend(c.ID, fmt.Sprintf("There are %d registered users.", userCount))
        return
    }

    if m.Content == "!unregister" {
        db, err := sql.Open("sqlite3", DBFile)
        if err != nil {
            log.Fatal(err)
        }
        defer db.Close()

        sqlCmd := fmt.Sprintf("delete from users where id=%s", m.Author.ID)
        _, err = db.Exec(sqlCmd)
        if err != nil {
            log.Fatal(err)
            return
        }

        s.ChannelMessageSend(c.ID, "Successfully unregistered you from the Kanji Challenge.")
        return
    }

    if m.Content == "!kanji" {
        if !isRegistered(m.Author.ID) {
            return
        }

        kanjiString := getKanjiString(m.Author.ID)

        s.ChannelMessageSend(c.ID, fmt.Sprintf("You have learned the following Kanji: %s", kanjiString))
        return
    }

    if m.Content == "!status" {
        if !isRegistered(m.Author.ID) {
            return
        }

        var kanjiCount int = getKanjiCount(m.Author.ID)
        var kanjiGoal int = getUserGoal(m.Author.ID)
        var kanjiProgress int = int((float32(kanjiCount) / float32(kanjiGoal)) * 100.0)

        s.ChannelMessageSend(c.ID, fmt.Sprintf("%d Kanji learned out of a goal of %d (%d%%).", kanjiCount, kanjiGoal, kanjiProgress))
        return
    }

    command := strings.Fields(m.Content)
    if !(len(command) >0) {
        return
    }

    if command[0] == "!register" {
        // Parse Params
        if !(len(command) >= 2) {
            s.ChannelMessageSend(c.ID, "Invalid 'register' Command!")
            return
        }

        kanjiGoal, err := strconv.Atoi(command[1])
        if err != nil || !(kanjiGoal>0) {
            s.ChannelMessageSend(c.ID, "Invalid Kanji Goal Number!")
            return
        }

        db, err := sql.Open("sqlite3", DBFile)
        if err != nil {
            log.Fatal(err)
        }
        defer db.Close()

        // Check user doesn't exist
        sqlCmd := fmt.Sprintf("select id from users where id=%s", m.Author.ID)
        row := db.QueryRow(sqlCmd)

        var count int
        row.Scan(&count)

        if count > 0 {
            s.ChannelMessageSend(c.ID, "You have already registered!\n\nTo unregister, use !unregister.")
            return
        }

        // Add user to DB
        sqlCmd = fmt.Sprintf("insert into users(id, kanjigoal) values('%s', %d)", m.Author.ID, kanjiGoal)
        log.Printf("Inserting user: %s", m.Author.ID)
        log.Println(sqlCmd)
        _, err = db.Exec(sqlCmd)
        if err != nil {
            log.Fatal(err)
            return
        }

        // Send Positive Response
        s.ChannelMessageSend(c.ID, fmt.Sprintf("You have been registered for %d Kanji in the Kanji Challenge!\n\nUse !checkin with any Kanji to add those to your learned Kanji. (e.g. !checkin 食日)", kanjiGoal))
        s.ChannelMessageSend(MainChannelID, fmt.Sprintf("<@%s> has been registered for %d Kanji in the Kanji Challenge!", m.Author.ID, kanjiGoal))

        return
    }

    if command[0] == "!checkin" {
        if !isRegistered(m.Author.ID) {
            return
        }

        // Get whole command and parse it for Kanji
        runes := []rune(m.Content)

        // Check for each kanji and add to list
        addKanji := []rune{}
        for _, r := range runes {
            if isKanji(r) {
                addKanji = append(addKanji, r)
            }
        }

        // Get Previous Checked in Kanji
        oldKanji := []rune{}
        db, err := sql.Open("sqlite3", DBFile)
        if err != nil {
            log.Fatal(err)
        }
        defer db.Close()

        log.Println("Checking for duplicates...")
        sqlCmd := fmt.Sprintf("select kanji from checkins where id=%s", m.Author.ID)
        rows, err := db.Query(sqlCmd)
        if err != nil {
            log.Fatal(err)
        }
        defer rows.Close()

        for rows.Next() {
            var kanji string
            err = rows.Scan(&kanji)
            if err != nil {
                log.Fatal(err)
            }

            checkinRunes := []rune(kanji)

            for _, r := range checkinRunes {
                oldKanji = append(oldKanji, r)
            }
        }
        err = rows.Err()
        if err != nil {
            log.Fatal(err)
            return
        }

        // Split duplicates
        newKanji := []rune{}
        duplicateKanji := []rune{}

        for _, r := range addKanji {
            // Compare to old
            var duplicate = false
            for _, or := range oldKanji {
                if r == or {
                    duplicate = true
                    break
                }
            }

            if duplicate {
                duplicateKanji = append(duplicateKanji, r)
            } else {
                newKanji = append(newKanji, r)
            }
        }

        if !(len(newKanji) > 0) {
            s.ChannelMessageSend(c.ID, fmt.Sprintf("No New Kanji were checked-in! Either you put none in the command, or they were all duplicates."))
        }

        if len(duplicateKanji) > 0 {
            // Convert Kanji to String
            var duplicateString = string(duplicateKanji)
            s.ChannelMessageSend(c.ID, fmt.Sprintf("%d Duplicate Kanji detected! These will be ignored. (%s)", len(duplicateKanji), duplicateString))
        }

        if !(len(newKanji) > 0) {
            return
        }

        // Otherwise insert new kanji
        var newKanjiString = string(newKanji)
        var unixTime int64 = time.Now().Unix()

        log.Println("Inserting new kanji...")
        sqlCmd = fmt.Sprintf("insert into checkins(id, kanji, date, count) values('%s', '%s', %d, %d)", m.Author.ID, newKanjiString, unixTime, len(newKanji))
        _, err = db.Exec(sqlCmd)
        if err != nil {
            log.Fatal(err)
            return
        }

        var kanjiCount int = len(newKanji) + len(oldKanji)
        var kanjiGoal int = getUserGoal(m.Author.ID)
        var kanjiProgress int = int((float32(kanjiCount) / float32(kanjiGoal)) * 100.0)

        s.ChannelMessageSend(c.ID, fmt.Sprintf("Successfully checked-in %d New Kanji (%s).\n\n%d/%d %d%% of Goal.", len(newKanji), newKanjiString, kanjiCount, kanjiGoal, kanjiProgress))
        s.ChannelMessageSend(MainChannelID, fmt.Sprintf("<@%s> just checked-in with %d New Kanji (%d%% of Goal). Keep it up!", m.Author.ID, len(newKanji), kanjiProgress))

        return
    }
}
