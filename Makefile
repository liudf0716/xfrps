export PATH := $(GOPATH)/bin:$(PATH)
export GO15VENDOREXPERIMENT := 1

all: fmt build

build: xfrps xfrpc

# compile assets into binary file
file:
	rm -rf ./assets/static/*
	cp -rf ./web/frps/dist/* ./assets/static
	go get -d github.com/rakyll/statik
	go install github.com/rakyll/statik
	rm -rf ./assets/statik
	go generate ./assets/...

fmt:
	go fmt ./...
	
xfrps:
	go build -o bin/xfrps ./cmd/frps
	@cp -rf ./assets/static ./bin

xfrpc:
	go build -o bin/xfrpc ./cmd/frpc

test: gotest

gotest:
	go test -v ./assets/...
	go test -v ./client/...
	go test -v ./cmd/...
	go test -v ./models/...
	go test -v ./server/...
	go test -v ./utils/...

alltest: gotest
	cd ./tests && ./run_test.sh && cd -
	go test -v ./tests/...
	cd ./tests && ./clean_test.sh && cd -

clean:
	rm -f ./bin/xfrpc
	rm -f ./bin/xfrps
	cd ./tests && ./clean_test.sh && cd -

save:
	godep save ./...
