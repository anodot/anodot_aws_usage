package main

import (
	"fmt"
	"io/ioutil"
	"log"

	metrics3 "github.com/anodot/anodot-common/pkg/metrics3"
	awsLambda "github.com/usage-lambda/pkg/aws"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/json"
)

// return schema per service
func GetSchemas(config ConfigForSchema) ([]metrics3.AnodotMetricsSchema, error) {
	schemas := make([]metrics3.AnodotMetricsSchema, 0)
	measurments := make(map[string]map[string]metrics3.MeasurmentBase)
	dimensions := make(map[string][]string, 0)

	for _, services := range config.Regions {

		for servicName, service := range services {
			measurments[servicName] = make(map[string]metrics3.MeasurmentBase)
			var customMetricsDefs []awsLambda.CustomMetricDefinition

			switch servicName {
			case "EC2":
				dimensions[servicName] = awsLambda.GetEc2Dimensions()
				customMetricsDefs = awsLambda.GetEc2CustomMetrics()
			case "EBS":
				dimensions[servicName] = awsLambda.GetEBSDimensions()
				customMetricsDefs = awsLambda.GetEBSCustomMetrics()
			case "ELB":
				dimensions[servicName] = awsLambda.GetELBDimensions()
			case "S3":
				dimensions[servicName] = awsLambda.GetS3Dimensions()
			case "Cloudfront":
				dimensions[servicName] = awsLambda.GetCloudfrontDimensions()
			case "NatGateway":
				dimensions[servicName] = awsLambda.GetNatGatewayMetricDimensions()
			case "Efs":
				dimensions[servicName] = awsLambda.GetEfsDimensions()
				customMetricsDefs = awsLambda.GetEfsCustomMetrics()
			case "DynamoDB":
				dimensions[servicName] = awsLambda.GetDynamoDimensions()
			case "Kinesis":
				dimensions[servicName] = awsLambda.GetStreamDimensions()
			case "Elasticache":
				dimensions[servicName] = awsLambda.GetElasticacheDimensions()
				customMetricsDefs = awsLambda.GetElasticacheCustomMetrics()
			default:
				return nil, fmt.Errorf("Unknown service name : %s", servicName)
			}

			// Add custom metric to schema
			for _, customMetric := range service.CustomMetrics {
				for _, customMetricDef := range customMetricsDefs {
					if customMetric == customMetricDef.Name || customMetric == customMetricDef.Alias {
						measurments[servicName][customMetricDef.Name] = metrics3.MeasurmentBase{
							CountBy:     "none",
							Aggregation: customMetricDef.TargetType,
						}
					}
				}
			}

			// Add cloudwatch metrics to schema
			for _, cm := range service.CloudwatchMetrics {
				measurments[servicName][cm.Name] = metrics3.MeasurmentBase{
					CountBy:     "none",
					Aggregation: cm.Stat,
				}
			}
		}
	}

	for k, v := range measurments {
		schemas = append(schemas, metrics3.AnodotMetricsSchema{
			Name:         k + "_" + "usage_schema",
			Measurements: v,
			Dimensions:   dimensions[k],
		})
	}
	return schemas, nil
}

func main() {
	fileData, err := ioutil.ReadFile("../cloudwatch_metrics.yaml")
	if err != nil {
		log.Fatalf("error: %v", err)
		panic(err)
	}

	c := ConfigForSchema{}
	err = yaml.Unmarshal(fileData, &c)
	if err != nil {
		panic(err)
	}
	schemas, err := GetSchemas(c)
	if err != nil {
		panic(err)
	}

	b, err := json.Marshal(schemas)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
