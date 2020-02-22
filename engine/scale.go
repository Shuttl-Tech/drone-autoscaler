package engine

import (
	"context"
	"errors"
	"fmt"
	"github.com/Shuttl-Tech/drone-autoscaler/cluster"
)

const (
	actionNone      = "noop"
	actionUpscale   = "upscale"
	actionDownscale = "downscale"
)

func (e *Engine) Upscale(ctx context.Context, count int) error {
	if err := e.drone.client.QueuePause(); err != nil {
		return errors.New(
			fmt.Sprintf("couldn't pause drone queue while upscaling: %v", err),
		)
	}
	defer e.resumeBuildQueue()
	return e.drone.agent.cluster.Add(ctx, count)
}

func (e *Engine) Downscale(ctx context.Context, agents []cluster.NodeId) error {
	if err := e.drone.client.QueuePause(); err != nil {
		return errors.New(
			fmt.Sprintf("couldn't pause drone queue while downscaling: %v", err),
		)
	}
	defer e.resumeBuildQueue()
	return e.drone.agent.cluster.Destroy(ctx, agents)
}

// resumeBuildQueue attempts to resume Drone's build queue
func (e *Engine) resumeBuildQueue() {
	// failing to resume is catastrophic because all builds will remain
	// stuck if the queue was previously paused, so the app must fail
	// immediately and queue must be resumed manually before re-starting it.
	if err := e.drone.client.QueueResume(); err != nil {
		panic(
			errors.New(fmt.Sprintf("failed to resume build queue: %v", err)),
		)
	}
}
