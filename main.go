package main

import (
	"errors"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/anodot/anodot-common/pkg/metrics"
	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"

	//"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

const metricVersion string = "5"
const metricsPerSecond = 1000

var accountId string

func GetAnodotMetric(name string, timestemps []*time.Time, values []*float64, properties map[string]string) []metricsAnodot.Anodot20Metric {
	properties["metric_version"] = metricVersion
	if accountId != "" {
		properties["account_id"] = accountId
	}

	metricList := make([]metricsAnodot.Anodot20Metric, 0)
	for i := 0; i < len(values); i++ {
		properties["what"] = name
		metric := metrics.Anodot20Metric{
			Properties: properties,
			Value:      float64(*values[i]),
			Timestamp: metrics.AnodotTimestamp{
				*timestemps[i],
			},
		}
		metricList = append(metricList, metric)
	}
	return metricList
}

func escape(s string) string {
	return strings.ReplaceAll(s, ":", "_")
}

func Send(metrics []metricsAnodot.Anodot20Metric, submiter *metricsAnodot.Anodot20Client) error {
	response, err := submiter.SubmitMetrics(metrics)

	if err != nil || response.HasErrors() {
		log.Fatalf("Error during sending metrics to Anodot response: %v   Error: %v", response.RawResponse(), err)
		if response.HasErrors() {
			return errors.New(response.ErrorMessage())
		}
	} else {
		log.Printf("Successfully pushed %d metric to anodot \n", len(metrics))
	}
	return err
}

func SendMetrics(metrics []metricsAnodot.Anodot20Metric, submiter *metricsAnodot.Anodot20Client) error {
	var previousIndex int = 0
	var index int = 0
	var totalCount int = 0

	for index < len(metrics) {
		previousIndex = index
		index = index + metricsPerSecond
		if index > len(metrics) {
			index = len(metrics)
		}
		var metricsSlice []metricsAnodot.Anodot20Metric = metrics[previousIndex:index]
		err := Send(metricsSlice, submiter)
		if err != nil {
			log.Printf("Retry sending metrics")
			err := Send(metricsSlice, submiter)
			if err != nil {
				return err
			}
		}
		totalCount = totalCount + len(metricsSlice)
	}
	log.Printf("Metrics pushed total count  %d \n", totalCount)
	return nil
}

func LambdaHandler() {
	c, err := GetConfig()
	if err != nil {
		log.Fatalf("Could not parse config: %v", err)
	}
	ml := &SyncMetricList{
		metrics: make([]metricsAnodot.Anodot20Metric, 0),
	}

	el := &ErrorList{
		errors: make([]error, 0),
	}

	accountId = c.AccountId
	var wg sync.WaitGroup

	session := session.Must(session.NewSession(&aws.Config{Region: aws.String(c.Region)}))
	cloudwatchSvc := cloudwatch.New(session)

	url, err := url.Parse(c.AnodotUrl)
	if err != nil {
		log.Fatalf("Could not parse Anodot url: %v", err)
	}

	metricSubmitter, err := metrics.NewAnodot20Client(*url, c.AnodotToken, nil)
	if err != nil {
		log.Fatalf("Could create Anodot metrc submitter: %v", err)
	}

	Handle(c.RegionsConfigs[c.Region].Resources, &wg, session, cloudwatchSvc, ml, el)
	wg.Wait()

	if len(el.errors) > 0 {
		for _, e := range el.errors {
			log.Printf("ERROR occured: %v", e)
		}
	}

	if len(ml.metrics) > 0 {
		log.Printf("Total metrics count %d", len(ml.metrics))
		err := SendMetrics(ml.metrics, metricSubmitter)
		if err != nil {
			log.Fatalf("Failed to send metrics")
		}
	} else {
		log.Print("No any metrics to push ")
	}
}

func main() {
	lambda.Start(LambdaHandler)
}
