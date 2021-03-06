package engine

import (
	"context"
	"encoding/json"
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

// serialization methods for better representation of Plan in logs
func (p *Plan) String() string {
	return fmt.Sprintf(
		"action=%v, upscaleCount=%v, nodesToDestroy=%v",
		p.action,
		p.upscaleCount,
		p.nodesToDestroy,
	)
}

func (p *Plan) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"action":         p.action,
		"upscaleCount":   p.upscaleCount,
		"nodesToDestroy": p.nodesToDestroy,
	})
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
	// default response is no operation (or noop)
	response := &Plan{
		action:         actionNone,
		upscaleCount:   0,
		nodesToDestroy: []cluster.NodeId{},
	}

	// let the cluster autoscale group reconcile before acting any further
	ok, err := e.drone.agent.cluster.ScalingActivityInProgress(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check for any scaling activity in progress: %v", err)
	}
	if ok {
		log.Debugln("Cluster has a scaling activity in progress, recommending noop")
		return response, nil
	}

	runningAgents, err := e.drone.agent.cluster.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch list of running agent nodes: %v", err)
	}

	runningAgentCount := len(runningAgents)
	if runningAgentCount < e.drone.agent.minCount {
		// reconcile the agent count to the minimum number to maintain
		c := e.drone.agent.minCount - runningAgentCount
		log.
			WithField("count", c).
			Info("Agent cluster size is below minimum required, recommending scale-up")

		response.action = actionUpscale
		response.upscaleCount = c
		return response, nil
	}

	stages, err := e.drone.client.Queue()
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch build queue from drone: %v", err)
	}

	// remove all builds that are pending or running for longer than their
	// maximum allowed duration
	// TODO: log stages that were discarded because they exceeded max duration
	stages = filterStages(stages, e.agedPendingBuildFilter)
	stages = filterStages(stages, e.agedRunningBuildFilter)

	pendingBuildCount, runningBuildCount := e.countBuilds(stages)
	if pendingBuildCount > 0 {
		log.
			WithField("count", pendingBuildCount).
			Debugln("Detected pending builds")

		// we need to scale up since builds are queued but not yet running
		c, err := e.calcUpscaleCount(pendingBuildCount)
		if err != nil {
			return nil, err
		}

		log.
			WithField("count", c).
			Infoln("Recommending adding more agents")

		response.action = actionUpscale
		response.upscaleCount = c
		return response, nil
	} else {
		log.Debugln("Checking for any under-utilized capacity")

		requiredAgentCount, err := e.calcRequiredAgentCount(runningBuildCount)
		if err != nil {
			return nil, err
		}
		if runningAgentCount == requiredAgentCount {
			log.Debugln("No scaling action required, recommending noop")
			return response, nil
		}

		log.
			WithField("required", requiredAgentCount).
			WithField("running", runningAgentCount).
			Debugln("Running agent count is more than required")

		busyAgents := e.listBusyAgents(stages)
		idleAgents := e.listIdleAgents(runningAgents, busyAgents)
		if len(idleAgents) < 1 {
			log.Debugln("No idle agents found, recommending noop")
			return response, nil
		}

		log.
			WithField("busy", busyAgents).
			WithField("idle", idleAgents).
			Debugln("Determined list of busy and idle agents")

		expendable, err := e.listAgentsAboveMinRetirementAge(ctx, idleAgents)
		if err != nil {
			return nil, fmt.Errorf("couldn't fetch agents above retirement age: %v", err)
		}
		if len(expendable) == 0 {
			// we have newly created agents, so they're not busy yet because it
			// might be a while before Drone starts assigning them jobs
			log.Debugln("Idle agents are not past retirement age, recommending noop")
			return response, nil
		}
		log.
			WithField("agents", expendable).
			Debugln("Found idle agents above min retirement age")

		if e.drone.agent.minCount > 0 {
			log.
				WithField("count", e.drone.agent.minCount).
				Debugln("Need to maintain a minimum number of agents in the cluster")
		}

		expendable = e.maintainMinAgentCount(runningAgents, expendable)
		if len(expendable) == 0 {
			log.Debugln("Cannot destroy agents to maintain min count, recommending noop")
			return response, nil
		}
		log.
			WithField("ids", expendable).
			Infoln("Recommending downscaling of agents")

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
		return 0, fmt.Errorf("max builds per agent cannot be %d", maxCountPerAgent)
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

func (e *Engine) maintainMinAgentCount(all, expendable []cluster.NodeId) []cluster.NodeId {
	var (
		allCount     = len(all)
		destroyCount = len(expendable)
		minCount     = e.drone.agent.minCount
	)
	if (allCount < minCount) || (allCount < destroyCount) {
		return []cluster.NodeId{}
	}
	if (allCount - destroyCount) < minCount {
		delta := minCount - (allCount - destroyCount)
		return expendable[delta:]
	}
	return expendable
}

// filter func that returns false if a build has been in pending state
// for longer than allowed duration
func (e *Engine) agedPendingBuildFilter(stage *drone.Stage) bool {
	// a negative value means no upper bound is enforced on the pending
	// build's duration of existence
	if e.drone.build.pendingMaxDuration < time.Duration(0) {
		return true
	}
	if stage.Status == drone.StatusPending {
		now := time.Now().UTC()
		upper := time.
			Unix(stage.Created, 0).
			Add(e.drone.build.pendingMaxDuration).
			UTC()
		return now.Before(upper)
	}
	return true
}

// filter func that returns false if a build has been in running state
// for longer than allowed duration
func (e *Engine) agedRunningBuildFilter(stage *drone.Stage) bool {
	// a negative value means no upper bound is enforced on the running
	// build's duration of existence
	if e.drone.build.runningMaxDuration < time.Duration(0) {
		return true
	}
	if stage.Status == drone.StatusRunning {
		now := time.Now().UTC()
		upper := time.
			Unix(stage.Started, 0).
			Add(e.drone.build.runningMaxDuration).
			UTC()
		return now.Before(upper)
	}
	return true
}
