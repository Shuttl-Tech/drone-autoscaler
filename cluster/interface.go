package cluster

import (
	"context"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Cluster is used to communicate with a Drone agent cluster managed
// by an AWS autoscaling group.
type Cluster interface {
	// Add upscales the cluster by adding the given number of instances
	// to the autoscaling group
	Add(ctx context.Context, count int) error

	// Destroy downscales the cluster by nuking the EC2 instances whose IDs
	// are given
	Destroy(ctx context.Context, agents []NodeId) error

	// List returns IDs of running drone agent nodes
	List(ctx context.Context) ([]NodeId, error)

	// Describe returns information about agents whose IDs are given
	Describe(ctx context.Context, ids []NodeId) ([]*ec2.Instance, error)

	// ScalingActivityInProgress returns true if number of instances in
	// cluster ASG is not the same as its desired capacity
	ScalingActivityInProgress(ctx context.Context) (bool, error)
}
