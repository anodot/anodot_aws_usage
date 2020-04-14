package main

import (
	"fmt"
	"github.com/anodot/anodot-common/pkg/metrics"
	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"log"
	"net/url"
	"strings"
	"time"
)

const metricVersion string = "4"

func GetEBSMetrics(session *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
	anodotMetrics := make([]metricsAnodot.Anodot20Metric, 0)
	ebss, err := GetEBSVolumes(session, resource.Tags)

	if err != nil {
		log.Printf("Cloud not describe EBS volumes %v", err)
		return anodotMetrics, err
	}
	log.Printf("Got %d EBS volumes to process", len(ebss))
	if len(resource.CustomMetrics) > 0 {
		for _, cm := range resource.CustomMetrics {
			if cm == "Size" {
				log.Printf("Processing EBS custom metric Size\n")
				anodotMetrics = append(anodotMetrics, getEBSSizeMetric(ebss)...)
			}
		}
	}
	return anodotMetrics, nil
}

func GetS3Metrics(session *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
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
		mi.SetStartTime(time.Now().Add(-36 * time.Hour))
	}

	metricdataresults, err := cloudWatchFetcher.FetchMetrics(dataInputs)
	if err != nil {
		log.Printf("Error during s3 metrics processing: %v", err)
		return anodotMetrics, err
	}

	var timestemps []*time.Time
	for _, m := range metrics {
		for _, mr := range metricdataresults {
			if *mr.Id == m.MStat.Id {
				s := m.Resource.(S3)

				if len(mr.Values) == 1 {
					now := time.Now()
					timestemps = []*time.Time{
						&now,
					}
				} else {
					timestemps = mr.Timestamps
				}
				anodotMetrics = append(anodotMetrics, GetAnodotMetric(m.MStat.Name, timestemps, mr.Values, GetS3MetricProperties(s))...)
			}
		}
	}
	return anodotMetrics, nil
}

func GetELBMetrics(session *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
	cloudWatchFetcher := CloudWatchFetcher{
		cloudwatchSvc: cloudwatchSvc,
	}

	anodotMetrics := make([]metricsAnodot.Anodot20Metric, 0)
	elbs, err := GetELBs(session, resource.Tags)
	if err != nil {
		log.Printf("Cloud not describe Load Balancers %v", err)
		return anodotMetrics, err
	}
	log.Printf("Got %d ELBs  to process", len(elbs))
	metrics, err := GetELBCloudwatchMetrics(resource, elbs)
	if err != nil {
		log.Printf("Error: %v", err)
		return anodotMetrics, err
	}

	metricdatainput := NewGetMetricDataInput(metrics)
	metricdataresults, err := cloudWatchFetcher.FetchMetrics(metricdatainput)
	if err != nil {
		log.Printf("Error during ELB metrics processing: %v", err)
		return anodotMetrics, err
	}

	if err != nil {
		log.Printf("Cloud not fetch ELB metrics from CLoudWatch : %v", err)
		return anodotMetrics, err
	}

	for _, m := range metrics {
		for _, mr := range metricdataresults {
			if *mr.Id == m.MStat.Id {
				e := m.Resource.(ELB)
				//log.Printf("Fetching CloudWatch metric: %s for ELB Id %s \n", m.MStat.Name, e.Name)
				anodotMetrics = append(anodotMetrics, GetAnodotMetric(m.MStat.Name, mr.Timestamps, mr.Values, GetELBMetricProperties(e))...)
			}
		}
	}
	return anodotMetrics, nil
}

func GetEc2Metrics(session *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
	anodotMetrics := make([]metricsAnodot.Anodot20Metric, 0)
	instanceFetcher := CreateEC2Fetcher(session)
	cloudWatchFetcher := CloudWatchFetcher{
		cloudwatchSvc: cloudwatchSvc,
	}

	for _, t := range resource.Tags {
		instanceFetcher.SetTag(t.Name, t.Value)
	}
	instances, err := instanceFetcher.GetInstances()
	if err != nil {
		log.Printf("Could not fetch EC2 instances from AWS %v", err)
		return anodotMetrics, err
	}

	log.Printf("Found %d instances to process \n", len(instances))
	metrics, err := GetEc2CloudwatchMetrics(resource, instances)
	if err != nil {
		log.Printf("Error: %v", err)
		return anodotMetrics, err
	}
	if len(metrics) > 0 {
		metricdatainput := NewGetMetricDataInput(metrics)
		metricdataresults, err := cloudWatchFetcher.FetchMetrics(metricdatainput)
		if err != nil {
			log.Printf("Error during EC2 metrics processing: %v", err)
			return anodotMetrics, err
		}

		for _, m := range metrics {
			for _, mr := range metricdataresults {
				if *mr.Id == m.MStat.Id {
					i := m.Resource.(Instance)
					log.Printf("Fetching CloudWatch metric: %s for: instance Id %s \n", m.MStat.Name, i.InstanceId)
					anodotMetrics = append(anodotMetrics, GetAnodotMetric(m.MStat.Name, mr.Timestamps, mr.Values, GetEc2MetricProperties(i))...)
				}
			}
		}
	}
	if len(resource.CustomMetrics) > 0 {
		for _, cm := range resource.CustomMetrics {
			if cm == "CoreCount" {
				log.Printf("Processing EC2 custom metric CoreCount\n")
				anodotMetrics = append(anodotMetrics, getCpuCountMetric(instances)...)
			}
		}
	}
	return anodotMetrics, nil
}

func GetAnodotMetric(name string, timestemps []*time.Time, values []*float64, properties map[string]string) []metricsAnodot.Anodot20Metric {
	properties["metric_version"] = metricVersion
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

func escape(s string) string {
	return strings.ReplaceAll(s, ":", "_")
}

func LambdaHandler() {
	c, err := GetConfig()
	if err != nil {
		log.Fatalf("Could not parse config: %v", err)
	}

	region := c.Region
	session := session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
	cloudwatchSvc := cloudwatch.New(session)

	anodotMetrics := make([]metricsAnodot.Anodot20Metric, 0)

	url, err := url.Parse(c.AnodotUrl)
	if err != nil {
		log.Fatalf("Could not parse Anodot url: %v", err)
	}

	metricSubmitter, err := metrics.NewAnodot20Client(*url, c.AnodotToken, nil)
	if err != nil {
		log.Fatalf("Could create Anodot metrc submitter: %v", err)
	}

	for _, r := range c.Resources {
		if r.Name == "EC2" {
			ec2metrics, err := GetEc2Metrics(session, cloudwatchSvc, r)
			if err != nil {
				log.Printf("Error: can't get ec2 metrics: %v", err)
			} else {
				log.Printf("Got %d metrics for EC2", len(ec2metrics))
				anodotMetrics = append(anodotMetrics, ec2metrics...)
			}
		}
		if r.Name == "ELB" {
			elbmetrics, err := GetELBMetrics(session, cloudwatchSvc, r)
			if err != nil {
				log.Printf("Error: can't get elb metrics: %v", err)
			} else {
				log.Printf("Got %d metrics for ELB", len(elbmetrics))
				anodotMetrics = append(anodotMetrics, elbmetrics...)
			}
		}
		if r.Name == "EBS" {
			ebsmetrics, err := GetEBSMetrics(session, cloudwatchSvc, r)
			if err != nil {
				log.Printf("Error: can't get EBS metrics: %v", err)
			} else {
				log.Printf("Got %d metrics for EBS", len(ebsmetrics))
				anodotMetrics = append(anodotMetrics, ebsmetrics...)
			}
		}
		if r.Name == "S3" {
			s3metrics, err := GetS3Metrics(session, cloudwatchSvc, r)
			if err != nil {
				log.Printf("Error: can't get S3 metrics: %v", err)
			} else {
				log.Printf("Got %d metrics for S3", len(s3metrics))
				anodotMetrics = append(anodotMetrics, s3metrics...)
			}
		}
	}

	if len(anodotMetrics) > 0 {
		response, err := metricSubmitter.SubmitMetrics(anodotMetrics)
		if err != nil {
			fmt.Printf("Error during sending metrics to Anodot", err)
		}

		if response.HasErrors() {
			log.Fatalf("Failed to push metrics to anodot: %v", response.RawResponse())
		} else {
			log.Printf("Successfully pushed %d metric to anodot \n", len(anodotMetrics))
		}

	} else {
		log.Print("No any metrics to push ")
	}
}

func main() {
	lambda.Start(LambdaHandler)
}
