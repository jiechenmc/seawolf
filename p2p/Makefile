BUILD_DIR=$(PWD)/bin

all: build

bin:
	mkdir -p $(BUILD_DIR)

build: bin
	go build -C $(BUILD_DIR) $(PWD)/cmd/p2p/main.go && \
	cp $(BUILD_DIR)/main seawolf_p2p

run: build
	$(BUILD_DIR)/main

clean:
	rm -rf $(BUILD_DIR)/*

