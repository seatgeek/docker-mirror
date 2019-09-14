# docker-mirror

This project will copy public DockerHub repositories to a private registry.

It's possible to filter by docker tags, tag age and number of latest tags.

<!-- TOC -->

- [docker-mirror](#docker-mirror)
    - [Install / Building](#install-building)
    - [Using](#using)
        - [Adding new mirror repository](#adding-new-mirror-repository)
        - [Updating / resync an existing repository](#updating-resync-an-existing-repository)
        - [Update all repositories](#update-all-repositories)
    - [Example config.yaml](#example-configyaml)

<!-- /TOC -->

## Install / Building

- make sure you got Go 1.10 or newer
  - OSX: `brew install go`
- make sure you got [`godep`](https://github.com/golang/dep) installed
  - OSX: `brew install dep`
- make sure you have `CGO` enabled
  - `export CGO_ENABLED=1`
- clone this repository to `$HOME/src/github.com/seatgeek/docker-mirror`
- change your working directory to `$HOME/go/src/github.com/seatgeek/docker-mirror`
- run `dep ensure -vendor-only` to install dependencies
- run `go install` to build and install the `docker-mirror` binary into your `$HOME/go/bin/` directory
  - alternative: `go build` to build the binary and put it in the current working directory

## Using

Make sure that your local Docker agent is logged into to `ECR` (`eval $(aws ecr get-login --no-include-email --region us-east-1)`)

`docker-mirror` will automatically create the ECR repository on demand, so you do not need to login and do any UI operations in the AWS Console.

`docker-mirror` will look for your AWS credentials in all the default locations (`env`, `~/.aws/` and so forth like normal AWS tools do)

### Adding new mirror repository

- add the new repository to the `config.yaml` file
  - TIP: omit the `max_tag_age` for the initial sync to mirror all historic tags (`match_tag` is fine to use in all cases)
- run `PREFIX=${reopsitory_name} docker-mirror` to trigger a sync for the specific new repository (you probably don't want to sync all the existing repositories)
- add the `max_tag_age` filter to the newly added repository so future syns won't cosider all historic tags

### Updating / resync an existing repository

- run `PREFIX=${reopsitory_name} docker-mirror` to trigger a sync for the specific repository
  - TIP: Consider if the tags you want to sync fits within the `max_tag_age` and other filters

### Update all repositories

- run `docker-mirror` and wait (for a while)

## Example config.yaml

```yml
---
target:
  # where to copy images to
  registry: ACCOUNT_ID.dkr.REGION.amazonaws.com

  # (optional) prefix all repositories with this name
  # ACCOUNT_ID.dkr.REGION.amazonaws.com/hub/jippi/hashi-ui
  prefix: "hub/"

# what repositories to copy
repositories:
    # will automatically know it's a "library" repository in dockerhub
  - name: elasticsearch
    match_tag: # tags to match, can be specific or glob pattern
      - "5.6.8" # specific tag match
      - "6.*"   # glob patterns will match
    ignore_tag: # tags to never match on (even if its matched by `tag`)
      - "*-alpine" # support both glob or specific strings

  - name: yotpo/resec
    max_tag_age: 8w # only import tags that are 8w or less old

  - name: jippi/hashi-ui
    max_tags: 10 # only copy the 10 latest tags
    match_tag:
      - "v*"

  - name: jippi/go-metadataproxy # import all tags

  - name: docker.elastic.co/kibana/kibana-oss # The image is not hosted on Docker Hub
    match_tag:
      - "6.4.3"
    skip_remote_tags: true # Do not fetch the tags from the registry, use the content of match_tag as existing tags
```

## Environment Variables

Environment Variable  |  Default       | Description
----------------------| ---------------| -------------------------------------------------
CONFIG_FILE           | config.yaml    | config file to use
