package engine

import (
	"github.com/Shuttl-Tech/drone-autoscaler/cluster"
	"github.com/drone/drone-go/drone"
	"testing"
)

func TestEngine_CountBuilds(t *testing.T) {
	e := Engine{}
	stages := []*drone.Stage{
		{Status: drone.StatusRunning},
		{Status: drone.StatusPending},
		{Status: drone.StatusBlocked},
		{Status: drone.StatusError},
		{Status: drone.StatusPending},
		{Status: drone.StatusRunning},
		{Status: drone.StatusPending},
	}

	pending, running := e.countBuilds(stages)
	if pending != 3 {
		t.Errorf("Want pending 3, got %d", pending)
	}
	if running != 2 {
		t.Errorf("Want running 2, got %d", running)
	}
}

func TestEngine_CalcRequiredAgentCount(t *testing.T) {
	e := Engine{
		drone: &droneConfig{
			agent: &droneAgentConfig{maxBuilds: 15},
		},
	}
	if count, _ := e.calcRequiredAgentCount(0); count != 0 {
		t.Errorf("Want count 0, got %d", count)
	}
	if count, _ := e.calcRequiredAgentCount(1); count != 1 {
		t.Errorf("Want count 1, got %d", count)
	}
	if count, _ := e.calcRequiredAgentCount(15); count != 1 {
		t.Errorf("Want count 1, got %d", count)
	}
	if count, _ := e.calcRequiredAgentCount(16); count != 2 {
		t.Errorf("Want count 2, got %d", count)
	}
	if count, _ := e.calcRequiredAgentCount(47); count != 4 {
		t.Errorf("Want count 4, got %d", count)
	}

	e.drone.agent.maxBuilds = 0
	if count, err := e.calcRequiredAgentCount(3); err == nil {
		t.Errorf("Want error, got count %d", count)
	}
}

func TestEngine_ListBusyAgents(t *testing.T) {
	e := Engine{}
	stages := []*drone.Stage{
		{
			Machine: "i-100",
			Status:  drone.StatusRunning,
		},
		{
			Machine: "i-198",
			Status:  drone.StatusPending,
		},
		{
			Machine: "i-130",
			Status:  drone.StatusBlocked,
		},
		{
			Machine: "i-100",
			Status:  drone.StatusError,
		},
		{
			Machine: "i-100",
			Status:  drone.StatusPending,
		},
		{
			Machine: "i-130",
			Status:  drone.StatusRunning,
		},
		{
			Machine: "i-130",
			Status:  drone.StatusRunning,
		},
		{
			Machine: "i-289",
			Status:  drone.StatusPending,
		},
		{
			Machine: "i-100",
			Status:  drone.StatusError,
		},
		{
			Status:  drone.StatusRunning,
			Machine: "i-100",
		},
	}
	want := []cluster.NodeId{"i-100", "i-130"}

	got := e.listBusyAgents(stages)
	for i := 0; i < len(want); i++ {
		if want[i] != got[i] {
			t.Errorf("Want %s, got %s at index %d", want[i], got[i], i)
		}
	}
}

func TestEngine_ListIdleAgents(t *testing.T) {
	e := Engine{}
	busy := []cluster.NodeId{"i-100", "i-101", "i-102"}
	idle := []cluster.NodeId{"i-104", "i-105", "i-106", "i-foobar"}

	got := e.listIdleAgents(append(busy, idle...), busy)
	for i := 0; i < len(idle); i++ {
		if got[i] != idle[i] {
			t.Errorf("Want %s, got %s at index %d", idle[i], got[i], i)
		}
	}
}

func TestEngine_MaintainMinAgentCount(t *testing.T) {
	e := Engine{
		drone: &droneConfig{
			agent: &droneAgentConfig{minCount: 0},
		},
	}
	all := []cluster.NodeId{"i-100", "i-200"}

	if got := e.maintainMinAgentCount(all, all); len(got) != len(all) {
		t.Errorf("Want all agents, got %v", got)
	}
	if got := e.maintainMinAgentCount(all, []cluster.NodeId{"i-200"}); len(got) != 1 {
		t.Errorf("Want single node i-200, got %v", got)
	}

	e.drone.agent.minCount = 2
	if got := e.maintainMinAgentCount(all, all); len(got) > 0 {
		t.Errorf("Want 0 nodes, got %v", got)
	}

	all = append(all, []cluster.NodeId{"i-395"}...)
	if got := e.maintainMinAgentCount(all, all); len(got) != 1 {
		t.Errorf("Want 1 node, got %v", got)
	}

	all = append(all, []cluster.NodeId{"i-411", "i-422"}...)
	if got := e.maintainMinAgentCount(all, all); len(got) != 3 {
		t.Errorf("Want 3 nodes, got %v", got)
	}

	e.drone.agent.minCount = len(all) + 1
	if got := e.maintainMinAgentCount(all, []cluster.NodeId{"i-100"}); len(got) != 0 {
		t.Errorf("Want 0 nodes, got %v", got)
	}
}
