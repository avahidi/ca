
compile: bin bin/ca

bin/ca: *.go
	go build -o bin/ca

run: bin/ca
	bin/ca "https://cht.sh/go/<what>" strings

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

clean-cache:
	rm -rf ~/.cache/ca
