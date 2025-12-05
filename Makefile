.PHONY: build run test clean

APP_NAME=agent-scheduler

build:
	go build -o $(APP_NAME) main.go

INPUT ?= testdata/data.csv

run: build
	./$(APP_NAME) -input $(INPUT) -format text

test:
	go test ./... -v

clean:
	rm -f $(APP_NAME)
