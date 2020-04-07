# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.2] - 2020-04-07

### Added
- Graceful shutdown upon receiving interrupt

### Removed
- `ctx` from `cluster` object

### Changed
- Refactored to use `fmt.Errorf(...)` instead of `errors.New(fmt.Sprintf(...))` 

## [1.0.1] - 2020-03-23

### Added
- Mock tests
- `Cluster` interface
- Dependency injection in `cluster.New()` to allow passing ec2 & autoscaling sdk objects from outside.
- `cluster.NodeIdsToAwsStrings()` to convert a list of `cluster.NodeID` objects to list of `aws.String` objects

### Fixed
- Upscale if the number of running agents is less than the minimum count set by user.
