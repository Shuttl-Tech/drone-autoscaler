package engine

import (
	"github.com/Shuttl-Tech/drone-autoscaler/cluster"
	"testing"
)

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
