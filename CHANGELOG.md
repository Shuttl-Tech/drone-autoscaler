# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Mock tests
- `Cluster` interface
- Dependency injection in `cluster.New()` to allow passing ec2 & autoscaling sdk objects from outside.
- `cluster.NodeIdsToAwsStrings()` to convert a list of `cluster.NodeID` objects to list of `aws.String` objects

### Fixed
- Upscale if the number of running agents is less than the minimum count set by user.