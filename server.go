package main

import (
	"context"
	"os"

	"github.com/sosedoff/gitkit"
	"golang.org/x/crypto/ssh"
)

var (
	welcomeBanner = "Welcome {{ .Name }}!\r\n\r\nRotter does not provide shell access\r\nBut please take this as a sign that your key is recognised... if it shouldn't be, then please shout up\r\n\r\n"
	user          = "rotter"

	config = gitkit.Config{
		KeyDir:         "keys",
		Dir:            os.Getenv("REPOS_DIR"),
		Auth:           true,
		BannerTemplate: welcomeBanner,
	}
)

type Server interface {
	Serve() error
}

type SSH struct {
	*gitkit.SSH

	keysClient *KeysClient
	addr       string
	reposDir   string
}

func NewSSH(addr string, keys *KeysClient) (s *SSH) {
	s = new(SSH)

	s.SSH = gitkit.NewSSH(config)
	s.SSH.PublicKeyLookupFunc = s.verifyKey
	s.SSH.PreLoginFunc = s.verifyUser
	s.SSH.AuthoriseOperationFunc = s.authorise

	s.keysClient = keys
	s.addr = addr
	s.reposDir = "/tmp/repos"

	return
}

func (s *SSH) verifyUser(ctx context.Context, metadata ssh.ConnMetadata) error {
	u := ctx.Value(gitkit.UserContextKey{})

	if u != user {
		return gitkit.ErrIncorrectUser
	}

	return nil
}

func (s *SSH) verifyKey(ctx context.Context, data string) (pk *gitkit.PublicKey, err error) {
	u, err := s.keysClient.Lookup([]byte(data))
	if err != nil {
		return
	}

	return &gitkit.PublicKey{
		Id:   u.Id,
		Name: u.DisplayName,
	}, nil
}

func (s *SSH) authorise(ctx context.Context, cmd *gitkit.GitCommand) error {
	// TODO: here is where we decide whether a user is allowed to do
	// something to a repo
	return nil
}

func (s *SSH) Serve() error {
	return s.ListenAndServe(s.addr)
}
