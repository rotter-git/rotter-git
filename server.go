package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hasura/go-graphql-client"
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

	keysClient  *KeysClient
	adminClient *graphql.Client
	addr        string
	reposDir    string
}

type pkContextKey struct{}

func NewSSH(addr string, keys *KeysClient, admin *graphql.Client) (s *SSH) {
	s = new(SSH)

	s.SSH = gitkit.NewSSH(config)
	s.SSH.PublicKeyLookupFunc = s.verifyKey
	s.SSH.PreLoginFunc = s.verifyUser
	s.SSH.AuthoriseOperationFunc = s.authorise

	s.keysClient = keys
	s.adminClient = admin

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

	pk = &gitkit.PublicKey{
		Id:   u.Id,
		Name: u.DisplayName,
	}

	return
}

func (s *SSH) authorise(ctx context.Context, cmd *gitkit.GitCommand) error {
	ctxPK := ctx.Value(gitkit.PublicKeyContextKey{})
	if ctxPK == nil {
		return fmt.Errorf("missing public key")
	}

	pk := ctxPK.(gitkit.PublicKey)

	var query struct {
		Repo struct {
			Namespace struct {
				Members []struct {
					Name string
				}
			}
		} `graphql:"repo(path: $path)"`
	}

	variables := map[string]interface{}{
		"path": cmd.Repo,
	}

	err := s.adminClient.Query(context.Background(), &query, variables)
	if err != nil {
		return err
	}

	for _, member := range query.Repo.Namespace.Members {
		if pk.Name == member.Name {
			return nil
		}
	}

	return fmt.Errorf("key %#v does not have access to %q",
		pk, cmd.Repo,
	)
}

func (s *SSH) Serve() error {
	return s.ListenAndServe(s.addr)
}
