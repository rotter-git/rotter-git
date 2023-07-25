package main

import (
	"os"

	"github.com/hasura/go-graphql-client"
)

var (
	keysAddr      = os.Getenv("KEYS")
	sshListener   = os.Getenv("SSH_ADDR")
	adminEndpoint = os.Getenv("ADMIN_ENDPOINT")
)

func main() {
	ksc, err := NewKeyServiceClient(keysAddr)
	if err != nil {
		panic(err)
	}

	adminClient := graphql.NewClient(adminEndpoint, nil)

	holder := make(chan error)

	for _, s := range []Server{
		NewSSH(sshListener, ksc, adminClient),
	} {
		go func() {
			holder <- s.Serve()
		}()
	}

	err = <-holder
	panic(err)
}
