default: keys/ rotter-git

keys/:
	mkdir -p $@


rotter-git: *.go go.mod go.sum
	CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o $@
