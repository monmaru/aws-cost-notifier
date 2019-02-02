package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
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

	if len(datapoints) == 0 {
		return "Datapoints is empty!", nil
	}

	if err := post2Slack(datapoints); err != nil {
		return "Failed post2Slack", err
	}

	return "Success!", nil
}

// Get yesterday's billing
func getBilling() ([]*cloudwatch.Datapoint, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return nil, err
	}
	endTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jst)
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

	statistics, err := svc.GetMetricStatistics(params)
	if err != nil {
		return nil, err
	}

	return statistics.Datapoints, nil
}

type field struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

type message struct {
	Channel string  `json:"channel"`
	BotName string  `json:"username"`
	PreText string  `json:"pretext"`
	Color   string  `json:"color"`
	Fields  []field `json:"fields"`
}

func post2Slack(datapoints []*cloudwatch.Datapoint) error {
	msg := &message{
		Channel: os.Getenv("slackChannel"),
		BotName: "aws-cost-bot",
		PreText: "AWSの料金",
		Color:   "#36a64f",
		Fields: []field{
			field{
				Title: "合計金額",
				Value: strconv.FormatFloat(*datapoints[0].Maximum, 'f', 2, 64) + "ドル（USD）",
			},
		},
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"POST",
		os.Getenv("slackPostURL"),
		bytes.NewBuffer(b),
	)

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return nil
}
