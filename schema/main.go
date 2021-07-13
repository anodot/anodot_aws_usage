package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"

	metrics3 "github.com/anodot/anodot-common/pkg/metrics3"
	awsLambda "github.com/usage-lambda/pkg/aws"
	"gopkg.in/yaml.v2"
)

var (
	DATA_TOKEN = "96c8c74cf52be98b395da9ca120f7067"
	ACESSS_KEY = "e6e666784cd90cde19d5cc7c9b7fa03b"
)

func ListServices() []string {
	return []string{
		"Elasticache",
		"Kinesis",
		"DynamoDB",
		"Efs",
		"NatGateway",
		"Cloudfront",
		"S3",
		"ELB",
		"EBS",
		"EC2",
	}
}

func CleanSchemas(client metrics3.Anodot30Client) error {
	resp, err := client.GetSchemas()
	if err != nil {
		return err
	}
	if resp.HasErrors() {
		return fmt.Errorf("failed to fetch schemas: %s", resp.ErrorMessage())
	}

	for _, schema := range resp.Schemas {
		for _, service := range ListServices() {
			if schema.Name == service+"_usage_schema" {
				delResp, err := client.DeleteSchema(schema.Id)
				if err != nil {
					return err
				}
				if delResp.HasErrors() {
					return fmt.Errorf("failed to delete schema %s:\n%s", schema.Name, resp.ErrorMessage())
				}
			}
		}
	}
	return nil
}

func CreateSchemas(client metrics3.Anodot30Client, schemas []metrics3.AnodotMetricsSchema) error {
	for _, schema := range schemas {
		resp, err := client.CreateSchema(schema)
		if err != nil {
			return err
		}
		if resp.HasErrors() {
			return fmt.Errorf("failed to create schema %s:\n%s", schema.Name, resp.ErrorMessage())
		}
	}
	return nil
}

func GetCustomMetricsAndDimensions(servicName string) ([]awsLambda.CustomMetricDefinition, []string) {
	emptyCm := make([]awsLambda.CustomMetricDefinition, 0)
	emptyD := make([]string, 0)
	switch servicName {
	case "EC2":
		return awsLambda.GetEc2CustomMetrics(), awsLambda.GetEc2Dimensions()
	case "EBS":
		return awsLambda.GetEBSCustomMetrics(), awsLambda.GetEBSDimensions()

	case "ELB":
		return emptyCm, awsLambda.GetELBDimensions()
	case "S3":
		return emptyCm, awsLambda.GetS3Dimensions()
	case "Cloudfront":
		return emptyCm, awsLambda.GetCloudfrontDimensions()
	case "NatGateway":
		return emptyCm, awsLambda.GetNatGatewayMetricDimensions()
	case "Efs":
		return awsLambda.GetEfsCustomMetrics(), awsLambda.GetEfsDimensions()
	case "DynamoDB":
		return emptyCm, awsLambda.GetDynamoDimensions()
	case "Kinesis":
		return emptyCm, awsLambda.GetStreamDimensions()
	case "Elasticache":
		return awsLambda.GetElasticacheCustomMetrics(), awsLambda.GetElasticacheDimensions()
	default:
		return emptyCm, emptyD
	}
}

// return schema per service
func GetSchemas(config ConfigForSchema) ([]metrics3.AnodotMetricsSchema, error) {
	schemas := make([]metrics3.AnodotMetricsSchema, 0)
	measurments := make(map[string]map[string]metrics3.MeasurmentBase)
	dimensions := make(map[string][]string, 0)

	var missingPolicy = &metrics3.DimensionPolicy{
		Action: "fill",
		Fill:   "unknown",
	}

	for _, services := range config.Regions {
		for servicName, service := range services {
			measurments[servicName] = make(map[string]metrics3.MeasurmentBase)

			customMetricsDefs, dims := GetCustomMetricsAndDimensions(servicName)
			if len(dims) == 0 {
				return nil, fmt.Errorf("unkown service %s", servicName)
			}

			dimensions[servicName] = dims
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
				var agg string
				if cm.Stat == "Sum" {
					agg = "sum"
				} else {
					agg = "average"
				}
				measurments[servicName][cm.Name] = metrics3.MeasurmentBase{
					CountBy:     "none",
					Aggregation: agg,
				}
			}
		}
	}

	for k, v := range measurments {
		schemas = append(schemas, metrics3.AnodotMetricsSchema{
			Name:             k + "_usage_schema",
			Measurements:     v,
			Dimensions:       dimensions[k],
			MissingDimPolicy: missingPolicy,
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

	/*b, err := json.Marshal(schemas)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	*/
	url, _ := url.Parse("https://app.anodot.com")

	client, err := metrics3.NewAnodot30Client(*url, &ACESSS_KEY, &DATA_TOKEN, nil)
	if err != nil {
		panic(err)
	}
	err = CleanSchemas(*client)
	if err != nil {
		panic(err)
	}
	err = CreateSchemas(*client, schemas)
	if err != nil {
		panic(err)
	}

}
