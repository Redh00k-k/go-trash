BINARY_NAME=go-trash

build:
ifeq ($(OS), Windows_NT)
	cd lib && gcc -lstdc++ -c trashbox.cpp && ar rusv libtrashbox.a trashbox.o 
	go build -o ${BINARY_NAME}.exe -ldflags="-s -w" -trimpath main.go utils_windows.go
else
	GOARCH=amd64 GOOS=linux go build -o ${BINARY_NAME} -ldflags="-s -w" -trimpath main.go utils_linux.go
endif

run: build

clean:
	go clean
ifeq ($(OS), Windows_NT)
	cd lib && del libtrashbox.a && del trashbox.o
endif