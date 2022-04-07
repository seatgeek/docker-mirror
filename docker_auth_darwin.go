//go:build darwin
// +build darwin

package main

import (
	"github.com/docker/docker-credential-helpers/osxkeychain"
	"github.com/fsouza/go-dockerclient"
)

var (
	keychain = osxkeychain.Osxkeychain{}
)

func getDockerCredentials(registry string) (*docker.AuthConfiguration, error) {
	user, pass, err := keychain.Get(registry)
	if err != nil {
		return nil, err
	}

	return &docker.AuthConfiguration{
		Username: user,
		Password: pass,
	}, nil
}
