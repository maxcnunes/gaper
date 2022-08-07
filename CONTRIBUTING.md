# Contributing to httpfake

:+1::tada: First off, thanks for taking the time to contribute! :tada::+1:

There are few ways of contributing to gaper

* Report an issue.
* Contribute to the code base.

## Report an issue

* Before opening the issue make sure there isn't an issue opened for the same problem
* Include the Go and Gaper version you are using
* If it is a bug, please include all info to reproduce the problem

## Contribute to the code base

### Pull Request

* Please discuss the suggested changes on a issue before working on it. Just to make sure the change makes sense before you spending any time on it.

### Setupping development

```
make setup
```

### Running gaper in development

```
make build && \
	./gaper \
	--verbose \
	--bin-name srv \
	--build-path ./testdata/server \
	--build-args="-ldflags=\"-X 'main.Version=v1.0.0'\"" \
	--extensions "go,txt"
```

### Running lint

```
make lint
```

### Running tests

All tests:
```
make test
```

A single test:
```
go test -run TestSimplePost ./...
```

