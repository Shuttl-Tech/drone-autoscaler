package cluster

import "github.com/aws/aws-sdk-go/aws"

func nodeIdsToAwsStrings(ids []NodeId) []*string {
	resp := make([]*string, len(ids), len(ids))
	for _, id := range ids {
		resp = append(resp, aws.String(string(id)))
	}
	return resp
}
