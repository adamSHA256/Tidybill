APP_NAME := invoice
SRC := ./cmd/invoice/

.PHONY: build run clean build-linux build-windows build-all

build:
	go build -o $(APP_NAME) $(SRC)

run: build
	./$(APP_NAME)

clean:
	rm -f $(APP_NAME) $(APP_NAME).exe $(APP_NAME)-linux

build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(APP_NAME)-linux $(SRC)

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(APP_NAME).exe $(SRC)

build-all: build-linux build-windows
