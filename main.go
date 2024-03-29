package main

import (
    "log"
    "os"
    "time"
    "fmt"
    "math"
    "net/http"
    "html/template"
    "github.com/adlio/trello"
    "github.com/joho/godotenv"
    "github.com/press-play/dashboard/calendar"
    googlecalendar "google.golang.org/api/calendar/v3"
)

func doneHandler(clients Clients, w http.ResponseWriter, r *http.Request) {
    id := r.FormValue("id")

    card, err := clients.trelloClient.GetCard(id, trello.Defaults())
    if err != nil {
        log.Fatal(err)
    }

    // Move to the list of done tasks.
    card.MoveToList("5c2186487bee19154d85003f", trello.Defaults())
    
    http.Redirect(w, r, "/", http.StatusFound)
}

func dashboardHandler(clients Clients, w http.ResponseWriter, r *http.Request) {
    // Get the list of next tasks.
    list, err := clients.trelloClient.GetList("63a0024b9a72fb001d5e9eba", trello.Defaults())
    if err != nil {
        log.Fatal(err)
    }

    cards, err := list.GetCards(trello.Defaults())
    if err != nil {
        log.Fatal(err)
    }

    var selectedCard *trello.Card

    id := r.FormValue("id")

    if id == "" {
        selectedCard = cards[0]
    } else {
        selectedCard, err = clients.trelloClient.GetCard(id, trello.Defaults())
        if err != nil {
            log.Fatal(err)
        }
    }

    // TODO: Push this function upstream.
    selectedCard.LoadActions(trello.Defaults())

    // TODO: Filter for comment cards only.
    // selectedCard.Actions.FilterToListCommentActions()

    log.Printf("selectedCard:\n\t%+v", selectedCard)
    for _, action := range selectedCard.Actions {
        // action
        log.Printf("action:\n\t%+v", action)
        log.Printf("action data:\n\t%+v", action.Data)
    }

    // Load the calendar data.
    tnow := time.Now().Local()
    timeNow := float32(tnow.Hour()) + (float32(tnow.Minute()) / 60)
    tnow = time.Date(tnow.Year(), tnow.Month(), tnow.Day(), 0, 0, 0, 0, tnow.Location())
    tmin := tnow.Format(time.RFC3339)
    tmax := tnow.AddDate(0, 0, 1).Format(time.RFC3339)
    events, err := clients.calendarClient.Events.List("primary").ShowDeleted(false).
        SingleEvents(true).TimeMin(tmin).TimeMax(tmax).OrderBy("startTime").Do()
    if err != nil {
        log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
    }
    fmt.Println("Events object:")
    fmt.Println(events)
    fmt.Println("Upcoming events:")
    var times []string
    var sortedEvents []*Event

    if len(events.Items) == 0 {
        fmt.Println("No upcoming events found.")
    } else {
        ttemp := tnow
        for !ttemp.After(tnow.AddDate(0, 0, 1).Add(-time.Hour / 2)) {
            // fmt.Println(ttemp.Format("3 PM"))
            if getHalfHour(ttemp) == 0 {
                times = append(times, ttemp.Format("3 PM"))
            }

            appendedEvent := false;

            // if this hour is equal to the start time, append to array
            // if this hour is between the time (start inclusive, end exclusive), then add 1 to the last element
            // else add an empty hour block
            for _, item := range events.Items {
                eventTime, _ := time.Parse(time.RFC3339, item.Start.DateTime)
                if eventTime.Hour() == ttemp.Hour() && getHalfHour(eventTime) == getHalfHour(ttemp) {
                    sortedEvents = append(sortedEvents,
                        &Event{item.Summary, item.Start.DateTime, item.End.DateTime, true, 0.5})
                    appendedEvent = true;
                    break
                }
            }

            if appendedEvent {
                ttemp = ttemp.Add(time.Hour / 2)
                continue
            }

            // If last inserted was a visible event, then
            if len(sortedEvents)-1 >= 0 && sortedEvents[len(sortedEvents)-1].Visible {
                currentEventEnd, _ := time.Parse(time.RFC3339, sortedEvents[len(sortedEvents)-1].TimeEnd)

                if currentEventEnd.After(ttemp) {
                    sortedEvents[len(sortedEvents)-1].Hours = sortedEvents[len(sortedEvents)-1].Hours + 0.5
                } else {
                    sortedEvents = append(sortedEvents,
                        &Event{"", "", "", false, 0.5})
                }
            } else {
                sortedEvents = append(sortedEvents,
                    &Event{"", "", "", false, 0.5})
            }
            ttemp = ttemp.Add(time.Hour / 2)
        }
    }

    data := struct {
        Title string
        Events []*Event
        DateNow string
        TimeNow float32
        Times []string
        Tasklist []*trello.Card
        Card *trello.Card
    }{
        Title: r.URL.Path,
        DateNow: tnow.Format("Monday 2, January"),
        TimeNow: timeNow,
        Times: times,
        Events: sortedEvents,
        Tasklist: cards,
        Card: selectedCard,
    }
    t, _ := template.ParseFiles("main.html")
    t.Execute(w, data)
}

type Event struct {
    Summary string
    TimeStart string
    TimeEnd string
    Visible bool
    Hours float32
}

func getHalfHour(t time.Time) float32 {
    return float32(math.Floor(float64(t.Minute()) / 30) * 30)
}

func getEnv(key string, defaultValue string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }

    return defaultValue
}

type TrelloConfig struct {
    AppKey string
    Token  string
}

type Clients struct {
    trelloClient  *trello.Client
    calendarClient  *googlecalendar.Service
}

type ClientHandler struct {
    clients Clients
    handler func(Clients, http.ResponseWriter, *http.Request)
}

func (h *ClientHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    h.handler(h.clients, w, r)
}

func getTrelloConfig() *TrelloConfig {
    return &TrelloConfig {
        AppKey: getEnv("TRELLO_APP_KEY", ""),
        Token: getEnv("TRELLO_TOKEN", ""),
    }
}

func NewClients(trelloClient *trello.Client, calendarClient *googlecalendar.Service) Clients {
    return Clients {
        trelloClient: trelloClient,
        calendarClient: calendarClient,
    }
}

func main() {
    var err error

    // Check for envrionment values config file.
    err = godotenv.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Load environment variables.
    trelloConfig := getTrelloConfig()
    log.Printf("trello:\n\t%+v", trelloConfig)

    // Initiate Trello client.
    trelloClient := trello.NewClient(trelloConfig.AppKey, trelloConfig.Token)
    log.Printf("trello client:\n\t%+v", trelloClient)

    // Initiate Calendar client.
    calendarClient := calendar.GetClient()
    log.Printf("calendar client:\n\t%+v", calendarClient)

    // Handle routes.
    clients := NewClients(trelloClient, calendarClient)
    http.Handle("/", &ClientHandler{clients, dashboardHandler})
    http.Handle("/done/", &ClientHandler{clients, doneHandler})
    http.Handle("/styles/", http.StripPrefix("/styles/", http.FileServer(http.Dir(""))))
    http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(""))))

    // Serve the webpage and listen for requests.
    log.Print("http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
