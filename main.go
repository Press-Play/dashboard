package main

import (
    "log"
    "os"
    "time"
    "fmt"
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
    list, err := clients.trelloClient.GetList("5a964a9ef272819fb87ebb15", trello.Defaults())
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
    tnow := time.Now().Local().Truncate(time.Hour * 24)
    tmin := tnow.Format(time.RFC3339)
    tmax := tnow.AddDate(0, 0, 1).Format(time.RFC3339)
    events, err := clients.calendarClient.Events.List("primary").ShowDeleted(false).
        SingleEvents(true).TimeMin(tmin).TimeMax(tmax).OrderBy("startTime").Do()
    if err != nil {
        log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
    }
    fmt.Println("Upcoming events:")
    if len(events.Items) == 0 {
        fmt.Println("No upcoming events found.")
    } else {
        for _, item := range events.Items {
            date := item.Start.DateTime
            if date == "" {
                    date = item.Start.Date
            }
            fmt.Printf("%v (%v)\n", item.Summary, date)
        }
    }

    data := struct {
        Title string
        Tasklist []*trello.Card
        Card *trello.Card
    }{
        Title: r.URL.Path,
        Tasklist: cards,
        Card: selectedCard,
    }
    t, _ := template.ParseFiles("main.html")
    t.Execute(w, data)
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

func NewClients(trelloClient  *trello.Client, calendarClient  *googlecalendar.Service) Clients {
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

    // Serve the webpage and listen for requests.
    log.Print("http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
