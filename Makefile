SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -path ./docker -prune -o -name '*.go')

zonewatcher: $(SOURCES)
	go build cmd/zonewatcher.go

zonewatcher-linux: $(SOURCES)
	GOOS=linux GOARCH=amd64 go build -o zonewatcher-linux-amd64 cmd/zonewatcher.go
