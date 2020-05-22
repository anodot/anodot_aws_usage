package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/anodot/anodot-common/pkg/metrics"
	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/efs"
)

type Efs struct {
	FileSystemId *string
	Name         *string
	Size         *float64
	Tags         []*efs.Tag
	SizeInIA     *float64 //// The latest known metered size (in bytes) of data stored in the Infrequent Access storage class.
	SizeS        *float64 //// The latest known metered size (in bytes) of data stored in the Standard storage class.
}

func DesribeFilesystems(session *session.Session) ([]Efs, error) {
	efss := make([]Efs, 0)
	svc := efs.New(session)
	input := &efs.DescribeFileSystemsInput{}
	result, err := svc.DescribeFileSystems(input)
	if err != nil {
		return efss, err
	}

	if len(result.FileSystems) == 0 {
		return efss, fmt.Errorf("Can not found Efs in selected region")
	}

	for _, efs := range result.FileSystems {
		sizeInA := float64(*efs.SizeInBytes.ValueInIA)
		sizeInS := float64(*efs.SizeInBytes.ValueInStandard)
		size := float64(*efs.SizeInBytes.Value)

		efss = append(efss, Efs{
			FileSystemId: efs.FileSystemId,
			Name:         efs.Name,
			Tags:         efs.Tags,
			Size:         &size,
			SizeInIA:     &sizeInA,
			SizeS:        &sizeInS,
		})
	}
	return efss, nil
}

func GetEfsMetricProperties(efs Efs) map[string]string {
	properties := map[string]string{
		"service":      "efs",
		"Name":         *efs.Name,
		"FileSystemId": *efs.FileSystemId,
	}

	for _, v := range efs.Tags {
		if len(*v.Key) > 50 || len(*v.Value) < 2 {
			continue
		}
		if len(properties) == 17 {
			break
		}
		properties[escape(*v.Key)] = escape(*v.Value)
	}

	for k, v := range properties {
		if len(v) > 50 || len(v) < 2 {
			delete(properties, k)
		}
	}
	return properties
}

func GetEfsCloudwatchMetrics(resource *MonitoredResource, efss []Efs) ([]MetricToFetch, error) {
	metrics := make([]MetricToFetch, 0)

	for _, mstat := range resource.Metrics {
		for _, fs := range efss {
			m := MetricToFetch{}
			m.Dimensions = []Dimension{
				Dimension{
					Name:  "FileSystemId",
					Value: *fs.FileSystemId,
				},
			}
			m.Resource = fs
			mstatCopy := mstat
			mstatCopy.Id = "efs" + strconv.Itoa(len(metrics))
			m.MStat = mstatCopy
			metrics = append(metrics, m)
		}
	}

	return metrics, nil
}

func getEfsSizetMetric(efss []Efs) []metricsAnodot.Anodot20Metric {
	metricList := make([]metricsAnodot.Anodot20Metric, 0)
	for _, efs := range efss {
		metricList = append(metricList, addAnodotMetric(efs, "Size", *efs.Size))
	}
	return metricList
}

func getEfsInfrequentSizeMetric(efss []Efs) []metricsAnodot.Anodot20Metric {
	metricList := make([]metricsAnodot.Anodot20Metric, 0)
	for _, efs := range efss {
		metricList = append(metricList, addAnodotMetric(efs, "Size_Infrequent", *efs.SizeInIA))
	}
	return metricList
}

func getEfsStandartSizetMetric(efss []Efs) []metricsAnodot.Anodot20Metric {
	metricList := make([]metricsAnodot.Anodot20Metric, 0)
	for _, efs := range efss {
		metricList = append(metricList, addAnodotMetric(efs, "Size_Standart", *efs.SizeS))
	}
	return metricList
}

func addAnodotMetric(efs Efs, what string, value float64) metricsAnodot.Anodot20Metric {
	properties := GetEfsMetricProperties(efs)
	if accountId != "" {
		properties["account_id"] = accountId
	}
	//properties["target_type"] = "counter"
	properties["metric_version"] = metricVersion
	properties["what"] = what
	metric := metrics.Anodot20Metric{
		Properties: properties,
		Value:      float64(value),
		Timestamp: metrics.AnodotTimestamp{
			time.Now(),
		},
	}
	return metric
}
