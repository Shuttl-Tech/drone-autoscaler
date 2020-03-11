package engine

import (
	"github.com/drone/drone-go/drone"
	"testing"
)

func TestFilterStages(t *testing.T) {
	stages := []*drone.Stage{
		{},
		{},
		{},
	}

	got := filterStages(stages, func(stage *drone.Stage) bool {
		return true
	})
	if len(got) != len(stages) {
		t.Errorf("Want %d stage objects, got %d", len(stages), len(got))
	}

	got = filterStages(stages, func(stage *drone.Stage) bool {
		return false
	})
	if len(got) != 0 {
		t.Errorf("Want empty list, got %v", got)
	}
}
