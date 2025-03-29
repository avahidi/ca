
compile: bin bin/ca

bin/ca: *.go
	go build -o bin/ca

run: bin/ca
	bin/ca --prefix https://cht.sh/go/ strings

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

bin:
	mkdir bin

clean:
	rm -rf bin
