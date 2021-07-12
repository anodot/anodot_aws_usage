package aws

import (
	"log"
	"strconv"

	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var operations = []string{"PutItem", "UpdateItem", "Scan", "GetItem"}

type DynamoTable struct {
	Name   string
	Region string
}

func ListTables(session *session.Session) ([]DynamoTable, error) {
	region := session.Config.Region
	tables := make([]DynamoTable, 0)

	svc := dynamodb.New(session)
	input := &dynamodb.ListTablesInput{}

	result, err := svc.ListTables(input)
	if err != nil {
		return tables, nil
	}
	for _, tn := range result.TableNames {
		tables = append(tables, DynamoTable{Name: *tn, Region: *region})
	}

	return tables, nil
}

func GetDynamoProperties(table DynamoTable) map[string]string {
	properties := map[string]string{
		"service":          "efs",
		"table_name":       table.Name,
		"anodot-collector": "aws",
		"region":           table.Region,
	}
	return properties
}

func GetCloudwatchDynamoMetricList(cloudwatchSvc *cloudwatch.CloudWatch) ([]*cloudwatch.Metric, error) {
	namespace := "AWS/DynamoDB"
	lmi := &cloudwatch.ListMetricsInput{
		Namespace: &namespace,
	}

	listmetrics, err := cloudwatchSvc.ListMetrics(lmi)
	if err != nil {
		return nil, err
	}
	return listmetrics.Metrics, nil
}

func GetDynamoCloudwatchMetrics(resource *MonitoredResource, tables []DynamoTable) ([]MetricToFetch, error) {
	metrics := make([]MetricToFetch, 0)
	for _, mstat := range resource.Metrics {
		for _, t := range tables {
			if mstat.Name == "SuccessfulRequestLatency" {
				for _, o := range operations {
					m := MetricToFetch{}
					m.Dimensions = []Dimension{
						Dimension{
							Name:  "TableName",
							Value: t.Name,
						},
						Dimension{
							Name:  "Operation",
							Value: o,
						},
					}
					m.Resource = t
					mstatCopy := mstat
					mstatCopy.Id = "dynamo" + strconv.Itoa(len(metrics))
					m.MStat = mstatCopy
					metrics = append(metrics, m)
				}
			}

			m := MetricToFetch{}
			m.Dimensions = []Dimension{
				Dimension{
					Name:  "TableName",
					Value: t.Name,
				},
			}

			if mstat.Name == "ReturnedItemCount" {
				m.Dimensions = append(m.Dimensions, Dimension{
					Name:  "Operation",
					Value: "Scan",
				})
			}
			m.Resource = t
			mstatCopy := mstat
			mstatCopy.Id = "dynamo" + strconv.Itoa(len(metrics))
			m.MStat = mstatCopy
			metrics = append(metrics, m)
		}
	}

	return metrics, nil
}

func GetDynamoDbMetrics(ses *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
	anodotMetrics := make([]metricsAnodot.Anodot20Metric, 0)

	cloudWatchFetcher := CloudWatchFetcher{
		cloudwatchSvc: cloudwatchSvc,
	}
	tables, err := ListTables(ses)
	if err != nil {
		log.Printf("Cloud not get list Dynamo DB tables : %v", err)
		return anodotMetrics, nil
	}
	log.Printf("Found %d Dynamo DB tables ", len(tables))
	metrics, err := GetDynamoCloudwatchMetrics(resource, tables)
	if err != nil {
		log.Printf("Error: %v", err)
		return anodotMetrics, err
	}

	metricdatainput := NewGetMetricDataInput(metrics)
	metricdataresults, err := cloudWatchFetcher.FetchMetrics(metricdatainput)
	if err != nil {
		log.Printf("Error during DynamoDB metrics processing: %v", err)
		return anodotMetrics, err
	}

	for _, m := range metrics {
		for _, mr := range metricdataresults {
			if *mr.Id == m.MStat.Id {

				table := m.Resource.(DynamoTable)
				properties := GetDynamoProperties(table)
				for _, d := range m.Dimensions {
					if d.Name == "Operation" {
						properties["operation"] = d.Value
					}
				}

				anodot_dynamo_metrics := GetAnodotMetric(m.MStat.Name, mr.Timestamps, mr.Values, properties)
				anodotMetrics = append(anodotMetrics, anodot_dynamo_metrics...)
			}
		}
	}
	return anodotMetrics, nil
}
