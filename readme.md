# Quickstart

1. Install Golang: https://golang.org/dl/
2. Configure environment variables for Trello: https://trello.com/app-key
    * `touch .env`
    * `TRELLO_APP_KEY=X` (Personal Key)
    * `TRELLO_TOKEN=X` (Server Token)
3. Configure environment variable for Google Oauth
    * Download `credentials.json` from Google Cloud Console.
3. Start server: `go run main.go`
