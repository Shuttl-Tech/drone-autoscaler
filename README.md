# drone-autoscaler
This app automatically scales Drone CI agent machines up or down to meet build capacity while optimizing for cost.

## Requirement
CI/CD in our infrastructure is handled by [Drone](https://drone.io/).

A server/master node schedules build jobs on all agent nodes. These agents are controlled by an AWS autoscaling group. This app automates upscaling & downscaling of the agents based on build traffic.

Following are the reasons we don't use [drone/autoscaler](https://github.com/drone/autoscaler):
1. As of this writing, it doesn't support AWS autoscaling groups. Instead, it creates standalone ec2 instances.
2. It installs and configures the drone agent on a newly provisioned machine itself. This is not desirable for us since we provision agent machines with custom configuration.
3. It needs to communicate to Docker daemons on all agent machines, which means we must bind `dockerd` to the `eth0` interface on these machines and expose them for reachability.
4. It has an underlying data storage layer used by all threads to coordinate. Our requirement is not that complex, so we don't need a concurrent & stateful app.
5. It  waits for builds to finish on an agent marked for destruction. We only destroy agents running 0 builds.

## Usage
- env vars, how to tune specific params
- infra assumptions (host has iam profile/permissions, agents are in aws autoscale)
- execute binary
- graceful shutdown

## Developing
- High level architecture
- setup
- test
- release

## TODO
- add versioning
- ensure aws client session is created properly (in both dev & prod env)
- write tests
- handle interrupt signal (SIGINT, SIGTERM, etc) - when signal received, run cleanup task, then shutdown gracefully
- ensure that anytime CI agent instances are fetched from AWS, we don't fetch info on Terminated instances
- handle bug where a drone build runs forever (in this case, drone.Queue() will always return some items, even though they're no longer relevant and we can downscale capacity)
- add more validations in config vars supplied by user (min, max, enum)
- cannot resume drone queue right after changing desired capacity of ASG (block until machines come up), also see if we can use autoscale cooldown (https://docs.aws.amazon.com/autoscaling/ec2/userguide/Cooldown.html)
- test whether this app will be able to handle an ephemeral pod's deployments correctly (short burst of builds, so upscale, then they finish, so downscale)
- check which objects to pass by value vs reference
- possibly refactor Plan()