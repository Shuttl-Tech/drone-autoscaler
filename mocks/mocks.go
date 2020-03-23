package mocks

//go:generate mockgen -package=mocks -destination=mock_drone.go github.com/drone/drone-go/drone Client
//go:generate mockgen -package=mocks -destination=mock_aws_autoscaling.go github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface AutoScalingAPI
//go:generate mockgen -package=mocks -destination=mock_aws_ec2.go github.com/aws/aws-sdk-go/service/ec2/ec2iface EC2API
