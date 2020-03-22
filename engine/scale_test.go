package engine

import (
	"context"
	"github.com/Shuttl-Tech/drone-autoscaler/cluster"
	"github.com/Shuttl-Tech/drone-autoscaler/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/golang/mock/gomock"
	"testing"
)

func TestScale_Upscale(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	asg := mocks.NewMockAutoScalingAPI(ctrl)
	describe := asg.
		EXPECT().
		DescribeAutoScalingGroups(gomock.Any()).
		Return(&autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []*autoscaling.Group{
				{DesiredCapacity: aws.Int64(2)},
			},
		}, nil)
	asg.
		EXPECT().
		SetDesiredCapacity(&autoscaling.SetDesiredCapacityInput{
			DesiredCapacity:      aws.Int64(5),
			AutoScalingGroupName: aws.String("test-asg"),
		}).
		Return(nil, nil).
		After(describe)

	c := cluster.New(context.TODO(), "test-asg", nil, asg)
	e := &Engine{
		drone: &droneConfig{
			agent: &droneAgentConfig{cluster: c},
		},
	}

	err := e.Upscale(context.TODO(), 3)
	if err != nil {
		t.Error(err)
	}
}

func TestScale_Downscale(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var downscale *gomock.Call
	targets := []cluster.NodeId{"i-100", "i-200", "i-350abc"}

	droneClient := mocks.NewMockClient(ctrl)
	pause := droneClient.EXPECT().QueuePause().Return(nil)

	asg := mocks.NewMockAutoScalingAPI(ctrl)
	for _, t := range cluster.NodeIdsToAwsStrings(targets) {
		downscale = asg.
			EXPECT().
			TerminateInstanceInAutoScalingGroup(&autoscaling.TerminateInstanceInAutoScalingGroupInput{
				InstanceId:                     t,
				ShouldDecrementDesiredCapacity: aws.Bool(true),
			}).
			Return(nil, nil).
			After(pause)
	}

	droneClient.EXPECT().QueueResume().Return(nil).After(downscale)

	c := cluster.New(context.TODO(), "test-asg", nil, asg)
	e := &Engine{
		drone: &droneConfig{
			client: droneClient,
			agent:  &droneAgentConfig{cluster: c},
		},
	}

	err := e.Downscale(context.TODO(), targets)
	if err != nil {
		t.Error(err)
	}
}
