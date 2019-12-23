all: build 

container_build: build
	podman build -t plant-controller:latest .

build: builddir
	GOOS=linux GOARCH=amd64 go build -o build/plant-controller cmd/plant-controller/main.go

test:
	go test -v ./...

builddir:
	mkdir -p build

clean:
	rm -rf build
