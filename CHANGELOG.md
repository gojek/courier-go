# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
## [0.7.15]

### Added
- Support codec chaining ([#90] https://github.com/gojek/courier-go/pull/90)

## [0.7.14]

### Added
- Add client id attribute on otelcourier ([#84] https://github.com/gojek/courier-go/pull/84)

## [0.7.13]

### Changed
- Adding debounce default ([#86] Feat: Adding debounce default)

## [0.7.12]

### Changed
- Adding logs for consul discovered ips ([#80] https://github.com/gojek/courier-go/pull/80)
- Adding Debounce on consul ([#82]https://github.com/gojek/courier-go/pull/82)
- Fix double value issue of service instance in consul metric ([#81]https://github.com/gojek/courier-go/pull/81)
- fix round-robin issue when publishing ([#71]https://github.com/gojek/courier-go/pull/71)

## [0.7.11]

### Changed
- Bump paho version ([#78] https://github.com/gojek/courier-go/pull/78)

## [0.7.10]

### Changed
- Change synchronous gauge to observable gauge ([#76] https://github.com/gojek/courier-go/pull/76)

## [0.7.9]

### Added
- Add otelcourier metric for pool size ([#73] https://github.com/gojek/courier-go/pull/73)

## [0.7.8]

### Added
- Add observability metrics for Consul ([#66] https://github.com/gojek/courier-go/pull/64)

## [0.7.7]

### Changed
- Add connection pooling support for courier ([#66] https://github.com/gojek/courier-go/pull/66)

## [0.7.6]

### Changed
- Replace direct courier.Client cast with CourierConfig interface for courier config metrics ([#63] https://github.com/gojek/courier-go/pull/63)

## [0.7.5]

### Added
- Added metrics connection timeout, write timeout, keep-alive, ack-watchdog timeout, and library version ([#61] https://github.com/gojek/courier-go/pull/61)

## [0.7.4]

### Changed
- Added caching of address for consul resolver and sending new addresses as update only ([#59] https://github.com/gojek/courier-go/pull/59)


## [0.7.3]

### Changed
- Fix courier.client.connected metric on multiple courier instance  ([#55](https://github.com/gojek/courier-go/pull/55))
- Update paho dependency

### Added
- Add stop middleware to unregister registered callback ([#55](https://github.com/gojek/courier-go/pull/55))

## [0.7.2]
### Added

- Add `Consul` for service discovery except MQTT connection spawning  ([#51](https://github.com/gojek/courier-go/pull/51))

## [0.7.1]

### Added

- Add `ParseLogLevel` func to parse string log levels ([#50](https://github.com/gojek/courier-go/pull/50))

## [0.7.0]

### Added

- Support paho client-based logging
- Add `WithPahoLogLevel` Option to set verbosity level of paho client-based logging ([#48](https://github.com/gojek/courier-go/pull/48))

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

[Unreleased]: https://github.com/gojek/courier-go/compare/v0.7.15...HEAD
[0.7.14]: https://github.com/gojek/courier-go/releases/tag/v0.7.15
[0.7.13]: https://github.com/gojek/courier-go/releases/tag/v0.7.14
[0.7.12]: https://github.com/gojek/courier-go/releases/tag/v0.7.13
[0.7.11]: https://github.com/gojek/courier-go/releases/tag/v0.7.12
[0.7.10]: https://github.com/gojek/courier-go/releases/tag/v0.7.10
[0.7.9]: https://github.com/gojek/courier-go/releases/tag/v0.7.9
[0.7.8]: https://github.com/gojek/courier-go/releases/tag/v0.7.8
[0.7.7]: https://github.com/gojek/courier-go/releases/tag/v0.7.7
[0.7.6]: https://github.com/gojek/courier-go/releases/tag/v0.7.6
[0.7.5]: https://github.com/gojek/courier-go/releases/tag/v0.7.5
[0.7.4]: https://github.com/gojek/courier-go/releases/tag/v0.7.4
[0.7.3]: https://github.com/gojek/courier-go/releases/tag/v0.7.3
[0.7.2]: https://github.com/gojek/courier-go/releases/tag/v0.7.2
[0.7.1]: https://github.com/gojek/courier-go/releases/tag/v0.7.1
[0.7.0]: https://github.com/gojek/courier-go/releases/tag/v0.7.0
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
