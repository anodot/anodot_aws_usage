package main

import (
	"strings"
	"time"

	"github.com/anodot/anodot-common/pkg/metrics"
	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
	"github.com/anodot/anodot-common/pkg/metrics3"
)

func GetSupportedService() []string {
	return []string{
		"EC2",
		"EBS",
		"ELB",
		"S3",
		"Cloudfront",
		"NatGateway",
		"Efs",
		"DynamoDB",
		"Kinesis",
		"ElastiCache",
	}
}

func removeDuplicates(list []string) []string {
	new := make([]string, 0)
	ifPresent := false
	for _, s := range list {
		for _, snew := range new {
			if s == snew {
				ifPresent = true
			}
		}
		if !ifPresent {
			new = append(new, s)
		} else {
			ifPresent = false
		}
	}
	return new
}

func escape(s string) string {
	return strings.ReplaceAll(s, ":", "_")
}

func GetAnodotMetric(name string, timestemps []*time.Time, values []*float64, properties map[string]string) []metricsAnodot.Anodot20Metric {
	/*if accountId != "" {
		properties["account_id"] = accountId
	}*/

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

func GetAnodotMetric30(name string, timestemps []*time.Time, values []*float64, dimensions map[string]string) []metrics3.AnodotMetrics30 {
	metrics := make([]metrics3.AnodotMetrics30, 0)
	for i := 0; i < len(values); i++ {

		metric := metrics3.AnodotMetrics30{
			Dimensions:   dimensions,
			Timestamp:    metrics3.AnodotTimestamp{*timestemps[i]},
			Measurements: map[string]float64{name: *values[i]},
		}
		metrics = append(metrics, metric)
	}

	return metrics
}
