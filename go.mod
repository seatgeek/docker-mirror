module github.com/seatgeek/docker-mirror

go 1.15

require (
	github.com/aws/aws-sdk-go-v2/config v1.1.1
	github.com/aws/aws-sdk-go-v2/service/ecr v1.1.1
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/docker/docker v17.12.0-ce-rc1.0.20180323031653-57c504799d97+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.6.4
	github.com/fsouza/go-dockerclient v1.3.3
	github.com/google/go-github v15.0.0+incompatible
	github.com/google/go-querystring v0.0.0-20170111101155-53e6ce116135 // indirect
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/opencontainers/selinux v1.6.0 // indirect
	github.com/ryanuber/go-glob v0.0.0-20160226084822-572520ed46db
	github.com/sirupsen/logrus v1.3.0
	github.com/stretchr/testify v1.6.1 // indirect
	golang.org/x/net v0.0.0-20200520004742-59133d7f0dd7 // indirect
	gopkg.in/yaml.v2 v2.4.0
)
