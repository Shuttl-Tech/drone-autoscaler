package cluster

import "github.com/aws/aws-sdk-go/aws"

func nodeIdsToAwsStrings(ids []NodeId) []*string {
	resp := make([]*string, len(ids), len(ids))
	for i, id := range ids {
		resp[i] = aws.String(string(id))
	}
	return resp
}
