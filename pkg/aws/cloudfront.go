package aws

import (
	"log"
	"strconv"

	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

type Ditribution struct {
	Id          string
	DomainName  string
	Enabled     string
	HttpVersion string
	Origins     map[string]string
	Status      string
}

func GetDitributions(session *session.Session) ([]Ditribution, error) {
	distributions := make([]Ditribution, 0)
	svc := cloudfront.New(session)
	input := &cloudfront.ListDistributionsInput{}
	result, err := svc.ListDistributions(input)
	if err != nil {
		return distributions, err
	}

	for _, d := range result.DistributionList.Items {

		distribution := Ditribution{
			Id:          *d.Id,
			DomainName:  *d.DomainName,
			Enabled:     strconv.FormatBool(*d.Enabled),
			HttpVersion: "None",
			Status:      *d.Status,
		}

		if d.HttpVersion != nil {
			distribution.HttpVersion = *d.HttpVersion
		}

		origins := make(map[string]string)
		for _, o := range d.Origins.Items {
			origins[*o.Id] = *o.DomainName
		}
		distribution.Origins = origins
		distributions = append(distributions, distribution)
	}
	return distributions, nil
}

func GetCloudfrontDimensions() []string {
	return []string{
		"service",
		"domain_name",
		"enabled",
		"http_version",
		"distribution_id",
		"status",
	}
}

func GetCloudfrontMetricProperties(distribution Ditribution) map[string]string {
	properties := map[string]string{
		"service":         "cloudfront",
		"domain_name":     distribution.DomainName,
		"enabled":         distribution.Enabled,
		"http_version":    distribution.HttpVersion,
		"distribution_id": distribution.Id,
		"status":          distribution.Status,
	}

	for k, v := range distribution.Origins {
		origin_id := k
		if len(k) >= 50 {
			origin_id = string(k[:50])
		}

		properties[origin_id] = v
	}

	for k, v := range properties {
		if len(v) > 50 || len(v) < 2 {
			delete(properties, k)
		}
	}

	return properties
}

func GetCloudfrontCloudwatchMetrics(resource *MonitoredResource, distributions []Ditribution) ([]MetricToFetch, error) {
	metrics := make([]MetricToFetch, 0)

	for _, mstat := range resource.Metrics {
		for _, distribution := range distributions {
			m := MetricToFetch{}
			m.Dimensions = []Dimension{
				Dimension{
					Name:  "DistributionId",
					Value: distribution.Id,
				},
				Dimension{
					Name:  "Region",
					Value: "Global",
				},
			}
			m.Resource = distribution
			mstatCopy := mstat
			mstatCopy.Id = "cloudfront" + strconv.Itoa(len(metrics))
			m.MStat = mstatCopy
			metrics = append(metrics, m)
		}
	}
	return metrics, nil
}

func GetCloudfrontMetrics(ses *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
	if resource.CustomRegion != "" {
		cloudwatchSvc = cloudwatch.New(session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-1")})))
	}

	cloudWatchFetcher := CloudWatchFetcher{
		cloudwatchSvc: cloudwatchSvc,
	}

	anodotMetrics := make([]metricsAnodot.Anodot20Metric, 0)
	ditributions, err := GetDitributions(ses)
	if err != nil {
		log.Printf("Cloud not get list of Cloudfront distributions: %v", err)
		return anodotMetrics, err
	}

	metrics, err := GetCloudfrontCloudwatchMetrics(resource, ditributions)
	if err != nil {
		log.Printf("Error: %v", err)
		return anodotMetrics, err
	}
	metricdatainput := NewGetMetricDataInput(metrics)
	metricdataresults, err := cloudWatchFetcher.FetchMetrics(metricdatainput)
	if err != nil {
		log.Printf("Error during Cloudfront metrics processing: %v", err)
		return anodotMetrics, err
	}

	for _, m := range metrics {
		for _, mr := range metricdataresults {
			if *mr.Id == m.MStat.Id {
				d := m.Resource.(Ditribution)
				//log.Printf("Fetching CloudWatch metric: %s for ELB Id %s \n", m.MStat.Name, e.Name)
				anodot_cloudwatch_metrics := GetAnodotMetric(m.MStat.Name, mr.Timestamps, mr.Values, GetCloudfrontMetricProperties(d))
				anodotMetrics = append(anodotMetrics, anodot_cloudwatch_metrics...)
			}
		}
	}
	return anodotMetrics, nil

}
