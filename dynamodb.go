package main

import (
	"strconv"

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
