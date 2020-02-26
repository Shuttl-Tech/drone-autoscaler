package config

import (
	"encoding/json"
	"github.com/kr/pretty"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	setEnvVars(required)
	defer unsetEnvVars(required)

	conf, err := Load()
	if err != nil {
		t.Fatalf("Did not expect loader error: %v", err)
	}

	if got, want := conf.ProbeInterval, time.Second*30; got != want {
		t.Errorf("Want default probe interval %v, got %v", want, got)
	}
	if got, want := conf.LogFormat, "json"; got != want {
		t.Errorf("Want default log format %v, got %v", want, got)
	}
	if got, want := conf.Debug, false; got != want {
		t.Errorf("Want default debug mode %v, got %v", want, got)
	}
	if got, want := conf.Agent.MinRetirementAge, time.Minute*10; got != want {
		t.Errorf("Want default minimum retirement age of agent %v, got %v", want, got)
	}
	if got, want := conf.Agent.MinCount, 1; got != want {
		t.Errorf("Want default minimum agent count %v, got %v", want, got)
	}
	if got, want := conf.Server.Proto, "http"; got != want {
		t.Errorf("Want default drone server protocl %v, got %v", want, got)
	}
}

func TestLoad(t *testing.T) {
	setEnvVars(required)
	defer unsetEnvVars(required)

	setEnvVars(optional)
	defer unsetEnvVars(optional)

	a, _ := Load()
	b := Config{}
	err := json.Unmarshal(jsonConfig, &b)
	if err != nil {
		t.Fatalf("Didn't expect json deserialization to fail: %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Error("Configuration mismatch")
		pretty.Ldiff(t, a, b)
	}
}

func setEnvVars(vars map[string]string) {
	for k, v := range vars {
		os.Setenv(k, v)
	}
}

func unsetEnvVars(vars map[string]string) {
	for k := range vars {
		os.Unsetenv(k)
	}
}

var required = map[string]string{
	"DRONE_AGENT_MAX_BUILDS":        "10",
	"DRONE_SERVER_HOST":             "drone.company.com",
	"DRONE_SERVER_AUTH_TOKEN":       "1234567890abcdxyz",
	"DRONE_AGENT_AUTOSCALING_GROUP": "ci-agent-cluster",
}

var optional = map[string]string{
	"SCALER_PROBE_INTERVAL":          "5m",
	"SCALER_LOG_FORMAT":              "text",
	"SCALER_DEBUG":                   "true",
	"DRONE_SERVER_PROTO":             "https",
	"DRONE_AGENT_MIN_COUNT":          "3",
	"DRONE_AGENT_MIN_RETIREMENT_AGE": "25m",
}

var jsonConfig = []byte(`{
  "ProbeInterval": 300000000000,
  "LogFormat": "text",
  "Debug": true,
  "Agent": {
    "MinRetirementAge": 1500000000000,
    "MaxBuilds": 10,
    "MinCount": 3,
    "AutoscalingGroup": "ci-agent-cluster"
  },
  "Server": {
    "Proto": "https",
    "Host": "drone.company.com",
    "AuthToken": "1234567890abcdxyz"
  }
}`)
