package main

import (
    "log"
    "os"
    "net/http"
    "html/template"
    "github.com/adlio/trello"
    "github.com/joho/godotenv"
)

func doneHandler(client *trello.Client, w http.ResponseWriter, r *http.Request) {
    id := r.FormValue("id")

    card, err := client.GetCard(id, trello.Defaults())
    if err != nil {
        log.Fatal(err)
    }

    // Move to the list of done tasks.
    card.MoveToList("5c2186487bee19154d85003f", trello.Defaults())
    
    http.Redirect(w, r, "/", http.StatusFound)
}

func dashboardHandler(client *trello.Client, w http.ResponseWriter, r *http.Request) {
    // Get the list of next tasks.
    list, err := client.GetList("5a964a9ef272819fb87ebb15", trello.Defaults())
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
        for _, card := range cards {
            selectedCard = card
        }
    } else {
        selectedCard, err = client.GetCard(id, trello.Defaults())
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

type TrelloConfig struct {
    AppKey string
    Token  string
}

func getTrelloConfig() *TrelloConfig {
    return &TrelloConfig {
        AppKey: getEnv("TRELLO_APP_KEY", ""),
        Token: getEnv("TRELLO_TOKEN", ""),
    }
}

func getEnv(key string, defaultValue string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }

    return defaultValue
}

type TrelloHandler struct {
    client  *trello.Client
    handler func(*trello.Client, http.ResponseWriter, *http.Request)
}

func (h *TrelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    h.handler(h.client, w, r)
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

    // Handle routes.
    http.Handle("/", &TrelloHandler{trelloClient, dashboardHandler})
    http.Handle("/done/", &TrelloHandler{trelloClient, doneHandler})
    http.Handle("/styles/", http.StripPrefix("/styles/", http.FileServer(http.Dir(""))))

    // Serve the webpage and listen for requests.
    log.Print("http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

