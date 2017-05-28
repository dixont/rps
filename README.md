# rps
Rock Paper Scissors coding challenge

### Run me!
#### Run server
1. `cd server`
2. `godep restore`
3. `go run server.go handlers.go` (You can have PORT, SECRET and LOG_LEVEL environment variables set, but they default to "8000", "NotReallyButKindOfSecret", and "INFO" respectively)
4. Navigate to `localhost:<DEFINED_PORT>`

#### Run app
1. `cd app`
2. `npm i`
3. `npm start`
4. If a browser doesn't automatically open with the app location, it should be at `localhost:3000`

(To play against yourself, open `localhost:3000` in 2 tabs)

### Test me!
1. `cd server`
2. `godep restore`
3. `go test`

### Next steps
Just a few next steps that would round out the project:
1. Views testing
2. More verbose testing of websocket handlers
3. Improved UX