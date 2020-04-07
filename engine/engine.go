package engine

import (
	"context"
	"github.com/Shuttl-Tech/drone-autoscaler/cluster"
	"github.com/Shuttl-Tech/drone-autoscaler/config"
	"github.com/drone/drone-go/drone"
	log "github.com/sirupsen/logrus"
	"time"
)

type droneBuildConfig struct {
	pendingMaxDuration time.Duration
	runningMaxDuration time.Duration
}

type droneAgentConfig struct {
	maxBuilds        int
	minCount         int
	minRetirementAge time.Duration
	cluster          cluster.Cluster
}

type droneConfig struct {
	client drone.Client
	build  *droneBuildConfig
	agent  *droneAgentConfig
}

type Engine struct {
	dry           bool
	drone         *droneConfig
	probeInterval time.Duration
}

func New(c config.Config, client drone.Client, fleet cluster.Cluster) *Engine {
	return &Engine{
		dry: c.Dry,
		drone: &droneConfig{
			client: client,
			build: &droneBuildConfig{
				pendingMaxDuration: c.Build.PendingMaxDuration,
				runningMaxDuration: c.Build.RunningMaxDuration,
			},
			agent: &droneAgentConfig{
				cluster:          fleet,
				minCount:         c.Agent.MinCount,
				maxBuilds:        c.Agent.MaxBuilds,
				minRetirementAge: c.Agent.MinRetirementAge,
			},
		},
		probeInterval: c.ProbeInterval,
	}
}

func (e *Engine) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Infoln("Shutting down gracefully")
			return

		case <-time.After(e.probeInterval):
			plan, err := e.Plan(ctx)
			if err != nil {
				log.WithError(err).Errorln("Failed to create scaling plan")
				continue
			}

			if e.dry {
				log.
					WithField("plan", plan).
					Infoln("Final plan generated")
				log.Infoln("Dry mode is enabled, no further action will be taken")
				continue
			}

			if plan.RequiresUpscaling() {
				if err = e.Upscale(ctx, plan.UpscaleCount()); err != nil {
					log.WithError(err).Errorln("Failed to upscale")
				}
			} else if plan.RequiresDownscaling() {
				if err = e.Downscale(ctx, plan.NodesToDestroy()); err != nil {
					log.WithError(err).Errorln("Failed to downscale")
				}
			}
		}
	}
}
