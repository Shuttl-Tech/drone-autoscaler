package engine

import (
	"context"
	"errors"
	"fmt"
	"github.com/Shuttl-Tech/drone-autoscaler/cluster"
	"github.com/drone/drone-go/drone"
	log "github.com/sirupsen/logrus"
	"math"
	"time"
)

// Plan describes the scaling action that needs to be taken, as determined
// by autoscaler's planner engine. It also supplies the data required to
// carry out the action.
type Plan struct {
	action         string
	upscaleCount   int
	nodesToDestroy []cluster.NodeId
}

// RequiresUpscaling returns true when more agents must be added
func (p *Plan) RequiresUpscaling() bool {
	return p.action == actionUpscale
}

// RequiresDownscaling returns true when we have extra compute capacity
// that must be shed
func (p *Plan) RequiresDownscaling() bool {
	return p.action == actionDownscale
}

// UpscaleCount returns the number of nodes to add to the agent cluster
// when upscaling
func (p *Plan) UpscaleCount() int {
	return p.upscaleCount
}

// NodesToDestroy returns IDs of agent machines to destroy when downscaling
func (p *Plan) NodesToDestroy() []cluster.NodeId {
	return p.nodesToDestroy
}

// Plan determines whether there is a need to upscale or downscale the agent
// cluster based on current capacity and build traffic
func (e *Engine) Plan(ctx context.Context) (*Plan, error) {
	response := &Plan{
		action:         actionNone,
		upscaleCount:   0,
		nodesToDestroy: []cluster.NodeId{},
	}

	stages, err := e.drone.client.Queue()
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("couldn't fetch build queue from drone: %v", err),
		)
	}

	pendingBuildCount, runningBuildCount := e.countBuilds(stages)
	if pendingBuildCount > 0 {
		// we need to scale up since builds are queued but not yet running
		c, err := e.calcUpscaleCount(pendingBuildCount)
		if err != nil {
			return nil, err
		}
		response.action = actionUpscale
		response.upscaleCount = c
		return response, nil
	} else {
		// we need to determine whether we have under-utilized capacity
		runningAgents, err := e.drone.agent.cluster.List(ctx, cluster.StateRunning)
		if err != nil {
			return nil, errors.New(
				fmt.Sprintf("couldn't fetch list of running agent nodes: %v", err),
			)
		}

		runningAgentCount := len(runningAgents)
		requiredAgentCount, err := e.calcRequiredAgentCount(runningBuildCount)
		if err != nil {
			return nil, err
		}

		if runningAgentCount == requiredAgentCount {
			// we optimally meet the build load, no scaling action is required
			return response, nil
		}

		busyAgents := e.listBusyAgents(stages)
		idleAgents := e.listIdleAgents(runningAgents, busyAgents)

		expendable, err := e.listAgentsAboveMinRetirementAge(ctx, idleAgents)
		if err != nil {
			return nil, errors.New(
				fmt.Sprintf("couldn't fetch agents above retirement age: %v", err),
			)
		}
		if len(expendable) == 0 {
			// we have newly created agent nodes, so even though they're not busy,
			// we can't destroy them because they haven't reached min retirement
			// age yet
			return response, nil
		}

		// we have extra capacity and must downscale
		log.
			WithField("count", len(expendable)).
			Infoln("Extra agent nodes detected")

		response.action = actionDownscale
		response.nodesToDestroy = expendable
		return response, nil
	}
}

// Returns the number of pending & running builds from given drone stages
func (e *Engine) countBuilds(stages []*drone.Stage) (pending, running int) {
	for _, stage := range stages {
		switch stage.Status {
		case drone.StatusRunning:
			running++
		case drone.StatusPending:
			pending++
		}
	}
	return
}

// Calculates number of agents required to run given number of builds.
func (e *Engine) calcRequiredAgentCount(buildCount int) (int, error) {
	maxCountPerAgent := e.drone.agent.maxBuilds
	if maxCountPerAgent < 1 {
		return 0, errors.New(
			fmt.Sprintf("max builds per agent cannot be %d", maxCountPerAgent),
		)
	}
	res := math.Ceil(float64(buildCount) / float64(maxCountPerAgent))
	return int(res), nil
}

// Calculates the number of agents to add to run pending builds.
// This method simply wraps around calcRequiredAgentCount() to provide
// a cleaner abstraction.
func (e *Engine) calcUpscaleCount(pendingBuildCount int) (int, error) {
	return e.calcRequiredAgentCount(pendingBuildCount)
}

// Returns a list of agents that are currently running 1 or more builds
func (e *Engine) listBusyAgents(stages []*drone.Stage) []cluster.NodeId {
	// because one agent can have multiple builds, we must maintain a
	// Set of IDs in order to return only unique IDs in the resultant
	// list
	set := make(map[cluster.NodeId]struct{})
	for _, stage := range stages {
		if stage.Status == drone.StatusRunning {
			set[cluster.NodeId(stage.Machine)] = struct{}{}
		}
	}
	return keys(set)
}

// Returns list of agents that are currently running 0 builds
// TODO: optimize
//  This method has a complexity of O(N^2) where N = total no.
//  of drone agents. We can take a map approach to make it O(N).
func (e *Engine) listIdleAgents(all, busy []cluster.NodeId) []cluster.NodeId {
	res := make([]cluster.NodeId, 0, len(all))
	for _, subject := range all {
		if !contains(busy, subject) {
			res = append(res, subject)
		}
	}
	return res
}

func (e *Engine) listAgentsAboveMinRetirementAge(ctx context.Context, ids []cluster.NodeId) (
	[]cluster.NodeId,
	error,
) {
	now := time.Now().UTC()
	age := e.drone.agent.minRetirementAge
	filtered := make([]cluster.NodeId, 0, len(ids))

	agents, err := e.drone.agent.cluster.Describe(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, agent := range agents {
		if now.After(agent.LaunchTime.Add(age)) {
			filtered = append(filtered, cluster.NodeId(*agent.InstanceId))
		}
	}
	return filtered, nil
}