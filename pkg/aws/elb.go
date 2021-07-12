package aws

import (
	"log"
	"strconv"

	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

var pageSize int64 = 400

type LoadBalancerTag struct {
	Key   string
	Value string
}

type LoadBalancer struct {
	Name   string
	Az     string
	VPCId  string
	Type   string
	Tags   []LoadBalancerTag
	Region string
}

func GetLoadBalancers(session *session.Session) ([]LoadBalancer, error) {
	balancers := make([]LoadBalancer, 0)
	netandapp, err := GetAppAndNetworkBalancers(session)
	if err != nil {
		return balancers, err
	}
	balancers = append(balancers, netandapp...)
	classic, err := GetClassicBalancers(session)
	if err != nil {
		return balancers, err
	}
	balancers = append(balancers, classic...)
	return balancers, nil
}

func GetAppAndNetworkBalancers(session *session.Session) ([]LoadBalancer, error) {
	elbSvc := elbv2.New(session)
	region := session.Config.Region
	balancers := make([]LoadBalancer, 0)

	input := &elbv2.DescribeLoadBalancersInput{
		PageSize: &pageSize,
	}

	result, err := elbSvc.DescribeLoadBalancers(input)
	if err != nil {
		return nil, err
	}

	for _, lb := range result.LoadBalancers {
		desctagsoutput, err := elbSvc.DescribeTags(&elbv2.DescribeTagsInput{
			ResourceArns: []*string{lb.LoadBalancerArn},
		})

		if err != nil {
			log.Printf("Could not get tags for %s", *lb.DNSName)
			continue
		}

		balancers = append(balancers, LoadBalancer{
			Name:   *lb.LoadBalancerName,
			Az:     *lb.AvailabilityZones[0].ZoneName,
			VPCId:  *lb.VpcId,
			Type:   *lb.Type,
			Tags:   convertTags(desctagsoutput.TagDescriptions[0].Tags),
			Region: *region,
		})

	}
	return balancers, nil
}

func GetClassicBalancers(session *session.Session) ([]LoadBalancer, error) {
	elbSvc := elb.New(session)
	region := session.Config.Region
	balancers := make([]LoadBalancer, 0)

	input := &elb.DescribeLoadBalancersInput{
		PageSize: &pageSize,
	}

	result, err := elbSvc.DescribeLoadBalancers(input)
	if err != nil {
		return nil, err
	}

	for _, lb := range result.LoadBalancerDescriptions {
		desctagsoutput, err := elbSvc.DescribeTags(&elb.DescribeTagsInput{
			LoadBalancerNames: []*string{lb.LoadBalancerName},
		})

		if err != nil {
			log.Printf("Could not get tags for %s", *lb.DNSName)
			continue
		}

		balancers = append(balancers, LoadBalancer{
			Name:   *lb.LoadBalancerName,
			Az:     *lb.AvailabilityZones[0],
			VPCId:  *lb.VPCId,
			Type:   "classic",
			Tags:   convertTags(desctagsoutput.TagDescriptions[0].Tags),
			Region: *region,
		})

	}
	return balancers, nil
}

func GetELBMetricProperties(elb LoadBalancer) map[string]string {
	properties := map[string]string{
		"service":          "elb",
		"name":             elb.Name,
		"az":               elb.Az,
		"vpc_id":           elb.VPCId,
		"region":           elb.Region,
		"anodot-collector": "aws",
		"type":             elb.Type,
	}

	for _, v := range elb.Tags {
		if len(v.Key) > 50 || len(v.Value) < 2 {
			continue
		}
		if len(properties) == 17 {
			break
		}
		properties[escape(v.Key)] = escape(v.Value)
	}

	for k, v := range properties {
		if len(v) > 50 || len(v) < 2 {
			delete(properties, k)
		}
	}
	return properties
}

func GetELBCloudwatchMetrics(resource *MonitoredResource, elbs []LoadBalancer) ([]MetricToFetch, error) {
	metrics := make([]MetricToFetch, 0)

	for _, mstat := range resource.Metrics {
		for _, elb := range elbs {
			m := MetricToFetch{}
			m.Dimensions = []Dimension{
				Dimension{
					Name:  "LoadBalancerName",
					Value: elb.Name,
				},
			}
			m.Resource = elb
			mstatCopy := mstat
			mstatCopy.Id = "elb" + strconv.Itoa(len(metrics))
			m.MStat = mstatCopy
			metrics = append(metrics, m)
		}
	}
	return metrics, nil
}

func convertTags(tags interface{}) []LoadBalancerTag {
	blancertags := make([]LoadBalancerTag, 0)
	if tags_, ok := tags.([]*elbv2.Tag); ok {
		for _, tag := range tags_ {
			blancertags = append(blancertags, LoadBalancerTag{Key: *tag.Key, Value: *tag.Value})
		}
	}

	if tags_, ok := tags.([]*elb.Tag); ok {
		for _, tag := range tags_ {
			blancertags = append(blancertags, LoadBalancerTag{Key: *tag.Key, Value: *tag.Value})
		}
	}
	return blancertags
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
