package aws

import (
	"log"
	"strconv"
	"time"

	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Metric struct {
	Name       string
	Dimensions []Dimension
}

type S3 struct {
	BucketName string
	Region     string
	S3Metrics  []S3Metric
}

func GetS3Buckets(session *session.Session, listmetrics []*cloudwatch.Metric) ([]S3, error) {
	s3list := make([]S3, 0)
	region := session.Config.Region

	s3input := &s3.ListBucketsInput{}
	svc := s3.New(session)
	result, err := svc.ListBuckets(s3input)
	if err != nil {
		return s3list, err
	}
	var dbucketname string
	for _, s := range result.Buckets {
		s3Metrics := make([]S3Metric, 0)
		for _, m := range listmetrics {

			dimensions := make([]Dimension, 0)

			for _, d := range m.Dimensions {

				if *d.Name == "BucketName" {
					dbucketname = *d.Value
					if *d.Value == *s.Name {
						for _, d2 := range m.Dimensions {
							dimensions = append(dimensions, Dimension{
								Name:  *d2.Name,
								Value: *d2.Value,
							})
						}
					}
				}
			}
			if dbucketname == *s.Name {
				s3Metrics = append(s3Metrics, S3Metric{
					Name:       *m.MetricName,
					Dimensions: dimensions,
				})
			}
		}
		s3list = append(s3list, S3{
			BucketName: *s.Name,
			Region:     *region,
			S3Metrics:  s3Metrics,
		})
	}

	return s3list, err
}

func GetS3Dimensions() []string {
	return []string{
		"storage_type",
		"service",
		"bucket_name",
		"region",
		"anodot-collector",
	}
}

func GetS3MetricProperties(bucket S3) map[string]string {
	properties := map[string]string{
		"service":          "s3",
		"bucket_name":      bucket.BucketName,
		"region":           bucket.Region,
		"anodot-collector": "aws",
	}

	for k, v := range properties {
		if len(v) > 50 || len(v) < 2 {
			delete(properties, k)
		}
	}
	return properties
}

func GetS3CloudwatchMetrics(resource *MonitoredResource, buckets []S3) ([]MetricToFetch, error) {
	metrics := make([]MetricToFetch, 0)
	for _, mstat := range resource.Metrics {
		for _, bucket := range buckets {
			m := MetricToFetch{}
			for _, s3m := range bucket.S3Metrics {
				if s3m.Name == mstat.Name {
					m.Dimensions = s3m.Dimensions
				}
			}
			m.Resource = bucket
			mstatCopy := mstat
			mstatCopy.Id = "s3" + strconv.Itoa(len(metrics))
			m.MStat = mstatCopy
			metrics = append(metrics, m)
		}
	}
	return metrics, nil
}

func GetCloudwatchMetricList(cloudwatchSvc *cloudwatch.CloudWatch) ([]*cloudwatch.Metric, error) {
	namespace := "AWS/S3"
	lmi := &cloudwatch.ListMetricsInput{
		Namespace: &namespace,
	}

	listmetrics, err := cloudwatchSvc.ListMetrics(lmi)
	if err != nil {
		return nil, err
	}
	return listmetrics.Metrics, nil
}

func GetS3Metrics(session *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
	anodotMetrics := make([]metricsAnodot.Anodot20Metric, 0)
	cloudWatchFetcher := CloudWatchFetcher{
		cloudwatchSvc: cloudwatchSvc,
	}

	listmetrics, err := GetCloudwatchMetricList(cloudwatchSvc)
	if err != nil {
		log.Printf("Could not get S3 metric list: %v", err)
		return anodotMetrics, err
	}

	buckets, err := GetS3Buckets(session, listmetrics)
	if err != nil {
		log.Printf("Could not describe S3 buckets: %v", err)
		return anodotMetrics, err
	}
	log.Printf("Got %d S3 buckets to process", len(buckets))

	metrics, err := GetS3CloudwatchMetrics(resource, buckets)
	if err != nil {
		log.Printf("Error during s3 metrics processing: %v", err)
		return anodotMetrics, err
	}

	dataInputs := NewGetMetricDataInput(metrics)

	for _, mi := range dataInputs {
		mi.SetStartTime(time.Now().Add(-48 * time.Hour))
	}

	metricdataresults, err := cloudWatchFetcher.FetchMetrics(dataInputs)
	if err != nil {
		log.Printf("Error during s3 metrics processing: %v", err)
		return anodotMetrics, err
	}

	now := time.Now()
	for _, m := range metrics {
		for _, mr := range metricdataresults {
			if *mr.Id == m.MStat.Id {
				s := m.Resource.(S3)
				properties := GetS3MetricProperties(s)
				if len(m.Dimensions) > 0 {
					for _, d := range m.Dimensions {
						if d.Name == "StorageType" {
							properties["storage_type"] = d.Value
						}
					}
				}

				if len(mr.Values) > 0 {
					timestemps := []*time.Time{&now}
					values := []*float64{mr.Values[0]}
					anodotMetrics = append(anodotMetrics, GetAnodotMetric(m.MStat.Name, timestemps, values, properties)...)
				}
			}
		}
	}
	return anodotMetrics, nil
}
