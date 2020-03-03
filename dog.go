package main

import (
  "database/sql"
  "io/ioutil"
  "fmt"
  "net/http"
  "log"
  "time"
  "strconv"
  "os"
  "encoding/json"
  _ "github.com/mattn/go-sqlite3"
	"github.com/slack-go/slack"
)

type Dog struct {
  AdoptableSearch DogAttributes
}

type DogAttributes struct {
  Arn []string
  Age string
  AgeGroup string
  AnimalType string
  BehaviorTestList []string
  BuddyID string
  ChipNumber string
  Featured string
  ID string
  Location string
  MemoList []string
  Name string
  NoCats []string
  NoDogs []string
  NoKids []string
  OnHold string
  Photo string
  PrimaryBreed string
  SN string
  SecondaryBreed string
  Sex string
  SpecialNeeds []string
  Species string
  Stage string
  Sublocation string
  WildlifeIntakeCause []string
  WildlifeIntakeInjury []string
}

type DogResponse struct {
  Collection []Dog
}

var AgeLimit int

func sendSlackMessage(message string) {
  webhookURL := os.Getenv("SLACK_TOKEN")
  if len(webhookURL) == 0 {
    webhookURL = "https://hooks.slack.com/services/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"
  }
  attachment := slack.Attachment{
    Color:         "good",
    Text:          message,
    Footer:        "doggo api",
    FooterIcon:    "https://platform.slack-edge.com/img/default_application_icon.png",
  }
  msg := slack.WebhookMessage{
    Attachments: []slack.Attachment{attachment},
  }

  err := slack.PostWebhook(webhookURL, &msg)
  if err != nil {
    fmt.Println(err)
  }
}

func initDB() *sql.DB {
  db, _ := sql.Open("sqlite3", "./dogs.db")
  statement, _ := db.Prepare("CREATE TABLE IF NOT EXISTS dogs (id INTEGER PRIMARY KEY, name TEXT, age INT, dogId TEXT UNIQUE, available BOOL)")
  statement.Exec()
  return db
}

func insertDog(db *sql.DB, name string, age string, id string) sql.Result {
  statement, _ := db.Prepare("INSERT OR IGNORE INTO dogs (name, age, dogId, available) VALUES (?, ?, ?, ?)")
  result, err := statement.Exec(name, age, id, "true")
  if err != nil {
    log.Fatal(err)
  }
  // rowCnt, err := result.RowsAffected()
  // if err != nil {
  //   log.Fatal(err)
  // }
  return result
}

func setDogUnavailable(db *sql.DB, dogId string) {
  statement, _ := db.Prepare("UPDATE dogs SET available = 'false' where dogID = ?")
  statement.Exec(dogId)
}

func getDogAvailability(db *sql.DB, dogId string) bool {
  var available string
  row := db.QueryRow(`SELECT available FROM dogs WHERE dogId=$1`, dogId)
  switch err := row.Scan(&available); err {
    case sql.ErrNoRows:
      fmt.Println("No rows were returned!")
    case nil:
      // fmt.Println(name)
    default:
      panic(err)
  }
  avail, _ := strconv.ParseBool(available)
  return avail
}

func getDogs(db *sql.DB) []string {
  // Query the DB
  rows, err := db.Query(`SELECT dogId FROM dogs`)
  if err != nil {
      log.Fatal(err)
  }
  defer rows.Close()

  var dogId string
  var dogIds []string

  for rows.Next() {
    err := rows.Scan(&dogId)
    if err != nil {
        log.Fatal(err)
    }
    dogIds = append(dogIds, dogId)
  }

  return dogIds
}

func getNameByDogID(db *sql.DB, dogId string) string {
  var name string
  row := db.QueryRow(`SELECT name FROM dogs WHERE dogId=$1`, dogId)
  switch err := row.Scan(&name); err {
    case sql.ErrNoRows:
      fmt.Println("No rows were returned!")
    case nil:
      // fmt.Println(name)
    default:
      panic(err)
  }
  return name
}

func removeAdoptedDogs(db *sql.DB, savedDogIds []string, jsonDogIds []string) {
  for i := range savedDogIds {
    if !sliceContainString(savedDogIds[i], jsonDogIds) {
      if getDogAvailability(db, savedDogIds[i]) {
        setDogUnavailable(db, savedDogIds[i])
        dogName := getNameByDogID(db, savedDogIds[i])
        log.Printf("%s (%s) is no longer available", dogName, savedDogIds[i])
        sendSlackMessage(dogName + " (" + savedDogIds[i] + ") is no longer available")
      }
    }
  }
}

func sliceContainString(str string, list []string) bool {
    for _, i := range list {
        if i == str {
            return true
        }
    }
    return false
}

func main() {
  // Init DB
  db := initDB()
  dogURL := "https://www.boulderhumane.org/wp-content/plugins/Petpoint-Webservices-2018/pullanimals.php?type=dog"

  queryFrequency := os.Getenv("FREQUENCY")
  frequency, err := strconv.Atoi(queryFrequency)
  if err != nil {
    log.Fatalln(err)
  }
  if len(queryFrequency) == 0 {
    frequency = 30
  }

  log.Printf("Querying every %ss", queryFrequency)
  for {
    // Make request for JSON
    resp, err := http.Get(dogURL)
    if err != nil {
      log.Fatalln(err)
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)

    // Parse JSON
    dogs := make([]Dog,0)
    json.Unmarshal(body, &dogs)

    // Current Dog IDs from JSON
    var dogIds []string
    // Saved Dog IDs in database
    curDogs := getDogs(db)

    // Loop through JSON and load into DB
    for i:= 0; i < len(dogs); i++ {
      // fmt.Printf("Name: %s, Age: %s, ID: %s\n", dogs[i].AdoptableSearch.Name, dogs[i].AdoptableSearch.Age, dogs[i].AdoptableSearch.ID)

      // Insert dogs from JSON into DB
      dogInsert := insertDog(db, dogs[i].AdoptableSearch.Name, dogs[i].AdoptableSearch.Age, dogs[i].AdoptableSearch.ID)
      // fmt.Println(dogInsert)
      newDog, _ := dogInsert.RowsAffected()

      if newDog > 0 {
        log.Println("Inserting dog", dogs[i].AdoptableSearch.Name)
        sendSlackMessage(dogs[i].AdoptableSearch.Name + " (" + dogs[i].AdoptableSearch.ID + ") is now available")
      }

      // Append dogIds from JSON to local array
      dogIds = append(dogIds, dogs[i].AdoptableSearch.ID)
    }
    removeAdoptedDogs(db, curDogs, dogIds)

    time.Sleep(time.Duration(frequency) * time.Second)
  }
}
