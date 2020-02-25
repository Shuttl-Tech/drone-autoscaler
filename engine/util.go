package engine

import "github.com/Shuttl-Tech/drone-autoscaler/cluster"

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
	for key, _ := range set {
		res = append(res, key)
	}
	return res
}
