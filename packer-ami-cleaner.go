package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func janitor() {
	const maxKeepHours = 720.0 //60 days

	dryRun, _ := strconv.ParseBool(os.Getenv("DRYRUN"))
	maxKeepDays := maxKeepHours / 24

	fmt.Printf("AMIs are kept no longer than %v days.\n", strconv.FormatFloat(maxKeepDays, 'f', -1, 64))
	fmt.Printf("Dryrun is set to: %v\n", dryRun)

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

L:
	for _, pkrImages := range result.Images {
		pkrTime, err := time.Parse(time.RFC3339, aws.StringValue(pkrImages.CreationDate))
		errorHandle(err)
		timeSince := time.Since(pkrTime).Hours()

		if timeSince <= maxKeepHours {
			fmt.Printf("AMI: %s is not older than %v days, skipping...\n", aws.StringValue(pkrImages.ImageId), maxKeepDays)
		} else {

			for _, tags := range pkrImages.Tags {
				if aws.StringValue(tags.Key) == "Role" && aws.StringValue(tags.Value) == "ecs-agent" {
					fmt.Println("Skipping ecs-agent ami: ", aws.StringValue(pkrImages.ImageId))
					continue L
				}
			}
			fmt.Printf("Deregistering AMI: %s\n", aws.StringValue(pkrImages.ImageId))
			_, err := svc.DeregisterImage(&ec2.DeregisterImageInput{
				DryRun:  aws.Bool(dryRun),
				ImageId: pkrImages.ImageId,
			})
			errorHandle(err)
		}
	}
}

func errorHandle(err error) {
	if err != nil {
		log.Fatalf("There was an error, %v\n", err)
	}
}

func main() {
	lambda.Start(janitor)
}
