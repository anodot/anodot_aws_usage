package main

import (
	"fmt"
	"log"
	"net/url"
	"sync"

	"github.com/anodot/anodot-common/pkg/metrics3"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	//"github.com/aws/aws-lambda-go/lambda"
)

const metricsPerSecond = 1000

var schemaIds map[string]string
var accountId string

func SetAccountId(in []metrics3.AnodotMetrics30) []metrics3.AnodotMetrics30 {
	for _, m := range in {
		if accountId != "" {
			m.Dimensions["account_id"] = accountId
		}
	}
	return in
}

func SendMetrics(metrics []metrics3.AnodotMetrics30, submiter *metrics3.Anodot30Client) error {
	var previousIndex int = 0
	var index int = 0
	var totalCount int = 0

	metrics = SetAccountId(metrics)
	for index < len(metrics) {
		previousIndex = index
		index = index + metricsPerSecond
		if index > len(metrics) {
			index = len(metrics)
		}
		var metricsSlice []metrics3.AnodotMetrics30 = metrics[previousIndex:index]
		err := SubmitMetrics(*submiter, metricsSlice)
		if err != nil {
			log.Printf("Retry sending metrics")
			err := SubmitMetrics(*submiter, metricsSlice)
			if err != nil {
				return err
			}
		}
		totalCount = totalCount + len(metricsSlice)
	}
	log.Printf("Metrics pushed total count  %d \n", totalCount)
	return nil
}

func SubmitMetrics(client metrics3.Anodot30Client, metrics []metrics3.AnodotMetrics30) error {
	respSubmit, err := client.SubmitMetrics(metrics)
	if err != nil {
		log.Printf("submition failed: %v", err)
		if respSubmit != nil {
			log.Printf("http status: %v", respSubmit.HttpResponse.Status)
		}
		return err
	}
	if respSubmit.HasErrors() {
		log.Printf("submition failed: %s", respSubmit.ErrorMessage())
		log.Printf("http status: %v", respSubmit.HttpResponse.Status)
		return fmt.Errorf(respSubmit.ErrorMessage())
	}
	return nil
}

func LambdaHandler() {
	var wg sync.WaitGroup

	schemaIds = make(map[string]string, 0)

	c, err := GetConfig()
	if err != nil {
		log.Fatalf("Could not parse config: %v", err)
	}
	accountId = c.AccountId

	ml := &SyncMetricList{
		metrics: make([]metrics3.AnodotMetrics30, 0),
	}

	el := &ErrorList{
		errors: make([]error, 0),
	}

	session := session.Must(session.NewSession(&aws.Config{Region: aws.String(c.Region)}))
	cloudwatchSvc := cloudwatch.New(session)

	url, err := url.Parse(c.AnodotUrl)
	if err != nil {
		log.Fatalf("Could not parse Anodot url: %v", err)
	}

	client, err := metrics3.NewAnodot30Client(*url, &c.AccessKey, &c.AnodotToken, nil)
	if err != nil {
		log.Fatalf("failed to create anodot30 client: %v", err)
	}

	sm := SchemasManager{*client}

	schemas, err := GetSchemasFromConfig(c)
	if err != nil {
		log.Fatalf("failed to get metrics schemas: %v", err)
	}

	respGetschemas, err := client.GetSchemas()
	if err != nil {
		log.Fatalf("failed to fetch metrics schemas: %v", err)
	}

	if respGetschemas.HasErrors() {
		log.Fatalf(respGetschemas.ErrorMessage())
	}

	err = sm.UpdateSchemas(schemas, respGetschemas.Schemas)
	if err != nil {
		log.Fatal(err)
	}
	// fetch schemas again after schema update
	respGetschemas, err = client.GetSchemas()
	if err != nil {
		log.Fatalf("failed to fetch metrics schemas: %v", err)
	}

	if respGetschemas.HasErrors() {
		log.Fatalf(respGetschemas.ErrorMessage())
	}

	for _, schema := range respGetschemas.Schemas {
		for _, service := range GetSupportedService() {
			if schema.Name == schemaName(accountId, service) {
				schemaIds[service] = schema.Id
			}
		}
	}

	Handle(c.RegionsConfigs[c.Region], &wg, session, cloudwatchSvc, ml, el)
	wg.Wait()

	if len(el.errors) > 0 {
		for _, e := range el.errors {
			log.Printf("ERROR occured: %v", e)
		}
		log.Fatalf("exiting...")
	}

	if len(ml.metrics) > 0 {
		log.Printf("Total fetched metrics count %d", len(ml.metrics))

		err := SendMetrics(ml.metrics, client)
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
