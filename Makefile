.PHONY: clean
clean:
	rm -rf output/ && rm -f moire

.PHONY: build
build:
	go build .

.PHONY: run
run:
	go run main.go
