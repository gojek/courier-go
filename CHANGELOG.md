# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.6.1]

- Fix WriteTimeout not working

## [0.6.0]

### Added

- [`otelcourier`](./otelcourier) Add OpenTelemetry Metrics support ([#42](https://github.com/gojek/courier-go/pull/42))

## [0.5.3]

### Added

- Paho's `mqtt.CredentialsProvider` is now used when `CredentialFetcher` is provided. ([#40](https://github.com/gojek/courier-go/pull/40))

### Changed

- De-duplicate subscription calls when using OnConnectHandler to avoid concurrent subscribe issues. ([#39](https://github.com/gojek/courier-go/pull/39))

## [0.5.2]

### Added

- Add `ConnectRetryInterval` Option to allow users to configure the interval between connection retries.

### Changed

- Update multi-connection mode connect logic ([#37](https://github.com/gojek/courier-go/pull/37))

## [0.5.1]

### Added

- Add a typed `KeepAlive` Option.

### Changed

- Add logging inside `OnConnectionLostHandler` & `OnReconnectHandler` Handlers.

## [0.5.0]

### Added

- Allow connection to multiple address simultaneously ([#31](https://github.com/gojek/courier-go/pull/31))
- add `CredentialFetcher` to allow updating credentials on each newOptions creation ([#28](https://github.com/gojek/courier-go/pull/28))
- add `WithExponentialStartOptions` ClientOption ([#29](https://github.com/gojek/courier-go/pull/29))

### Changed

- add revision counter to avoid client-id clashes ([#32](https://github.com/gojek/courier-go/pull/32))
- Handle client init errors ([#30](https://github.com/gojek/courier-go/pull/30))
- update dependencies ([#26](https://github.com/gojek/courier-go/pull/26))

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

[Unreleased]: https://github.com/gojek/courier-go/compare/v0.6.0...HEAD
[0.6.1]: https://github.com/gojek/courier-go/releases/tag/v0.6.1
[0.6.0]: https://github.com/gojek/courier-go/releases/tag/v0.6.0
[0.5.3]: https://github.com/gojek/courier-go/releases/tag/v0.5.3
[0.5.2]: https://github.com/gojek/courier-go/releases/tag/v0.5.2
[0.5.1]: https://github.com/gojek/courier-go/releases/tag/v0.5.1
[0.5.0]: https://github.com/gojek/courier-go/releases/tag/v0.5.0
[0.4.0]: https://github.com/gojek/courier-go/releases/tag/v0.4.0
[0.3.1]: https://github.com/gojek/courier-go/releases/tag/v0.3.1
[0.3.0]: https://github.com/gojek/courier-go/releases/tag/v0.3.0
[0.2.1]: https://github.com/gojek/courier-go/releases/tag/v0.2.1
[0.2.0]: https://github.com/gojek/courier-go/releases/tag/v0.2.0
[0.1.1]: https://github.com/gojek/courier-go/releases/tag/v0.1.1
[0.1.0]: https://github.com/gojek/courier-go/releases/tag/v0.1.0
