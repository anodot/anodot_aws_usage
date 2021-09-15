package main

import (
	"fmt"
	"log"
	"reflect"

	"github.com/anodot/anodot-common/pkg/metrics3"
)

func schemaName(accountId string, sname string) string {
	return accountId + "_" + sname + "_usage_schema"
}

type SchemasManager struct {
	client metrics3.Anodot30Client
}

func (sm *SchemasManager) DeleteSchema(schema metrics3.AnodotMetricsSchema) error {
	delResp, err := sm.client.DeleteSchema(schema.Id)
	if err != nil {
		return err
	}
	if delResp.HasErrors() {
		return fmt.Errorf("failed to delete schema %s:\n%s", schema.Name, delResp.ErrorMessage())
	}
	return nil
}

func (sm *SchemasManager) CreateSchema(schema metrics3.AnodotMetricsSchema) error {
	resp, err := sm.client.CreateSchema(schema)
	if err != nil {
		return err
	}
	if resp.HasErrors() {
		return fmt.Errorf("failed to create schema %s:\n%s", schema.Name, resp.ErrorMessage())
	}
	return nil
}

func (sm *SchemasManager) UpdateSchemas(snew, sold []metrics3.AnodotMetricsSchema) error {
	for _, s2 := range snew {
		for i1, s1 := range sold {
			if s1.Name == s2.Name {
				if !isSchemasEq(s1, s2) {
					log.Printf("schema config for %s has been changed, will recreate it", s1.Name)
					err := sm.DeleteSchema(s1)
					if err != nil {
						return fmt.Errorf("failed to delete schema %s:\n%v", s1.Name, err)
					}
					err = sm.CreateSchema(s2)
					if err != nil {
						return fmt.Errorf("failed to create schema %s:\n%v", s2.Name, err)
					}
				}
				break
			} else {
				// In case when schema is missed - create it
				if i1 == len(sold)-1 {
					log.Printf("schema %s is absent, will create it", s2.Name)
					err := sm.CreateSchema(s2)
					if err != nil {
						return fmt.Errorf("failed to create schema %s:\n%v", s2.Name, err)
					}
				}
			}
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

func GetSchemasFromConfig(config Config) ([]metrics3.AnodotMetricsSchema, error) {
	schemas := make([]metrics3.AnodotMetricsSchema, 0)
	measurments := make(map[string]map[string]metrics3.MeasurmentBase)
	dimensions := make(map[string][]string, 0)

	var missingPolicy = &metrics3.DimensionPolicy{
		Action: "fill",
		Fill:   "unknown",
	}

	region := config.RegionsConfigs[config.Region]

	for servicName, service := range region {
		measurments[servicName] = make(map[string]metrics3.MeasurmentBase)

		customMetricsDefs, dims := GetCustomMetricsAndDimensions(servicName, service)
		if len(dims) == 0 {
			return nil, fmt.Errorf("unkown service %s", servicName)
		}

		dimensions[servicName] = append(dims, "account_id")
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

func isSchemasEq(s1, s2 metrics3.AnodotMetricsSchema) bool {
	if !reflect.DeepEqual(s1.Dimensions, s2.Dimensions) {
		return false
	}

	if !reflect.DeepEqual(s1.Measurements, s2.Measurements) {
		return false
	}

	if !reflect.DeepEqual(s1.MissingDimPolicy, s2.MissingDimPolicy) {
		return false
	}

	if s1.Name != s2.Name {
		return false
	}
	return true
}
