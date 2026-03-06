.PHONY: build test clean

build:
	go build -o monorhyme-search .

test:
	go test ./...

clean:
	rm -f monorhyme-search monorhyme-search.exe
