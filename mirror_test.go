package main

import (
	"strconv"
	"testing"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

func TestGetSleepTime(t *testing.T) {
	fakeNow := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	// Zero
	result := getSleepTime(getTimeAsString(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)), fakeNow)
	expected := 0 * time.Second
	if result != expected {
		t.Errorf("Expected %s got %s", expected, result)
	}

	// Default
	result = getSleepTime(getTimeAsString(time.Date(2021, 1, 1, 0, 0, 10, 0, time.UTC)), fakeNow)
	expected = 10 * time.Second
	if result != expected {
		t.Errorf("Expected %s got %s", expected, result)
	}

	// Random junk
	result = getSleepTime("random-string-of-rubbish", fakeNow)
	expected = 60 * time.Second
	if result != expected {
		t.Errorf("Expected %s got %s", expected, result)
	}

	// Negative
	result = getSleepTime(getTimeAsString(time.Date(2020, 12, 30, 0, 0, 10, 0, time.UTC)), fakeNow)
	expected = 0 * time.Second
	if result != expected {
		t.Errorf("Expected %s got %s", expected, result)
	}

}

type ResponseContainer struct {
	TagImageName               string
	TagImageOptions            docker.TagImageOptions
	PullImageOptions           docker.PullImageOptions
	PullImageAuthConfiguration docker.AuthConfiguration
	PushImageOptions           docker.PushImageOptions
	PushImageAuthConfiguration docker.AuthConfiguration
	RemoveImageName            string
}

type TestDockerClient struct {
	ResponseContainer *ResponseContainer
}

func (t *TestDockerClient) Info() (*docker.DockerInfo, error) {
	return &docker.DockerInfo{}, nil
}

func (t *TestDockerClient) TagImage(name string, opts docker.TagImageOptions) error {
	t.ResponseContainer.TagImageName = name
	t.ResponseContainer.TagImageOptions = opts
	return nil
}

func (t *TestDockerClient) PullImage(opts docker.PullImageOptions, authConfig docker.AuthConfiguration) error {
	t.ResponseContainer.PullImageOptions = opts
	t.ResponseContainer.PullImageAuthConfiguration = authConfig
	return nil
}

func (t *TestDockerClient) PushImage(opts docker.PushImageOptions, auth docker.AuthConfiguration) error {
	t.ResponseContainer.PushImageOptions = opts
	t.ResponseContainer.PushImageAuthConfiguration = auth
	return nil
}

func (t *TestDockerClient) RemoveImage(name string) error {
	t.ResponseContainer.RemoveImageName = name
	return nil
}

func CreateTestDockerClient(responseContainer *ResponseContainer) *TestDockerClient {
	return &TestDockerClient{ResponseContainer: responseContainer}
}

func TestPullImage(t *testing.T) {
	responseContainer := &ResponseContainer{}
	var client DockerClient
	client = CreateTestDockerClient(responseContainer)
	repo := Repository{
		PrivateRegistry: "private-registry/",
		Name:            "elasticsearch",
	}

	m := mirror{
		dockerClient: &client,
	}
	m.setup(repo)
	m.pullImage("latest")

	got := responseContainer.PullImageOptions.Repository
	want := "private-registry/elasticsearch"

	if got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
}

func getTimeAsString(date time.Time) string {
	return strconv.FormatInt(date.Unix(), 10)
}
