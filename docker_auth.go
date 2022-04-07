//go:build !darwin
// +build !darwin

package main

import (
	"fmt"

	"github.com/fsouza/go-dockerclient"
	log "github.com/sirupsen/logrus"
)

func getDockerCredentials(registry string) (*docker.AuthConfiguration, error) {
	authOptions, err := docker.NewAuthConfigurationsFromDockerCfg()
	if err != nil {
		log.Fatal(err)
	}

	creds, ok := authOptions.Configs[registry]
	if !ok {
		return nil, fmt.Errorf("No auth found for %s", registry)
	}

	return &creds, nil
}
