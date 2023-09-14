package main

import (
	"context"
	"crypto/tls"

	"github.com/rotter-git/rotter-keys/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type authProvider interface {
	Lookup(key []byte) (u *server.User, err error)
}

type KeysClient struct {
	ksc server.KeyServiceClient
}

func NewKeyServiceClient(addr string) (c *KeysClient, err error) {
	conf := grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	}))

	conn, err := grpc.Dial(addr, conf)
	if err != nil {
		return
	}

	c = new(KeysClient)
	c.ksc = server.NewKeyServiceClient(conn)

	return
}

func (c *KeysClient) Lookup(key []byte) (u *server.User, err error) {
	return c.ksc.Lookup(context.Background(), &server.PublicKey{
		Contents: key,
		Type:     server.KeyType_SSH,
	})
}
