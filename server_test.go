package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/hasura/go-graphql-client"
	"github.com/rotter-git/rotter-keys/server"
	"github.com/sosedoff/gitkit"
)

type dummyKSC struct {
	err bool
}

func (d dummyKSC) Lookup([]byte) (*server.User, error) {
	if d.err {
		return nil, fmt.Errorf("some error")
	}

	return &server.User{DisplayName: "test-user"}, nil
}

type dummyAdminClient struct {
	err       bool
	path      string
	finalUser string
}

func (d *dummyAdminClient) Query(ctx context.Context, q interface{}, variables map[string]interface{}, options ...graphql.Option) error {
	if d.err {
		return fmt.Errorf("some error")
	}

	d.path = variables["path"].(string)
	if d.finalUser == "" {
		d.finalUser = "test-user"
	}

	q.(*adminQuery).Repo.Namespace.Members = []struct {
		Name string
	}{
		{Name: "foo"},
		{Name: "bar"},
		{Name: "baz"},
		{Name: d.finalUser},
	}

	return nil
}

func TestSSH_VerifyUser(t *testing.T) {
	for _, test := range []struct {
		name        string
		ctx         context.Context
		expectError bool
	}{
		{"Context is empty", context.Background(), true},
		{"Context contains incorrect data", context.WithValue(context.Background(), gitkit.UserContextKey{}, 123456789), true},
		{"Context shows incorrect user", context.WithValue(context.Background(), gitkit.UserContextKey{}, "root"), true},
		{"Context shows correct user", context.WithValue(context.Background(), gitkit.UserContextKey{}, user), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := NewSSH("", dummyKSC{}, nil)

			err := s.VerifyUser(test.ctx, nil)
			if err != nil && !test.expectError {
				t.Errorf("unexpected error: %v", err)
			} else if err == nil && test.expectError {
				t.Error("expected error")
			}
		})
	}
}

func TestSSH_VerifyKey(t *testing.T) {
	for _, test := range []struct {
		name        string
		ksc         authProvider
		expectError bool
	}{
		{"Auth client errors bubble up", dummyKSC{true}, true},
		{"Known tokens don't error", dummyKSC{}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := NewSSH("", test.ksc, nil)

			_, err := s.VerifyKey(context.Background(), "some-key")
			if err != nil && !test.expectError {
				t.Errorf("unexpected error: %v", err)
			} else if err == nil && test.expectError {
				t.Error("expected error")
			}
		})
	}
}

func TestSSH_Authorise(t *testing.T) {
	for _, test := range []struct {
		name        string
		ctx         context.Context
		ac          *dummyAdminClient
		clonePath   string
		expectPath  string
		expectError bool
	}{
		{"Empty context", context.Background(), &dummyAdminClient{}, "some/repo", "", true},
		{"Incorrect context type", context.WithValue(context.Background(), gitkit.PublicKeyContextKey{}, 123456789), &dummyAdminClient{}, "some/repo", "", true},
		{"AC Errors halt authorisation", context.WithValue(context.Background(), gitkit.PublicKeyContextKey{}, gitkit.PublicKey{Name: "test-user"}), &dummyAdminClient{err: true}, "some/repo", "", true},
		{"Invalid user gets dropped", context.WithValue(context.Background(), gitkit.PublicKeyContextKey{}, gitkit.PublicKey{Name: "test-user"}), &dummyAdminClient{finalUser: "dave"}, "some/repo", "some/repo", true},

		{"Valid user gets access", context.WithValue(context.Background(), gitkit.PublicKeyContextKey{}, gitkit.PublicKey{Name: "test-user"}), &dummyAdminClient{}, "some/repo", "some/repo", false},
		{"Valid user gets access despite repo having .git suffix", context.WithValue(context.Background(), gitkit.PublicKeyContextKey{}, gitkit.PublicKey{Name: "test-user"}), &dummyAdminClient{}, "some/repo.git", "some/repo", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := NewSSH("", nil, test.ac)

			err := s.Authorise(test.ctx, &gitkit.GitCommand{
				Repo: "some/repo",
			})

			if err != nil && !test.expectError {
				t.Errorf("unexpected error: %v", err)
			} else if err == nil && test.expectError {
				t.Error("expected error")
			}

			if test.expectPath != test.ac.path {
				t.Errorf("expected %q, received %q", test.expectPath, test.ac.path)
			}
		})
	}
}

func TestSSH_Serve_Invalid_address_errors(t *testing.T) {
	// This is mainly to prove the service fails fast,
	// but also to provide a cheap test to bump up my
	// coverage stats ¯\_(ツ)_/¯

	s := NewSSH("sedfkl;dfl;kjsdf", nil, nil)

	err := s.Serve()
	if err == nil {
		t.Errorf("uhh....")
	}
}
