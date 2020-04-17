package main

import (
	"strconv"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudfront"
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
			HttpVersion: *d.HttpVersion,
			Status:      *d.Status,
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
