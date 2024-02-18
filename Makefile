build:
	go build -o bin/bot

start: build
	./bin/bot