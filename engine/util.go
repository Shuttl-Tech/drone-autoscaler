package engine

import (
	"github.com/Shuttl-Tech/drone-autoscaler/cluster"
	"github.com/drone/drone-go/drone"
)

// StageFilter is a filter function applied to a single drone stage
type StageFilter func(stage *drone.Stage) bool

// returns the resulting list of stages after applying a StageFilter
// func to the initially passed list
// also returns the list of stages discarded
func filterStages(stages []*drone.Stage, f StageFilter) ([]*drone.Stage, []*drone.Stage) {
	res := make([]*drone.Stage, 0, len(stages))
	discarded := make([]*drone.Stage, 0, len(stages))
	for _, s := range stages {
		if f(s) {
			res = append(res, s)
		} else {
			discarded = append(discarded, s)
		}
	}
	return res, discarded
}

// returns true if the given list of Node IDs contains the target Id
func contains(arr []cluster.NodeId, subject cluster.NodeId) bool {
	for _, s := range arr {
		if s == subject {
			return true
		}
	}
	return false
}

// returns list of node IDs from the given Set of nodes
func keys(set map[cluster.NodeId]struct{}) []cluster.NodeId {
	res := make([]cluster.NodeId, 0, len(set))
	for key := range set {
		res = append(res, key)
	}
	return res
}
