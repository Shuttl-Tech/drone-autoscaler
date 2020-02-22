package cluster

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
)

type NodeId string

// Cluster represents an AWS autoscaling group containing Drone agents
type Cluster struct {
	client               *cloud
	autoscalingGroupName string
}

// cloud contains clients for services provided by AWS
type cloud struct {
	ec2       *ec2.EC2
	autoscale *autoscaling.AutoScaling
}

const (
	StateRunning = "running"
)

// New returns a new Cluster object
func New(asgName string) Cluster {
	sess := session.Must(session.NewSession())
	return Cluster{
		client: &cloud{
			ec2:       ec2.New(sess),
			autoscale: autoscaling.New(sess),
		},
		autoscalingGroupName: asgName,
	}
}

// Add upscales the cluster by adding the given number of instances to
// the autoscaling group
func (c Cluster) Add(ctx context.Context, count int) error {
	group, err := c.describeSelfAsg()
	if err != nil {
		return err
	}

	// calculate & set new desired capacity
	desiredCap := *group.DesiredCapacity + int64(count)
	_, err = c.client.autoscale.SetDesiredCapacity(
		&autoscaling.SetDesiredCapacityInput{
			DesiredCapacity:      aws.Int64(desiredCap),
			AutoScalingGroupName: aws.String(c.autoscalingGroupName),
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
func (c Cluster) Destroy(ctx context.Context, agents []NodeId) error {
	targets := nodeIdsToAwsStrings(agents)

	// detach target agent nodes from autoscaling group and also reduce
	// desired cap
	_, err := c.client.autoscale.DetachInstances(&autoscaling.DetachInstancesInput{
		AutoScalingGroupName:           aws.String(c.autoscalingGroupName),
		InstanceIds:                    targets,
		ShouldDecrementDesiredCapacity: aws.Bool(true),
	})
	if err != nil {
		return errors.New(
			fmt.Sprintf("failed to detach instances from autoscale group: %v", err),
		)
	}

	// destroy detached ec2 instances
	_, err = c.client.ec2.TerminateInstances(&ec2.TerminateInstancesInput{
		DryRun:      aws.Bool(false),
		InstanceIds: targets,
	})
	if err != nil {
		log.
			WithField("instances", agents).
			WithField("autoscaling-group", c.autoscalingGroupName).
			Errorln("Failed to terminate agent nodes detached from autoscaling group")
	}
	return err
}

// List returns IDs of running drone agent nodes
func (c Cluster) List(ctx context.Context, state string) ([]NodeId, error) {
	group, err := c.describeSelfAsg()
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
func (c Cluster) Describe(ctx context.Context, ids []NodeId) ([]*ec2.Instance, error) {
	agents := make([]*ec2.Instance, 0, len(ids))
	response, err := c.client.ec2.DescribeInstances(&ec2.DescribeInstancesInput{
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

// Describes the drone agent cluster's AWS autoscaling group
func (c Cluster) describeSelfAsg() (*autoscaling.Group, error) {
	response, err := c.client.autoscale.DescribeAutoScalingGroups(
		&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{aws.String(c.autoscalingGroupName)},
		},
	)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("failed to fetch info about agent autoscale group: %v", err),
		)
	}
	return response.AutoScalingGroups[0], nil
}