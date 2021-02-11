package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/google/go-github/github"
	"github.com/ryanuber/go-glob"
	log "github.com/sirupsen/logrus"
)

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
)

// TagsResponse is Docker Registry v2 compatible struct
type TagsResponse struct {
	Count    int             `json:"count"`
	Next     *string         `json:"next"`
	Previous *string         `json:"previous"`
	Results  []RepositoryTag `json:"results"`
}

// RepositoryTag is Docker Registry v2 compatible struct, holding the indiviual
// tags for the requested repository
type RepositoryTag struct {
	Name        string    `json:"name"`
	LastUpdated time.Time `json:"last_updated"`
}

// logWriter is a io.Writer compatible wrapper, piping the outputt
// to a specific logrus entry
type logWriter struct {
	logger *log.Entry
}

func (l logWriter) Write(p []byte) (n int, err error) {
	l.logger.Debug(strings.Trim(string(p), "\n"))
	return len(p), nil
}

type mirror struct {
	dockerClient *docker.Client  // docker client used to pull, tag and push images
	ecrManager   *ecrManager     // ECR manager, used to ensure the ECR repository exist
	log          *log.Entry      // logrus logger with the relevant custom fields
	repo         Repository      // repository the mirror
	remoteTags   []RepositoryTag // list of remote repository tags (post filtering)
}

func (m *mirror) setup(repo Repository) (err error) {
	m.log = log.WithField("full_repo", repo.Name)
	m.repo = repo

	// specific tag to mirror
	if strings.Contains(repo.Name, ":") {
		chunk := strings.SplitN(repo.Name, ":", 2)
		m.repo.Name = chunk[0]
		m.repo.MatchTags = []string{chunk[1]}
	}

	// fetch remote tags
	m.remoteTags, err = m.getRemoteTags()
	if err != nil {
		return err
	}

	m.filterTags()

	m.log = m.log.WithField("repo", m.repo.Name)
	m.log = m.log.WithField("num_tags", len(m.remoteTags))
	return nil
}

// filter tags by
//  - by matching tag name (with glob support)
//  - by exluding tag name (with glob support)
//  - by tag age
//  - by max number of tags to process
func (m *mirror) filterTags() {
	now := time.Now()
	res := make([]RepositoryTag, 0)

	for _, remoteTag := range m.remoteTags {
		// match tags, with glob
		if len(m.repo.MatchTags) > 0 {
			keep := false
			for _, tag := range m.repo.MatchTags {
				if !glob.Glob(tag, remoteTag.Name) {
					m.log.Debugf("Dropping tag '%s', it doesn't match glob pattern '%s'", remoteTag.Name, tag)
					continue
				}

				keep = true
			}

			if !keep {
				continue
			}
		}

		// filter all tags what should be ignored, with glob
		if len(m.repo.DropTags) > 0 {
			keep := true
			for _, tag := range m.repo.DropTags {
				if glob.Glob(tag, remoteTag.Name) {
					m.log.Debugf("Dropping tag '%s', its ignored by glob '%s'", remoteTag.Name, tag)
					keep = false
					break
				}
			}

			if !keep {
				continue
			}
		}

		// filter on tag age
		if m.repo.MaxTagAge != nil {
			dur := time.Duration(*m.repo.MaxTagAge)
			if now.Sub(remoteTag.LastUpdated) > dur {
				m.log.Debugf("Dropping tag '%s', its older than %s", remoteTag.Name, m.repo.MaxTagAge.String())
				continue
			}
		}

		res = append(res, remoteTag)
	}

	// limit list of tags to $n newest (sorted by age by default)
	if m.repo.MaxTags > 0 && len(res) > m.repo.MaxTags {
		m.log.Debugf("Dropping %d tags, only need %d newest", len(res)-m.repo.MaxTags, m.repo.MaxTags)
		res = res[:m.repo.MaxTags]
	}

	m.remoteTags = res
}

// return the name of repostiory, as it should be on the target
// this include any target repository prefix + the repository name in DockerHub
func (m *mirror) targetRepositoryName() string {
	if m.repo.TargetPrefix != nil {
		return fmt.Sprintf("%s%s", *m.repo.TargetPrefix, m.repo.Name)
	}

	return fmt.Sprintf("%s%s", config.Target.Prefix, m.repo.Name)
}

// pull the image from remote repository to local docker agent
func (m *mirror) pullImage(tag string) error {
	m.log.Info("Starting docker pull")
	defer m.timeTrack(time.Now(), "Completed docker pull")

	pullOptions := docker.PullImageOptions{
		Repository:        m.repo.Name,
		Tag:               tag,
		InactivityTimeout: 1 * time.Minute,
		OutputStream:      &logWriter{logger: m.log.WithField("docker_action", "pull")},
	}

	authConfig := docker.AuthConfiguration{}
	if os.Getenv("DOCKERHUB_USER") != "" && os.Getenv("DOCKERHUB_PASSWORD") != "" {
		m.log.Info("Using docker hub credentials from environment")
		authConfig.Username = os.Getenv("DOCKERHUB_USER")
		authConfig.Password = os.Getenv("DOCKERHUB_PASSWORD")
	}

	return m.dockerClient.PullImage(pullOptions, authConfig)
}

// (re)tag the (local) docker image with the target repository name
func (m *mirror) tagImage(tag string) error {
	m.log.Info("Starting docker tag")
	defer m.timeTrack(time.Now(), "Completed docker tag")

	tagOptions := docker.TagImageOptions{
		Repo:  fmt.Sprintf("%s/%s", config.Target.Registry, m.targetRepositoryName()),
		Tag:   tag,
		Force: true,
	}

	return m.dockerClient.TagImage(fmt.Sprintf("%s:%s", m.repo.Name, tag), tagOptions)
}

// push the local (re)tagged image to the target docker registry
func (m *mirror) pushImage(tag string) error {
	m.log.Info("Starting docker push")
	defer m.timeTrack(time.Now(), "Completed docker push")

	pushOptions := docker.PushImageOptions{
		Name:              fmt.Sprintf("%s/%s", config.Target.Registry, m.targetRepositoryName()),
		Registry:          config.Target.Registry,
		Tag:               tag,
		OutputStream:      &logWriter{logger: m.log.WithField("docker_action", "push")},
		InactivityTimeout: 1 * time.Minute,
	}

	creds, err := getDockerCredentials(pushOptions.Registry)
	if err != nil {
		return err
	}

	return m.dockerClient.PushImage(pushOptions, *creds)
}

func (m *mirror) deleteImage(tag string) error {
	repository := fmt.Sprintf("%s:%s", m.repo.Name, tag)
	m.log.Info("Cleaning images: " + repository)
	err := m.dockerClient.RemoveImage(repository)
	if err != nil {
		return err
	}

	target := fmt.Sprintf("%s/%s:%s", config.Target.Registry, m.targetRepositoryName(), tag)
	m.log.Info("Cleaning images: " + target)
	err = m.dockerClient.RemoveImage(target)
	if err != nil {
		return err
	}

	return nil
}

func (m *mirror) work() {
	m.log.Debugf("Starting work")

	if err := m.ecrManager.ensure(m.targetRepositoryName()); err != nil {
		log.Errorf("Failed to create ECR repo %s: %s", m.targetRepositoryName(), err)
		return
	}

	for _, tag := range m.remoteTags {
		m.log = m.log.WithField("tag", tag.Name)
		m.log.Info("Start mirror tag")

		if err := m.pullImage(tag.Name); err != nil {
			m.log.Errorf("Failed to pull docker image: %s", err)
			continue
		}

		if err := m.tagImage(tag.Name); err != nil {
			m.log.Errorf("Failed to (re)tag docker image: %s", err)
			continue
		}

		if err := m.pushImage(tag.Name); err != nil {
			m.log.Errorf("Failed to push (re)tagged image: %s", err)
			continue
		}

		if config.Cleanup == true {
			if err := m.deleteImage(tag.Name); err != nil {
				m.log.Errorf("Failed to clean image: %s", err)
				continue
			}
		}

		m.log.Info("Successfully pushed (re)tagged image")
	}

	m.log.WithField("tag", "")
	m.log.Info("Repository mirror completed")
}

// get the remote tags from the remote (v2) compatible registry.
// read out the image tag and when it was updated, and sort by the updated time
func (m *mirror) getRemoteTags() ([]RepositoryTag, error) {
	if m.repo.RemoteTagSource == "github" {
		client := github.NewClient(nil)
		limit, err := strconv.Atoi(m.repo.RemoteTagConfig["num_releases"])
		if err != nil {
			return nil, fmt.Errorf("Invalid/missing int value for remote_tag_config -> num_releases")
		}

		remoteTags, _, err := client.Repositories.ListTags(context.Background(), m.repo.RemoteTagConfig["owner"], m.repo.RemoteTagConfig["repo"], &github.ListOptions{PerPage: limit})
		if err != nil {
			return nil, err
		}

		var allTags []RepositoryTag
		for _, tag := range remoteTags {
			allTags = append(allTags, RepositoryTag{
				Name: strings.TrimPrefix(*tag.Name, "v"),
			})
		}

		return allTags, nil
	}

	// docker hub
	fullRepoName := m.repo.Name
	if !strings.Contains(fullRepoName, "/") {
		fullRepoName = "library/" + m.repo.Name
	}

	var token string
	if os.Getenv("DOCKERHUB_USER") != "" && os.Getenv("DOCKERHUB_PASSWORD") != "" {
		m.log.Info("Getting tags using docker hub credentials from environment")

		message, err := json.Marshal(map[string]string{
			"username": os.Getenv("DOCKERHUB_USER"),
			"password": os.Getenv("DOCKERHUB_PASSWORD"),
		})

		if err != nil {
			return nil, err
		}

		resp, err := http.Post("https://hub.docker.com/v2/users/login/", "application/json", bytes.NewBuffer(message))
		if err != nil {
			return nil, err
		}

		var result map[string]interface{}

		json.NewDecoder(resp.Body).Decode(&result)
		token = result["token"].(string)
	}

	url := fmt.Sprintf("https://registry.hub.docker.com/v2/repositories/%s/tags/?page_size=2048", fullRepoName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("JWT %s", token))

	}
	r, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()

	var tags TagsResponse
	if err := json.NewDecoder(r.Body).Decode(&tags); err != nil {
		return nil, err
	}

	// sort the tags by updated time, newest first
	sort.Slice(tags.Results, func(i, j int) bool {
		return tags.Results[i].LastUpdated.After(tags.Results[j].LastUpdated)
	})

	return tags.Results, nil
}

// will help output how long time a function took to do its work
func (m *mirror) timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	m.log.Infof("%s in %s", name, elapsed)
}
