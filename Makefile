all: build 

container_build: build
	podman build -t greenhouse-controller:latest .

build: builddir
	GOOS=linux GOARCH=amd64 go build -o build/greenhouse-controller cmd/greenhouse-controller/main.go

test:
	go test -v ./...

builddir:
	mkdir -p build

clean:
	rm -rf build
