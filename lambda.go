package main

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

const (
	region         = "us-east-1"
	namespace      = "AWS/Billing"
	metricName     = "EstimatedCharges"
	dimensionName  = "Currency"
	dimensionValue = "USD"
	period         = 86400
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context) (string, error) {
	datapoints, err := getBilling()
	if err != nil {
		return "Failed getBilling", err
	}

	log.Print(datapoints)
	// TODO: slack or email
	return "Hello, Lambda Go!", nil
}

// Get yesterday's billing
func getBilling() ([]*cloudwatch.Datapoint, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	endTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	startTime := endTime.Add(-24 * time.Hour)

	svc := cloudwatch.New(sess)
	params := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metricName),
		Period:     aws.Int64(period),
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Statistics: []*string{
			aws.String(cloudwatch.StatisticMaximum),
		},
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String(dimensionName),
				Value: aws.String(dimensionValue),
			},
		},
		Unit: aws.String(cloudwatch.StandardUnitNone),
	}

	resp, err := svc.GetMetricStatistics(params)
	if err != nil {
		return nil, err
	}

	return resp.Datapoints, nil
}
