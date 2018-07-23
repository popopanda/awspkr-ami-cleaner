package main

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {

	const maxKeepHours = 24.0
	dryRun := aws.Bool(true)

	sess := session.Must(session.NewSession())
	svc := ec2.New(sess)

	input := &ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:Managed_by"),
				Values: []*string{
					aws.String("packer"),
				},
			},
		},
	}

	result, err := svc.DescribeImages(input)
	errorHandle(err)

	for _, pkrImages := range result.Images {
		for _, value := range pkrImages.Tags {
			if *value.Key == *aws.String("Role") {
				pkrTime, err := time.Parse(time.RFC3339, aws.StringValue(pkrImages.CreationDate))
				errorHandle(err)
				timeSince := time.Since(pkrTime).Hours()

				if timeSince <= maxKeepHours {
					fmt.Printf("Skipping AMI: %s\n", aws.StringValue(pkrImages.ImageId))
				} else {
					fmt.Printf("Deregistering AMI: %s\n", aws.StringValue(pkrImages.ImageId))
					svc.DeregisterImage(&ec2.DeregisterImageInput{
						DryRun:  dryRun,
						ImageId: pkrImages.ImageId,
					})
				}
			} else {
				continue
			}
		}
	}
}

func errorHandle(err error) {
	if err != nil {
		log.Fatalf("There was an error, %v\n", err)
	}
}
