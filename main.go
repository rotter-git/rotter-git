package main

import (
	"os"
)

var (
	keysAddr    = os.Getenv("KEYS")
	sshListener = os.Getenv("SSH_ADDR")
)

func main() {
	ksc, err := NewKeyServiceClient(keysAddr)
	if err != nil {
		panic(err)
	}

	holder := make(chan error)

	for _, s := range []Server{
		NewSSH(sshListener, ksc),
	} {
		go func() {
			holder <- s.Serve()
		}()
	}

	err = <-holder
	panic(err)
}
