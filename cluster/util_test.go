package cluster

import (
	"github.com/aws/aws-sdk-go/aws"
	"testing"
)

func TestNodeIdsToAwsStrings(t *testing.T) {
	tests := []struct {
		c    []NodeId
		want []*string
	}{
		{[]NodeId{}, []*string{}},
		{
			[]NodeId{
				NodeId("foo"),
				NodeId("bar"),
				NodeId("bax-lorem_dollar"),
			},
			[]*string{
				aws.String("foo"),
				aws.String("bar"),
				aws.String("bax-lorem_dollar"),
			},
		},
	}
	for _, test := range tests {
		got := nodeIdsToAwsStrings(test.c)
		for i := 0; i < len(test.want); i++ {
			if *got[i] != *test.want[i] {
				t.Errorf("Want aws string %v, got %v at index %d", *test.want[i], *got[i], i)
			}
		}
	}
}
