package cluster

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
)

type cluster struct {
	ctx       context.Context
	asgName   string
	ec2       *ec2.EC2
	autoscale *autoscaling.AutoScaling
}

type NodeId string

// New returns a new Cluster object
func New(ctx context.Context, asgName string, ec2 *ec2.EC2, asg *autoscaling.AutoScaling) Cluster {
	return cluster{
		ctx:       ctx,
		ec2:       ec2,
		autoscale: asg,
		asgName:   asgName,
	}
}

// Add upscales the cluster by adding the given number of instances
// to the autoscaling group
func (c cluster) Add(ctx context.Context, count int) error {
	group, err := c.describeSelfAsg(ctx)
	if err != nil {
		return err
	}

	desiredCap := *group.DesiredCapacity + int64(count)
	log.
		WithField("old", *group.DesiredCapacity).
		WithField("new", desiredCap).
		Infoln("Updating desired capacity of agent autoscaling group")

	_, err = c.autoscale.SetDesiredCapacity(
		&autoscaling.SetDesiredCapacityInput{
			DesiredCapacity:      aws.Int64(desiredCap),
			AutoScalingGroupName: aws.String(c.asgName),
		},
	)
	if err != nil {
		return errors.New(
			fmt.Sprintf("failed to update autoscale group desired capacity: %v", err),
		)
	}
	return nil
}

// Destroy downscales the cluster by nuking the EC2 instances whose IDs
// are given
func (c cluster) Destroy(ctx context.Context, agents []NodeId) error {
	for _, agent := range agents {
		log.
			WithField("id", agent).
			Debugln("Terminating agent node")

		i := &autoscaling.TerminateInstanceInAutoScalingGroupInput{
			InstanceId:                     aws.String(string(agent)),
			ShouldDecrementDesiredCapacity: aws.Bool(true),
		}
		if _, err := c.autoscale.TerminateInstanceInAutoScalingGroup(i); err != nil {
			log.
				WithField("id", agent).
				Errorln("Failed to terminate agent")
			return err
		}
	}
	return nil
}

// List returns IDs of running drone agent nodes
func (c cluster) List(ctx context.Context) ([]NodeId, error) {
	group, err := c.describeSelfAsg(ctx)
	if err != nil {
		return nil, err
	}
	running := make([]NodeId, 0, len(group.Instances))
	for _, i := range group.Instances {
		if *i.HealthStatus == "Healthy" {
			running = append(running, NodeId(*i.InstanceId))
		}
	}
	return running, nil
}

// Describe returns information about agents whose IDs are given
func (c cluster) Describe(ctx context.Context, ids []NodeId) ([]*ec2.Instance, error) {
	agents := make([]*ec2.Instance, 0, len(ids))
	response, err := c.ec2.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: nodeIdsToAwsStrings(ids),
	})
	if err != nil {
		return nil, err
	}
	for _, reservation := range response.Reservations {
		agents = append(agents, reservation.Instances...)
	}
	return agents, nil
}

// ScalingActivityInProgress returns true if number of instances in
// cluster ASG is not the same as its desired capacity
func (c cluster) ScalingActivityInProgress(ctx context.Context) (bool, error) {
	group, err := c.describeSelfAsg(ctx)
	if err != nil {
		return false, err
	}
	reconciled := int(*group.DesiredCapacity) == len(group.Instances)
	return !reconciled, nil
}

// Describes the drone agent cluster's AWS autoscaling group
func (c cluster) describeSelfAsg(ctx context.Context) (*autoscaling.Group, error) {
	response, err := c.autoscale.DescribeAutoScalingGroups(
		&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{aws.String(c.asgName)},
		},
	)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("failed to fetch info about agent autoscale group: %v", err),
		)
	}
	return response.AutoScalingGroups[0], nil
}
