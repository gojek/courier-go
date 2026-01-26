# Courier Golang Client Library

[![build][build-workflow-badge]][build-workflow]
[![codecov][coverage-badge]][codecov]
[![docs][docs-badge]][pkg-dev]
[![go-report-card][report-badge]][report-card]
[![docs][docusaurus-badge]][gh-pages]
<a href="https://github.com/gojek/courier-go/graphs/commit-activity">
	<img alt="Maintenance" src="https://img.shields.io/badge/maintained-yes-green.svg" target="_blank" />
</a>
<a href="https://github.com/gojek/courier-go/releases/latest">
	<img alt="GitHub Release Date" src="https://img.shields.io/github/release-date/gojek/courier-go">
</a>
<a href="https://github.com/gojek/courier-go/commits/main">
	<img alt="GitHub last commit" src="https://img.shields.io/github/last-commit/gojek/courier-go">
</a>
[![Discord : Gojek Courier](https://img.shields.io/badge/Discord-Gojek%20Courier-blue.svg)](https://discord.gg/C823qK4AK7)

## Introduction

Courier Golang client library provides an opinionated wrapper over paho MQTT library to add features on top of it.

Head over to [Documentation][gh-pages] to get started.

Find the end-to-end courier example [here][e2e-example].

## Features

- Supports MQTT v3.1.1
- Flexible Encoder/Decoder support from Go type to MQTT payload conversion and back
- Middleware chaining
- [OpenTelemetry](https://github.com/open-telemetry/opentelemetry-go) support

## Usage

```bash
go get -u github.com/gojek/courier-go
```

### Contributing Guide

Read our [contributing guide](./CONTRIBUTING.md) to learn about our development process, how to propose bugfixes and improvements, and how to build and test your changes to Courier Go Client.

### Release Process

This repo uses Golang [`submodules`](https://github.com/golang/go/wiki/Modules#faqs--multi-module-repositories), to make a new release, make sure to follow the release process described in [RELEASING](RELEASING.md) doc exactly.

## License

Courier Go Client is [MIT licensed](./LICENSE).

[build-workflow-badge]: https://github.com/gojek/courier-go/workflows/build/badge.svg
[build-workflow]: https://github.com/gojek/courier-go/actions?query=workflow%3Abuild
[lint-workflow-badge]: https://github.com/gojek/courier-go/workflows/lint/badge.svg
[lint-workflow]: https://github.com/gojek/courier-go/actions?query=workflow%3Alint
[coverage-badge]: https://codecov.io/gh/gojek/courier-go/branch/main/graph/badge.svg?token=QPLV2ZDE84
[codecov]: https://codecov.io/gh/gojek/courier-go
[docs-badge]: https://pkg.go.dev/badge/github.com/gojek/courier-go
[pkg-dev]: https://pkg.go.dev/github.com/gojek/courier-go
[report-badge]: https://goreportcard.com/badge/github.com/gojek/courier-go
[report-card]: https://goreportcard.com/report/github.com/gojek/courier-go
[docusaurus-badge]: https://img.shields.io/badge/Docs-latest-blue.svg
[gh-pages]: https://gojek.github.io/courier-go/
[e2e-example]: https://gojek.github.io/courier/docs/Introduction
