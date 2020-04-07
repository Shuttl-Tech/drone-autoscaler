package engine

import (
	"context"
	"github.com/Shuttl-Tech/drone-autoscaler/cluster"
	"github.com/Shuttl-Tech/drone-autoscaler/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/drone/drone-go/drone"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

// Verifies that planner recommends noop when a scaling activity
// is in progress in the agent pool.
func TestPlan_ScalingInProgress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	asg := mocks.NewMockAutoScalingAPI(ctrl)
	asg.
		EXPECT().
		DescribeAutoScalingGroups(gomock.Any()).
		Return(&autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []*autoscaling.Group{
				{
					Instances: []*autoscaling.Instance{
						{InstanceId: aws.String("i-009eed7816")},
					},
					DesiredCapacity: aws.Int64(3),
				},
			},
		}, nil)

	c := cluster.New("test-asg", nil, asg)
	e := &Engine{
		drone: &droneConfig{
			agent: &droneAgentConfig{cluster: c},
		},
	}

	p, err := e.Plan(context.TODO())
	if err != nil {
		t.Error(err)
	}
	if p.action != actionNone {
		t.Errorf("Want plan noop, got %v", p)
	}
}

// Verifies that planner recommends noop when extra agents are
// below minimum retirement age.
func TestPlan_BelowMinRetirement(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	asg := mocks.NewMockAutoScalingAPI(ctrl)
	asg.
		EXPECT().
		DescribeAutoScalingGroups(gomock.Any()).
		Return(&autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []*autoscaling.Group{
				{
					Instances: []*autoscaling.Instance{
						{
							HealthStatus: aws.String("Healthy"),
							InstanceId:   aws.String("i-001"),
						},
						{
							HealthStatus: aws.String("Healthy"),
							InstanceId:   aws.String("i-002"),
						},
					},
					DesiredCapacity: aws.Int64(2),
				},
			},
		}, nil).
		Times(2)

	ec2Client := mocks.NewMockEC2API(ctrl)
	ec2Client.
		EXPECT().
		DescribeInstances(gomock.Any()).
		Return(&ec2.DescribeInstancesOutput{
			Reservations: []*ec2.Reservation{
				{
					Instances: []*ec2.Instance{
						// idle but below min retirement
						{
							InstanceId: aws.String("i-002"),
							LaunchTime: aws.Time(time.Now().UTC().Add(-4 * time.Minute)),
						},
					},
				},
			},
		}, nil)

	droneClient := mocks.NewMockClient(ctrl)
	droneClient.
		EXPECT().
		Queue().
		Return([]*drone.Stage{
			{Status: drone.StatusRunning, Machine: "i-001"},
			{Status: drone.StatusRunning, Machine: "i-001"},
		}, nil)

	c := cluster.New("test-asg", ec2Client, asg)
	e := &Engine{
		drone: &droneConfig{
			build: &droneBuildConfig{
				pendingMaxDuration: -1 * time.Second,
				runningMaxDuration: -1 * time.Second,
			},
			agent: &droneAgentConfig{
				cluster:          c,
				maxBuilds:        2,
				minRetirementAge: 10 * time.Minute,
			},
			client: droneClient,
		},
	}

	p, err := e.Plan(context.TODO())
	if err != nil {
		t.Error(err)
	}
	if p.action != actionNone {
		t.Errorf("Want plan noop, got %v", p)
	}
}

// Verifies that planner recommends noop when extra agents exist
// but the minimum agent count needs to be maintained.
func TestPlan_MinAgentCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	asg := mocks.NewMockAutoScalingAPI(ctrl)
	asg.
		EXPECT().
		DescribeAutoScalingGroups(gomock.Any()).
		Return(&autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []*autoscaling.Group{
				{
					Instances: []*autoscaling.Instance{
						{
							HealthStatus: aws.String("Healthy"),
							InstanceId:   aws.String("i-001"),
						},
					},
					DesiredCapacity: aws.Int64(1),
				},
			},
		}, nil).
		Times(2)

	ec2Client := mocks.NewMockEC2API(ctrl)
	ec2Client.
		EXPECT().
		DescribeInstances(gomock.Any()).
		Return(&ec2.DescribeInstancesOutput{
			Reservations: []*ec2.Reservation{
				{
					Instances: []*ec2.Instance{
						// idle & past min retirement
						{
							InstanceId: aws.String("i-001"),
							LaunchTime: aws.Time(time.Now().UTC().Add(-20 * time.Minute)),
						},
					},
				},
			},
		}, nil)

	droneClient := mocks.NewMockClient(ctrl)
	droneClient.
		EXPECT().
		Queue().
		Return([]*drone.Stage{}, nil)

	c := cluster.New("test-asg", ec2Client, asg)
	e := &Engine{
		drone: &droneConfig{
			build: &droneBuildConfig{
				pendingMaxDuration: -1 * time.Second,
				runningMaxDuration: -1 * time.Second,
			},
			agent: &droneAgentConfig{
				cluster:          c,
				maxBuilds:        10,
				minRetirementAge: 10 * time.Minute,
				minCount:         1,
			},
			client: droneClient,
		},
	}

	p, err := e.Plan(context.TODO())
	if err != nil {
		t.Error(err)
	}
	if p.action != actionNone {
		t.Errorf("Want plan noop, got %v", p)
	}
}

// Verifies that planner recommends noop when demand is met exactly
// and no extra agents exist.
func TestPlan_NoExtra(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	asg := mocks.NewMockAutoScalingAPI(ctrl)
	asg.
		EXPECT().
		DescribeAutoScalingGroups(gomock.Any()).
		Return(&autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []*autoscaling.Group{
				{
					Instances: []*autoscaling.Instance{
						{
							HealthStatus: aws.String("Healthy"),
							InstanceId:   aws.String("i-009eed7816"),
						},
					},
					DesiredCapacity: aws.Int64(1),
				},
			},
		}, nil).
		Times(2)

	droneClient := mocks.NewMockClient(ctrl)
	droneClient.
		EXPECT().
		Queue().
		Return([]*drone.Stage{
			{Status: drone.StatusRunning},
			{Status: drone.StatusRunning},
		}, nil)

	c := cluster.New("test-asg", nil, asg)
	e := &Engine{
		drone: &droneConfig{
			client: droneClient,
			build: &droneBuildConfig{
				pendingMaxDuration: -1 * time.Second,
				runningMaxDuration: -1 * time.Second,
			},
			agent: &droneAgentConfig{cluster: c, maxBuilds: 2},
		},
	}

	p, err := e.Plan(context.TODO())
	if err != nil {
		t.Error(err)
	}
	if p.action != actionNone {
		t.Errorf("Want plan noop, got %v", p)
	}
}

// Verifies that planner recommends noop when extra agents exist
// but all are busy (ie, running at least 1 build).
func TestPlan_NoneIdle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	asg := mocks.NewMockAutoScalingAPI(ctrl)
	asg.
		EXPECT().
		DescribeAutoScalingGroups(gomock.Any()).
		Return(&autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []*autoscaling.Group{
				{
					Instances: []*autoscaling.Instance{
						{
							HealthStatus: aws.String("Healthy"),
							InstanceId:   aws.String("i-001"),
						},
						{
							HealthStatus: aws.String("Healthy"),
							InstanceId:   aws.String("i-002"),
						},
					},
					DesiredCapacity: aws.Int64(2),
				},
			},
		}, nil).
		Times(2)

	droneClient := mocks.NewMockClient(ctrl)
	droneClient.
		EXPECT().
		Queue().
		Return([]*drone.Stage{
			{Status: drone.StatusRunning, Machine: "i-001"},
			{Status: drone.StatusRunning, Machine: "i-002"},
		}, nil)

	c := cluster.New("test-asg", nil, asg)
	e := &Engine{
		drone: &droneConfig{
			client: droneClient,
			build: &droneBuildConfig{
				pendingMaxDuration: -1 * time.Second,
				runningMaxDuration: -1 * time.Second,
			},
			agent: &droneAgentConfig{cluster: c, maxBuilds: 2},
		},
	}

	p, err := e.Plan(context.TODO())
	if err != nil {
		t.Error(err)
	}
	if p.action != actionNone {
		t.Errorf("Want plan noop, got %v", p)
	}
}

// Verifies that planner recommends upscaling when running agent
// count < min agent count.
func TestPlan_BelowMinCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	asg := mocks.NewMockAutoScalingAPI(ctrl)
	asg.
		EXPECT().
		DescribeAutoScalingGroups(gomock.Any()).
		Return(&autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []*autoscaling.Group{
				{
					Instances: []*autoscaling.Instance{
						{
							HealthStatus: aws.String("Healthy"),
							InstanceId:   aws.String("i-009eed7816"),
						},
					},
					DesiredCapacity: aws.Int64(1),
				},
			},
		}, nil).
		Times(2)

	c := cluster.New("test-asg", nil, asg)
	e := &Engine{
		drone: &droneConfig{
			agent: &droneAgentConfig{cluster: c, minCount: 3},
		},
	}

	p, err := e.Plan(context.TODO())
	if err != nil {
		t.Error(err)
	}
	if p.action != actionUpscale {
		t.Errorf("Want plan upscale, got %v", p)
	}
	if p.upscaleCount != 2 {
		t.Errorf("Want plan upscale count 2, got %d", p.upscaleCount)
	}
}

// Verifies that planner recommends upscaling when there are
// pending builds.
func TestPlan_PendingBuilds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	asg := mocks.NewMockAutoScalingAPI(ctrl)
	asg.
		EXPECT().
		DescribeAutoScalingGroups(gomock.Any()).
		Return(&autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []*autoscaling.Group{
				{
					Instances: []*autoscaling.Instance{
						{
							HealthStatus: aws.String("Healthy"),
							InstanceId:   aws.String("i-009eed7816"),
						},
					},
					DesiredCapacity: aws.Int64(1),
				},
			},
		}, nil).
		Times(2)

	droneClient := mocks.NewMockClient(ctrl)
	droneClient.
		EXPECT().
		Queue().
		Return(
			[]*drone.Stage{
				// pending for 6 mins
				{
					Status:  drone.StatusPending,
					Created: time.Now().UTC().Add(-6 * time.Minute).Unix(),
				},
				// running for 6 mins
				{
					Status:  drone.StatusRunning,
					Created: time.Now().UTC().Add(-6 * time.Minute).Unix(),
				},
				// pending
				{
					Status:  drone.StatusPending,
					Created: time.Now().UTC().Add(-1 * time.Minute).Unix(),
				},
				{
					Status:  drone.StatusPending,
					Created: time.Now().UTC().Add(-1 * time.Minute).Unix(),
				},
			},
			nil,
		)

	c := cluster.New("test-asg", nil, asg)
	e := &Engine{
		drone: &droneConfig{
			client: droneClient,
			build: &droneBuildConfig{
				pendingMaxDuration: 5 * time.Minute,
				runningMaxDuration: 5 * time.Minute,
			},
			agent: &droneAgentConfig{cluster: c, minCount: 1, maxBuilds: 2},
		},
	}

	p, err := e.Plan(context.TODO())
	if err != nil {
		t.Error(err)
	}
	if p.action != actionUpscale {
		t.Errorf("Want plan upscale, got %v", p)
	}
	if p.upscaleCount != 1 {
		t.Errorf("Want plan upscale count 1, got %d", p.upscaleCount)
	}
}

// Verifies that planner recommends downscaling when there are
// extra agents that can be destroyed.
func TestPlan_ExtraDestroyable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	asg := mocks.NewMockAutoScalingAPI(ctrl)
	asg.
		EXPECT().
		DescribeAutoScalingGroups(gomock.Any()).
		Return(&autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []*autoscaling.Group{
				{
					Instances: []*autoscaling.Instance{
						{
							HealthStatus: aws.String("Healthy"),
							InstanceId:   aws.String("i-123"),
						},
					},
					DesiredCapacity: aws.Int64(1),
				},
			},
		}, nil).
		Times(2)

	ec2Client := mocks.NewMockEC2API(ctrl)
	ec2Client.
		EXPECT().
		DescribeInstances(gomock.Any()).
		Return(&ec2.DescribeInstancesOutput{
			Reservations: []*ec2.Reservation{
				{
					Instances: []*ec2.Instance{
						// idle & past min retirement
						{
							InstanceId: aws.String("i-123"),
							LaunchTime: aws.Time(time.Now().UTC().Add(-20 * time.Minute)),
						},
					},
				},
			},
		}, nil)

	droneClient := mocks.NewMockClient(ctrl)
	droneClient.
		EXPECT().
		Queue().
		Return([]*drone.Stage{}, nil)

	c := cluster.New("test-asg", ec2Client, asg)
	e := &Engine{
		drone: &droneConfig{
			build: &droneBuildConfig{
				pendingMaxDuration: -1 * time.Second,
				runningMaxDuration: -1 * time.Second,
			},
			agent: &droneAgentConfig{
				cluster:          c,
				maxBuilds:        10,
				minRetirementAge: 10 * time.Minute,
				minCount:         0,
			},
			client: droneClient,
		},
	}

	p, err := e.Plan(context.TODO())
	if err != nil {
		t.Error(err)
	}
	if p.action != actionDownscale {
		t.Errorf("Want plan downscale, got %v", p)
	}
	if len(p.nodesToDestroy) != 1 || p.nodesToDestroy[0] != cluster.NodeId("i-123") {
		t.Errorf("Want only i-123 as node to destroy, got %v", p.nodesToDestroy)
	}
}

func TestPlan_RequiresUpscaling(t *testing.T) {
	p := Plan{action: actionUpscale}
	if !p.RequiresUpscaling() {
		t.Errorf("Expected RequiresUpscaling to be true")
	}
	p = Plan{action: actionDownscale}
	if p.RequiresUpscaling() {
		t.Errorf("Expected RequiresUpscaling to be false")
	}
}

func TestPlan_RequiresDownscaling(t *testing.T) {
	p := Plan{action: actionDownscale}
	if !p.RequiresDownscaling() {
		t.Errorf("Expected RequiresDownscaling to be true")
	}
	p = Plan{action: actionUpscale}
	if p.RequiresDownscaling() {
		t.Errorf("Expected RequiresDownscaling to be false")
	}
}

func TestPlan_UpscaleCount(t *testing.T) {
	p := Plan{upscaleCount: 7}
	if c := p.UpscaleCount(); c != 7 {
		t.Errorf("Want upscale count %d, got %d", 7, c)
	}
}

func TestPlan_NodesToDestroy(t *testing.T) {
	nodes := []cluster.NodeId{
		"i-198826ad1",
		"i-0000000",
		"i-ab182hs87198nnsh",
	}
	p := Plan{nodesToDestroy: nodes}
	got := p.NodesToDestroy()
	for i := 0; i < len(nodes); i++ {
		if got[i] != nodes[i] {
			t.Errorf("Want node %s, got %s at index %d", nodes[i], got[i], i)
		}
	}
}
