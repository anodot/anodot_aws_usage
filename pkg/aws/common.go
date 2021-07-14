package aws

import (
	"strings"
	"time"

	"github.com/anodot/anodot-common/pkg/metrics"
	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
)

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
