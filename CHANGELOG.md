# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial public release as a Go library
- BLE client for connecting to GoCube smart cubes
- Real-time move tracking with timestamps
- Cube state simulation and phase detection
- Orientation tracking via quaternion conversion
- Analysis algorithms for solve performance
- CLI application for recording and analyzing solves

### Changed
- Restructured project as a public library with `package gocube`
- Public API exposed at root package level
- Application code moved to `internal/` and `cmd/`

## [0.1.0] - 2024-XX-XX

### Added
- Initial library release
- Support for GoCube device discovery and connection
- Move decoding from BLE rotation messages
- Battery level and orientation monitoring
- Cube state tracking with phase detection
- Solve recording with SQLite storage
- Report generation with analysis metrics

[Unreleased]: https://github.com/SeamusWaldron/gocube/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/SeamusWaldron/gocube/releases/tag/v0.1.0
