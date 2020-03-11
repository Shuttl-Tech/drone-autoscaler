# drone-autoscaler
This app scales a [Drone CI](https://drone.io/) agent cluster up or down based on build volume.

It was created because the [autoscaler provided by Drone](https://github.com/drone/autoscaler) has several limitations:
1. It doesn't support AWS's Autoscaling Groups (ASGs). Instead, it creates standalone ec2 instances. `drone-autoscaler` is designed to simply manipulate the desired capacity of the ASG that manages your Drone Agents.
2. It installs & configures an agent on a newly created machine. `drone-autoscaler` assumes that any new machine in the agent ASG is already provisioned.
3. It waits for upto 60 minutes for running builds to finish on an agent marked for termination. `drone-autoscaler` only terminates nodes that aren't running any builds.

## Usage
### Setup
This app assumes that your Drone agent cluster is managed by an AWS Autoscaling Group.

### Configuration
The app's behaviour can be configured using various parameters

| Environment variable | Required |
| --- | ---- |
| `DRONE_AGENT_MAX_BUILDS` | Yes |
| `DRONE_AGENT_AUTOSCALING_GROUP` | Yes |
| `DRONE_SERVER_HOST` | Yes |
| `DRONE_SERVER_AUTH_TOKEN` | Yes |
| `SCALER_PROBE_INTERVAL` | No |
| `SCALER_LOG_FORMAT` | No |
| `SCALER_DEBUG` | No |
| `SCALER_DRY` | No |
| `DRONE_AGENT_MIN_RETIREMENT_AGE` | No |
| `DRONE_AGENT_MIN_COUNT` | No |
| `DRONE_SERVER_PROTO` | No |
| `DRONE_BUILD_PENDING_MAX_DURATION` | No |
| `DRONE_BUILD_RUNNING_MAX_DURATION` | No |

See [config.go](config/config.go) for parameter descriptions

### Running
1. Download a pre-compiled binary from the releases page or build it from code using `make dist`.
2. Set the required configuration parameters via environment variables.
3. Run the acquired binary.

## Developing
The recommended way to run the app in development mode is to use the following configuration:
```bash
SCALER_LOG_FORMAT=text
SCALER_DEBUG=true

DRONE_BUILD_PENDING_MAX_DURATION="4h"
DRONE_BUILD_RUNNING_MAX_DURATION="1h"

# Outputs what the app plans to do, without making any changes
# to the actual infrastructure. This allows you to point the
# app to the actual agent ASG in development mode without
# worrying about accidental destruction of nodes. 
SCALER_DRY=true

# If AWS profile is configured, use these params to load creds
# and all profile configuration.
AWS_PROFILE=my_profile
AWS_SDK_LOAD_CONFIG=true
```

Run `make fmt` to format the Go code. To run tests, use `make test`.

To create a new release, bump the app version in `main` and run `make dist`.
