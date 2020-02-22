package config

import (
	"github.com/kelseyhightower/envconfig"
	"time"
)

type Config struct {
	// The time interval between 2 consecutive runs of the autoscaler.
	// Value can be any string parseable by time.ParseDuration()
	ProbeInterval time.Duration `default:"30s" split_words:"true"`
	Debug         bool          `default:"false"`

	Agent struct {
		// Minimum amount of time for which an Agent node should've been
		// up before it can be considered for destruction during downscaling.
		// This avoids accidentally deleting an agent that's not running
		// any workloads only because it was provisioned very recently
		MinRetirementAge time.Duration `envconfig:"DRONE_AGENT_MIN_RETIREMENT_AGE" default:"10m"`

		// Max number of builds that can run on an agent at any point
		// of time
		MaxBuilds int `envconfig:"DRONE_AGENT_MAX_BUILDS" required:"true"`

		// Name of the AWS autoscaling group containing agent nodes
		AutoscalingGroup string `envconfig:"DRONE_AGENT_AUTOSCALING_GROUP" required:"true"`
	}

	// Information about the Drone server the app will talk to
	Server struct {
		Proto     string `envconfig:"DRONE_SERVER_PROTO" default:"http"`
		Host      string `envconfig:"DRONE_SERVER_HOST" required:"true"`
		AuthToken string `envconfig:"DRONE_SERVER_AUTH_TOKEN" required:"true"`
	}
}

func Load() (Config, error) {
	conf := Config{}
	err := envconfig.Process("SCALER", &conf)
	return conf, err
}