package main

import (
	"fmt"

	"github.com/anodot/anodot-common/pkg/metrics3"
)

func schemaName(accountId string, sname string) string {
	return accountId + "_" + sname + "_usage_schema"
}

func CleanSchemas(client metrics3.Anodot30Client, accountId string) error {
	resp, err := client.GetSchemas()
	if err != nil {
		return err
	}
	if resp.HasErrors() {
		return fmt.Errorf("failed to fetch schemas: %s", resp.ErrorMessage())
	}

	for _, schema := range resp.Schemas {
		for _, service := range GetSupportedService() {
			if schema.Name == schemaName(accountId, service) {
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

func GetCustomMetricsAndDimensions(servicName string, resource *MonitoredResource) ([]CustomMetricDefinition, []string) {
	emptyCm := make([]CustomMetricDefinition, 0)
	emptyD := make([]string, 0)
	switch servicName {
	case "EC2":
		return GetEc2CustomMetrics(), GetEc2Dimensions(resource)
	case "EBS":
		return GetEBSCustomMetrics(), GetEBSDimensions(resource)
	case "ELB":
		return emptyCm, GetELBDimensions(resource)
	case "S3":
		return emptyCm, GetS3Dimensions()
	case "Cloudfront":
		return emptyCm, GetCloudfrontDimensions()
	case "NatGateway":
		return emptyCm, GetNatGatewayMetricDimensions(resource)
	case "Efs":
		return GetEfsCustomMetrics(), GetEfsDimensions(resource)
	case "DynamoDB":
		return emptyCm, GetDynamoDimensions()
	case "Kinesis":
		return emptyCm, GetStreamDimensions()
	case "ElastiCache":
		return GetElasticacheCustomMetrics(), GetElasticacheDimensions()
	default:
		return emptyCm, emptyD
	}
}

func GetSchemas(config Config) ([]metrics3.AnodotMetricsSchema, error) {
	schemas := make([]metrics3.AnodotMetricsSchema, 0)
	measurments := make(map[string]map[string]metrics3.MeasurmentBase)
	dimensions := make(map[string][]string, 0)

	var missingPolicy = &metrics3.DimensionPolicy{
		Action: "fill",
		Fill:   "unknown",
	}

	for _, region := range config.RegionsConfigs {
		for servicName, service := range region {
			measurments[servicName] = make(map[string]metrics3.MeasurmentBase)

			customMetricsDefs, dims := GetCustomMetricsAndDimensions(servicName, service)
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
			for _, cm := range service.Metrics {
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
			Name:             schemaName(config.AccountId, k),
			Measurements:     v,
			Dimensions:       dimensions[k],
			MissingDimPolicy: missingPolicy,
		})
	}
	return schemas, nil
}
