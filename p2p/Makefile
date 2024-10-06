BUILD_DIR=$(PWD)/bin

all: build

bin:
	mkdir -p $(BUILD_DIR)

build: bin
	go build -C $(BUILD_DIR) $(PWD)/cmd/p2p/main.go

run: build
	$(BUILD_DIR)/main

clean:
	rm -rf $(BUILD_DIR)/*

