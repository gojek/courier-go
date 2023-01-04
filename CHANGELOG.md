# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0]

### Changed

- Handle `context.Context` deadline in `Publish`, `Subscribe` and `Unsubscribe`
  calls. ([#22](https://github.com/gojek/courier-go/pull/22))

## [0.3.1]

### Changed

- `ExponentialStartStrategy` func now takes an `interface{ Start() error }` as
  input. ([#20](https://github.com/gojek/courier-go/pull/20))

## [0.3.0]

### Changed

- `Message.DecodePayload` method is now a pointer receiver method. ([#18](https://github.com/gojek/courier-go/pull/18))

## [0.2.1]

### Added

- [`otelcourier`](./otelcourier)
  - Tracer now has public Middleware(s) so that end users can decide the order in which they get applied. 
  - New `Option(s)` are added
    - `WithTextMapPropagator`
    - `WithTextMapCarrierExtractFunc`

## [0.2.0]

### Changed

- add context to EncoderFunc/DecoderFunc signature ([#14](https://github.com/gojek/courier-go/pull/14))

## [0.1.1]

### Added

- add xds resolver ([#3](https://github.com/gojek/courier-go/pull/3))
- add support for TLS connection ([#12](https://github.com/gojek/courier-go/pull/12))

### Changed

- update Options API to retain Option type definition ([#11](https://github.com/gojek/courier-go/pull/11))

## [0.1.0]

Initial Release

[0.4.0]: https://github.com/gojek/courier-go/releases/tag/v0.4.0
[0.3.1]: https://github.com/gojek/courier-go/releases/tag/v0.3.1
[0.3.0]: https://github.com/gojek/courier-go/releases/tag/v0.3.0
[0.2.1]: https://github.com/gojek/courier-go/releases/tag/v0.2.1
[0.2.0]: https://github.com/gojek/courier-go/releases/tag/v0.2.0
[0.1.1]: https://github.com/gojek/courier-go/releases/tag/v0.1.1
[0.1.0]: https://github.com/gojek/courier-go/releases/tag/v0.1.0
