package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	"github.com/cenkalti/backoff"
	log "github.com/sirupsen/logrus"
)

type ecrPublicManager struct {
	client       *ecrpublic.Client // AWS public ECR client
	repositories map[string]bool   // list of repositories in public ECR
}

func (e *ecrPublicManager) exists(name string) bool {
	if _, ok := e.repositories[name]; ok {
		return true
	}

	return false
}

func (e *ecrPublicManager) ensure(name string) error {
	if e.exists(name) {
		return nil
	}

	return e.create(name)
}

func (e *ecrPublicManager) create(name string) error {
	_, err := e.client.CreateRepository(context.TODO(), &ecrpublic.CreateRepositoryInput{
		RepositoryName: &name,
	})
	if err != nil {
		return err
	}

	e.repositories[name] = true
	return nil
}

func (e *ecrPublicManager) buildCache(nextToken *string) error {
	if nextToken == nil {
		log.Info("Loading the list of ECR public repositories")
	}

	resp, err := e.client.DescribeRepositories(context.TODO(), &ecrpublic.DescribeRepositoriesInput{
		NextToken: nextToken,
	})
	if err != nil {
		return err
	}

	if e.repositories == nil {
		e.repositories = make(map[string]bool)
	}

	for _, repo := range resp.Repositories {
		e.repositories[*repo.RepositoryName] = true
	}

	// keep paging as long as there is a token for the next page
	if resp.NextToken != nil {
		err := e.buildCache(resp.NextToken)
		if err != nil {
			return err
		}
	}

	// no next token means we hit the last page
	if nextToken == nil {
		log.Info("Done loading ECR public repositories")
	}

	return nil
}

func (e *ecrPublicManager) buildCacheBackoff() backoff.Operation {
	return func() error {
		return e.buildCache(nil)
	}
}
