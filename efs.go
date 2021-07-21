package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/anodot/anodot-common/pkg/metrics3"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/efs"
)

type Efs struct {
	FileSystemId  *string
	Name          *string
	Size          *float64
	Tags          []*efs.Tag
	SizeInIA      *float64 //// The latest known metered size (in bytes) of data stored in the Infrequent Access storage class.
	SizeS         *float64 //// The latest known metered size (in bytes) of data stored in the Standard storage class.
	Region        string
	DimensionTags []string
}

func DesribeFilesystems(session *session.Session, resource *MonitoredResource) ([]Efs, error) {
	region := session.Config.Region
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
			FileSystemId:  efs.FileSystemId,
			Name:          efs.Name,
			Tags:          efs.Tags,
			Size:          &size,
			SizeInIA:      &sizeInA,
			SizeS:         &sizeInS,
			Region:        *region,
			DimensionTags: resource.DimensionTags,
		})
	}
	return efss, nil
}

func GetEfsDimensions(resource *MonitoredResource) []string {
	dims := []string{
		"service",
		"FileSystemId",
		"anodot-collector",
		"region",
		"Name",
	}
	return removeDuplicates(append(dims, resource.DimensionTags...))
}

func GetEfsCustomMetrics() []CustomMetricDefinition {

	return []CustomMetricDefinition{
		CustomMetricDefinition{
			Name:       "Size_All",
			Alias:      "Size_All",
			TargetType: "average",
		},
		CustomMetricDefinition{
			Name:       "Size_Infrequent",
			Alias:      "Size_Infrequent",
			TargetType: "average",
		},
		CustomMetricDefinition{
			Name:       "Size_Standard",
			Alias:      "Size_Standard",
			TargetType: "average",
		},
	}
}

func GetEfsMetricProperties(efs Efs) map[string]string {
	properties := map[string]string{
		"service":          "efs",
		"FileSystemId":     *efs.FileSystemId,
		"anodot-collector": "aws",
		"region":           efs.Region,
	}

	if efs.Name != nil {
		properties["Name"] = *efs.Name
	}

	for _, v := range efs.Tags {
		for _, dt := range efs.DimensionTags {
			if *v.Key == dt {
				if len(*v.Key) > 50 || len(*v.Value) < 2 {
					continue
				}
				if len(properties) == 17 {
					break
				}
				properties[escape(*v.Key)] = escape(*v.Value)
			}
		}
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

func getEfsSizetMetric(efss []Efs) []metrics3.AnodotMetrics30 {
	metricList := make([]metrics3.AnodotMetrics30, 0)
	for _, efs := range efss {
		metricList = append(metricList, addAnodotMetric(efs, "Size_All", *efs.Size))
	}
	return metricList
}

func getEfsInfrequentSizeMetric(efss []Efs) []metrics3.AnodotMetrics30 {
	metricList := make([]metrics3.AnodotMetrics30, 0)
	for _, efs := range efss {
		metricList = append(metricList, addAnodotMetric(efs, "Size_Infrequent", *efs.SizeInIA))
	}
	return metricList
}

func getEfsStandardSizetMetric(efss []Efs) []metrics3.AnodotMetrics30 {
	metricList := make([]metrics3.AnodotMetrics30, 0)
	for _, efs := range efss {
		metricList = append(metricList, addAnodotMetric(efs, "Size_Standard", *efs.SizeS))
	}
	return metricList
}

func addAnodotMetric(efs Efs, what string, value float64) metrics3.AnodotMetrics30 {

	/*if accountId != "" {
		properties["account_id"] = accountId
	}*/
	//properties["target_type"] = "counter"

	metric := metrics3.AnodotMetrics30{
		Dimensions:   GetEfsMetricProperties(efs),
		Timestamp:    metrics3.AnodotTimestamp{time.Now()},
		Measurements: map[string]float64{what: float64(value)},
	}

	return metric
}

func GetEfsMetrics30(ses *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metrics3.AnodotMetrics30, error) {
	anodotMetrics := make([]metrics3.AnodotMetrics30, 0)

	cloudWatchFetcher := CloudWatchFetcher{
		cloudwatchSvc: cloudwatchSvc,
	}
	efss, err := DesribeFilesystems(ses, resource)
	if err != nil {
		log.Printf("Cloud not get Efs: %v", err)
		return anodotMetrics, nil
	}
	log.Printf("Found %d Elastic file systems", len(efss))
	metrics, err := GetEfsCloudwatchMetrics(resource, efss)
	if err != nil {
		log.Printf("Error: %v", err)
		return anodotMetrics, err
	}

	metricdatainput := NewGetMetricDataInput(metrics)
	metricdataresults, err := cloudWatchFetcher.FetchMetrics(metricdatainput)
	if err != nil {
		log.Printf("Error during EFS metrics processing: %v", err)
		return anodotMetrics, err
	}

	for _, m := range metrics {
		for _, mr := range metricdataresults {
			if *mr.Id == m.MStat.Id {
				efs := m.Resource.(Efs)
				anodot_efs_metrics := GetAnodotMetric30(m.MStat.Name, mr.Timestamps, mr.Values, GetEfsMetricProperties(efs))
				anodotMetrics = append(anodotMetrics, anodot_efs_metrics...)
			}
		}
	}

	if len(resource.CustomMetrics) > 0 {
		for _, cm := range resource.CustomMetrics {
			if cm == "Size_All" {
				log.Printf("Processing EFS custom metric Size\n")
				anodotMetrics = append(anodotMetrics, getEfsSizetMetric(efss)...)
			}
			if cm == "Size_Standard" {
				log.Printf("Processing EFS custom metric Size_Standard\n")
				anodotMetrics = append(anodotMetrics, getEfsStandardSizetMetric(efss)...)
			}
			if cm == "Size_Infrequent" {
				log.Printf("Processing EFS custom metric Size_Infrequent\n")
				anodotMetrics = append(anodotMetrics, getEfsInfrequentSizeMetric(efss)...)
			}
		}
	}

	return anodotMetrics, nil
}
