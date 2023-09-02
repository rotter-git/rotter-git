package main

import (
	"context"
	"fmt"
	"os"
	"strings"

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

type adminClient interface {
	Query(ctx context.Context, q interface{}, variables map[string]interface{}, options ...graphql.Option) error
}

type adminQuery struct {
	Repo struct {
		Namespace struct {
			Members []struct {
				Name string
			}
		}
	} `graphql:"repo(path: $path)"`
}

// Server is an interface allowing for multiple servers
// to be run with different settings
type Server interface {
	Serve() error
}

// SSH provides git operations over SSH, by wrapping gitkit.SSH,
// with a number of extra verification and validation functions
// such as:
//
//  1. Calling out to the rotter keys service to return information
//     about the validity of a key (Authn)
//  2. Calling out to the rotter graphql API to ensure a user is allowed
//     to perform certain operations against specific repositories
type SSH struct {
	*gitkit.SSH

	keysClient  authProvider
	adminClient adminClient
	addr        string
	reposDir    string
}

// NewSSH configures an SSH type, which in turn is an implementation of the
// Server interface, in order to provide git operations over... you guessed it...
// SSH
func NewSSH(addr string, keys authProvider, admin adminClient) (s *SSH) {
	s = new(SSH)

	s.SSH = gitkit.NewSSH(config)
	s.SSH.PublicKeyLookupFunc = s.VerifyKey
	s.SSH.PreLoginFunc = s.VerifyUser
	s.SSH.AuthoriseOperationFunc = s.Authorise

	s.keysClient = keys
	s.adminClient = admin

	s.addr = addr
	s.reposDir = "/tmp/repos"

	return
}

// VerifyUser is called as the gitkit.SSH.PreLoginFunc to verify
// the **ssh user** is correct for this connection
//
// For instance: `git clone git@example.com` - is 'git' the correct
// ssh user for this connection?
func (s *SSH) VerifyUser(ctx context.Context, metadata ssh.ConnMetadata) error {
	u := ctx.Value(gitkit.UserContextKey{})

	if u != user {
		return gitkit.ErrIncorrectUser
	}

	return nil
}

// VerifyKey calls out to the key store to verify various ssh public keys
// are known or not.
//
// The key server returns the user name attached to this key
func (s *SSH) VerifyKey(ctx context.Context, data string) (pk *gitkit.PublicKey, err error) {
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

// Authorise is called prior to git operations and is used to
// provide authorisation for those operations.
//
// For instance; if we get this far then the user name is correct for this
// connection, and the key is one we know but... is the owner of the key
// _allowed_ to work on this repo?
//
// We call out to the graphql service to check
func (s *SSH) Authorise(ctx context.Context, cmd *gitkit.GitCommand) error {
	ctxPK := ctx.Value(gitkit.PublicKeyContextKey{})
	if ctxPK == nil {
		return fmt.Errorf("missing public key")
	}

	pk, ok := ctxPK.(gitkit.PublicKey)
	if !ok {
		return fmt.Errorf("unprocessible public key")
	}

	var query adminQuery

	variables := map[string]interface{}{
		"path": removeGitSuffix(cmd.Repo),
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

// Serve implements the Server interface and exposes this Server
// on an address for SSH
func (s *SSH) Serve() error {
	return s.ListenAndServe(s.addr)
}

func removeGitSuffix(s string) string {
	return strings.Replace(s, ".git", "", -1)
}
