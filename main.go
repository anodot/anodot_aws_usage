package main

import (
	"errors"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/anodot/anodot-common/pkg/metrics"
	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"

	//"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

const metricVersion string = "5"

var accountId string

func GetKinesisMetrics(ses *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
	anodotMetrics := make([]metricsAnodot.Anodot20Metric, 0)
	cloudWatchFetcher := CloudWatchFetcher{
		cloudwatchSvc: cloudwatchSvc,
	}
	streams, err := GetStreams(ses)
	if err != nil {
		return anodotMetrics, nil
	}

	metrics, err := GetKinesisStreamCloudwatchMetrics(resource, streams)
	if err != nil {
		return anodotMetrics, nil
	}

	metricdatainput := NewGetMetricDataInput(metrics)
	metricdataresults, err := cloudWatchFetcher.FetchMetrics(metricdatainput)
	if err != nil {
		log.Printf("Error during Kinesis metrics processing: %v", err)
		return anodotMetrics, err
	}

	for _, m := range metrics {
		for _, mr := range metricdataresults {
			if *mr.Id == m.MStat.Id {
				stream := m.Resource.(KinesisStream)
				anodot_stream_metrics := GetAnodotMetric(m.MStat.Name, mr.Timestamps, mr.Values, GetStreamMetricProperties(stream))
				anodotMetrics = append(anodotMetrics, anodot_stream_metrics...)
			}
		}
	}

	return anodotMetrics, nil
}

func GetEfsMetrics(ses *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
	anodotMetrics := make([]metricsAnodot.Anodot20Metric, 0)

	cloudWatchFetcher := CloudWatchFetcher{
		cloudwatchSvc: cloudwatchSvc,
	}
	efss, err := DesribeFilesystems(ses)
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
				anodot_efs_metrics := GetAnodotMetric(m.MStat.Name, mr.Timestamps, mr.Values, GetEfsMetricProperties(efs))
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

func GetDynamoDbMetrics(ses *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
	anodotMetrics := make([]metricsAnodot.Anodot20Metric, 0)

	cloudWatchFetcher := CloudWatchFetcher{
		cloudwatchSvc: cloudwatchSvc,
	}
	tables, err := ListTables(ses)
	if err != nil {
		log.Printf("Cloud not get list Dynamo DB tables : %v", err)
		return anodotMetrics, nil
	}
	log.Printf("Found %d Dynamo DB tables ", len(tables))
	metrics, err := GetDynamoCloudwatchMetrics(resource, tables)
	if err != nil {
		log.Printf("Error: %v", err)
		return anodotMetrics, err
	}

	metricdatainput := NewGetMetricDataInput(metrics)
	metricdataresults, err := cloudWatchFetcher.FetchMetrics(metricdatainput)
	if err != nil {
		log.Printf("Error during DynamoDB metrics processing: %v", err)
		return anodotMetrics, err
	}

	for _, m := range metrics {
		for _, mr := range metricdataresults {
			if *mr.Id == m.MStat.Id {

				table := m.Resource.(DynamoTable)
				properties := GetDynamoProperties(table)
				for _, d := range m.Dimensions {
					if d.Name == "Operation" {
						properties["operation"] = d.Value
					}
				}

				anodot_dynamo_metrics := GetAnodotMetric(m.MStat.Name, mr.Timestamps, mr.Values, properties)
				anodotMetrics = append(anodotMetrics, anodot_dynamo_metrics...)
			}
		}
	}
	return anodotMetrics, nil
}

func GetCloudfrontMetrics(ses *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
	if resource.CustomRegion != "" {
		cloudwatchSvc = cloudwatch.New(session.Must(session.NewSession(&aws.Config{Region: aws.String(resource.CustomRegion)})))
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

func GetEBSMetrics(session *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
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

func GetELBMetrics(session *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
	cloudWatchFetcher := CloudWatchFetcher{
		cloudwatchSvc: cloudwatchSvc,
	}

	anodotMetrics := make([]metricsAnodot.Anodot20Metric, 0)
	elbs, err := GetLoadBalancers(session)
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
		log.Printf("Cloud not fetch ELB metrics from CLoudWatch : %v", err)
		return anodotMetrics, err
	}

	for _, m := range metrics {
		for _, mr := range metricdataresults {
			if *mr.Id == m.MStat.Id {
				e := m.Resource.(LoadBalancer)
				//log.Printf("Fetching CloudWatch metric: %s for ELB Id %s \n", m.MStat.Name, e.Name)
				anodot_cloudwatch_metrics := GetAnodotMetric(m.MStat.Name, mr.Timestamps, mr.Values, GetELBMetricProperties(e))
				anodotMetrics = append(anodotMetrics, anodot_cloudwatch_metrics...)
			}
		}
	}
	return anodotMetrics, nil
}

func GetNatGatewayMetrics(session *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
	anodotMetrics := make([]metricsAnodot.Anodot20Metric, 0)
	cloudWatchFetcher := CloudWatchFetcher{
		cloudwatchSvc: cloudwatchSvc,
	}
	gateways, err := DescribeNatGateways(session)
	if err != nil {
		return anodotMetrics, err
	}
	metrics, err := GetNatGatewayCloudwatchMetrics(resource, gateways)
	if err != nil {
		return anodotMetrics, err
	}
	if len(metrics) > 0 {
		metricdatainput := NewGetMetricDataInput(metrics)
		metricdataresults, err := cloudWatchFetcher.FetchMetrics(metricdatainput)
		if err != nil {
			return anodotMetrics, err
		}

		for _, m := range metrics {
			for _, mr := range metricdataresults {
				if *mr.Id == m.MStat.Id {
					n := m.Resource.(NatGateway)
					anodot_cloudwatch_metrics := GetAnodotMetric(m.MStat.Name, mr.Timestamps, mr.Values, GetNatGatewayMetricProperties(n))
					anodotMetrics = append(anodotMetrics, anodot_cloudwatch_metrics...)

				}
			}
		}
	}
	return anodotMetrics, nil
}

func GetEc2Metrics(session *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
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
					properties := GetEc2MetricProperties(i)
					properties["target_type"] = "counter"
					//log.Printf("Fetching CloudWatch metric: %s for: instance Id %s \n", m.MStat.Name, i.InstanceId)
					anodot_cloudwatch_metrics := GetAnodotMetric(m.MStat.Name, mr.Timestamps, mr.Values, properties)
					anodotMetrics = append(anodotMetrics, anodot_cloudwatch_metrics...)

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
			if cm == "VCpuCount" {
				log.Printf("Processing EC2 custom metric VCpuCount\n")
				anodotMetrics = append(anodotMetrics, getVirtualCpuCountMetric(instances)...)
			}
		}
	}
	return anodotMetrics, nil
}

func GetAnodotMetric(name string, timestemps []*time.Time, values []*float64, properties map[string]string) []metricsAnodot.Anodot20Metric {
	properties["metric_version"] = metricVersion
	if accountId != "" {
		properties["account_id"] = accountId
	}

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

func SendMetrics(metrics []metricsAnodot.Anodot20Metric, submiter *metrics.Anodot20Client) error {
	response, err := submiter.SubmitMetrics(metrics)

	if err != nil || response.HasErrors() {
		log.Fatalf("Error during sending metrics to Anodot response: %v   Error: %v", response.RawResponse(), err)
		if response.HasErrors() {
			return errors.New(response.ErrorMessage())
		}
	} else {
		log.Printf("Successfully pushed %d metric to anodot \n", len(metrics))
	}
	return err
}

func LambdaHandler() {
	c, err := GetConfig()
	if err != nil {
		log.Fatalf("Could not parse config: %v", err)
	}
	ml := &SyncMetricList{
		metrics: make([]metricsAnodot.Anodot20Metric, 0),
	}

	el := &ErrorList{
		errors: make([]error, 0),
	}

	accountId = c.AccountId
	var wg sync.WaitGroup

	session := session.Must(session.NewSession(&aws.Config{Region: aws.String(c.Region)}))
	cloudwatchSvc := cloudwatch.New(session)

	url, err := url.Parse(c.AnodotUrl)
	if err != nil {
		log.Fatalf("Could not parse Anodot url: %v", err)
	}

	metricSubmitter, err := metrics.NewAnodot20Client(*url, c.AnodotToken, nil)
	if err != nil {
		log.Fatalf("Could create Anodot metrc submitter: %v", err)
	}

	Handle(c.RegionsConfigs[c.Region].Resources, &wg, session, cloudwatchSvc, ml, el)
	wg.Wait()

	if len(el.errors) > 0 {
		for _, e := range el.errors {
			log.Printf("ERROR occured: %v", e)
		}
	}

	if len(ml.metrics) > 0 {
		err := SendMetrics(ml.metrics, metricSubmitter)
		if err != nil {
			log.Printf("Retry sending metrics to anodot ... ")
			_ = SendMetrics(ml.metrics, metricSubmitter)
		}
	} else {
		log.Print("No any metrics to push ")
	}
}

func main() {
	lambda.Start(LambdaHandler)
}
