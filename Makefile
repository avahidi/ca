
compile: build build/ca

build/ca: *.go
	go build -o build/ca

run: build/ca
	build/ca --prefix https://cht.sh/go/ strings

fmt:
	go fmt ./...

build:
	mkdir build

clean:
	rm -rf build
