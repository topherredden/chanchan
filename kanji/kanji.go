package kanji

import (
	"log"
	"fmt"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"time"
	"github.com/topherredden/chanchan/bot"
)

var DBFile string = "./cc.db"
//var MainChannelID string = "203238995117867008" //#japanesefromzero
var MainChannelID string = "365484608671842304" //#area51

func KanjiCommands()() {
	// Make sure DB loads
	db, err := sql.Open("sqlite3", DBFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	bot.BotAddCommand("!count", CountCmd, "!count", true)
	bot.BotAddCommand("!purge", PurgeCmd, "!purge", true)
	bot.BotAddCommand("!unregister", UnregisterCmd, "!unregister", false)
	bot.BotAddCommand("!kanji", KanjiCmd, "!kanji", false)
	bot.BotAddCommand("!status", StatusCmd, "!status", false)
	bot.BotAddCommand("!register", RegisterCmd, "!register <kanjigoal>", false)
	bot.BotAddCommand("!checkin", CheckinCmd, "!checkin <kanji>", false)
}

func CountCmd(state *bot.BotCommandState)(err error) {
	userCount := getUserCount()
	state.SendReply(fmt.Sprintf("There are %d registered users.", userCount))
	return
}

func PurgeCmd(state *bot.BotCommandState)(err error) {
	db, err := sql.Open("sqlite3", DBFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Go through each checkin
	sqlCmd := fmt.Sprintf("SELECT kanji, checkin_id, id FROM checkins")
	rows, err := db.Query(sqlCmd)
	if err != nil {
		log.Fatal(err)
	}

	type DuplicateEntry struct {
		newKanji string
		checkinID  int
		userID string
		duplicateCount int
	}

	var duplicateArray []DuplicateEntry

	for rows.Next() {
		var kanji string
		var checkinID int
		var userID string
		err = rows.Scan(&kanji, &checkinID, &userID)
		if err != nil {
			log.Fatal(err)
		}

		kanjiArray := []rune(kanji)
		newKanji := []rune{}

		for _, k := range kanjiArray {
			duplicate := false
			for _, j := range newKanji {
				if k == j {
					duplicate = true
					break
				}
			}

			if !duplicate {
				newKanji = append(newKanji, k)
			}
		}

		duplicates := len(kanjiArray) - len(newKanji)
		if duplicates > 0 {
			var duplicateEntry DuplicateEntry
			duplicateEntry.newKanji = string(newKanji)
			duplicateEntry.checkinID = checkinID
			duplicateEntry.duplicateCount = duplicates
			duplicateEntry.userID = userID

			duplicateArray = append(duplicateArray, duplicateEntry)
		}
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	rows.Close()

	// Fix duplicates
	for _, v := range duplicateArray {
		state.SendReply(fmt.Sprintf("Removed %d duplicates from %s checkinid %d", v.duplicateCount, v.userID, v.checkinID))

		// Write back
		sqlCmd := fmt.Sprintf("UPDATE checkins SET kanji='%s' WHERE checkin_id=%d", v.newKanji, v.checkinID)
		dbExec(sqlCmd)
	}

	return
}

func UnregisterCmd(state *bot.BotCommandState)(err error) {
	sqlCmd := fmt.Sprintf("delete from users where id=%s", state.AuthorID)
	_, err = dbExec(sqlCmd)
	if err != nil {
		return
	}

	state.SendReply(fmt.Sprintf("Successfully unregistered you from the Kanji Challenge."))
	return
}

func KanjiCmd(state *bot.BotCommandState)(err error) {
	if !isRegistered(state.AuthorID) {
		return
	}

	kanjiString := getKanjiString(state.AuthorID)
	state.SendReply(fmt.Sprintf("You have learned the following Kanji: %s", kanjiString))
	return
}

func StatusCmd(state *bot.BotCommandState)(err error) {
	if !isRegistered(state.AuthorID) {
		return
	}

	var kanjiCount int = getKanjiCount(state.AuthorID)
	var kanjiGoal int = getUserGoal(state.AuthorID)
	var kanjiProgress int = int((float32(kanjiCount) / float32(kanjiGoal)) * 100.0)

	state.SendReply(fmt.Sprintf("%d Kanji learned out of a goal of %d (%d%%).", kanjiCount, kanjiGoal, kanjiProgress))
	return
}

func CheckinCmd(state *bot.BotCommandState)(err error) {
	if !isRegistered(state.AuthorID) {
		return
	}

	// Get whole command and parse it for Kanji
	runes := []rune(state.ArgText)

	// Check for each kanji and add to list
	addKanji := []rune{}
	for _, r := range runes {
		if isKanji(r) {
			// Only add if not duplicate
			duplicate := false
			for _, k := range addKanji {
				if r == k {
					duplicate = true
					break
				}
			}

			if !duplicate {
				addKanji = append(addKanji, r)
			}
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
	sqlCmd := fmt.Sprintf("select kanji from checkins where id=%s", state.AuthorID)
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
		state.SendReply(fmt.Sprintf("No New Kanji were checked-in! Either you put none in the command, or they were all duplicates."))
	}

	if len(duplicateKanji) > 0 {
		// Convert Kanji to String
		var duplicateString = string(duplicateKanji)
		state.SendReply(fmt.Sprintf("%d Duplicate Kanji detected! These will be ignored. (%s)", len(duplicateKanji), duplicateString))
	}

	if !(len(newKanji) > 0) {
		return
	}

	// Otherwise insert new kanji
	var newKanjiString = string(newKanji)
	var unixTime int64 = time.Now().Unix()

	log.Println("Inserting new kanji...")
	sqlCmd = fmt.Sprintf("insert into checkins(id, kanji, date, count) values('%s', '%s', %d, %d)", state.AuthorID, newKanjiString, unixTime, len(newKanji))
	_, err = db.Exec(sqlCmd)
	if err != nil {
		log.Fatal(err)
		return
	}

	var kanjiCount int = len(newKanji) + len(oldKanji)
	var kanjiGoal int = getUserGoal(state.AuthorID)
	var kanjiProgress int = int((float32(kanjiCount) / float32(kanjiGoal)) * 100.0)

	state.SendReply(fmt.Sprintf("Successfully checked-in %d New Kanji (%s).\n\n%d/%d %d%% of Goal.", len(newKanji), newKanjiString, kanjiCount, kanjiGoal, kanjiProgress))
	state.SendChannel(MainChannelID, fmt.Sprintf("<@%s> just checked-in with %d New Kanji (%d%% of Goal). Keep it up!", state.AuthorID, len(newKanji), kanjiProgress))

	return
}

func RegisterCmd(state *bot.BotCommandState)(err error) {
	var kanjiGoal int64
	err = state.ParseInt(&kanjiGoal)
	if err != nil {
		state.SendHelp()
		return
	}

	db, err := sql.Open("sqlite3", DBFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check user doesn't exist
	if isRegistered(state.AuthorID) {
		state.SendReply(fmt.Sprintf("You have already registered!\n\nTo unregister, use !unregister."))
		return
	}

	// Add user to DB
	sqlCmd := fmt.Sprintf("insert into users(id, kanjigoal) values('%s', %d)", state.AuthorID, kanjiGoal)
	log.Printf("Inserting user: %s", state.AuthorID)
	_, err = dbExec(sqlCmd)
	if err != nil {
		return
	}

	// Send Positive Response
	state.SendReply(fmt.Sprintf("You have been registered for %d Kanji in the Kanji Challenge!\n\nUse !checkin with any Kanji to add those to your learned Kanji. (e.g. !checkin 食日)", kanjiGoal))
	state.SendChannel(MainChannelID, fmt.Sprintf("<@%s> has been registered for %d Kanji in the Kanji Challenge!", state.AuthorID, kanjiGoal))

	return
}

func dbExec(cmd string)(sql.Result, error) {
	db, err := sql.Open("sqlite3", DBFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	result, err := db.Exec(cmd)
	if err != nil {
		log.Fatal(err)
	}

	return result, err
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
