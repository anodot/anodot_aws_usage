package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/anodot/anodot-common/pkg/metrics3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type Instance struct {
	InstanceId         string
	InstanceType       string
	Monitoring         string
	AvailabilityZone   string
	GroupName          string
	Tags               []*ec2.Tag
	State              string
	VpcId              string
	VirtualizationType string
	CoreCount          int64
	ThreadsPerCore     int64
	Region             string
	Lifecycle          string
	DimensionTags      []string
}

type Filters []*ec2.Filter
type ListInstances []Instance

type Metrics struct {
	Tags      map[string]string
	Instances ListInstances
}

type EC2Fetcher struct {
	region          string
	filters         Filters
	instanceService *ec2.EC2
	tags            map[string]string //Set of tags which defines intances to be reported
}

func CreateEC2Fetcher(session *session.Session) EC2Fetcher {
	region := session.Config.Region
	f := &ec2.Filter{
		Name: aws.String("instance-state-code"),
		Values: []*string{
			aws.String("16"),
		},
	}
	ec2s := ec2.New(session)
	return EC2Fetcher{
		instanceService: ec2s,
		filters: []*ec2.Filter{
			f,
		},
		tags:   make(map[string]string, 0),
		region: *region,
	}
}

func (i *EC2Fetcher) setFilter(name string, value string) {
	f := &ec2.Filter{
		Name: aws.String("tag:" + name),
		Values: []*string{
			aws.String(value),
		},
	}
	i.filters = append(i.filters, f)
}

func (i *EC2Fetcher) SetTag(name string, value string) error {
	i.tags[name] = value
	i.setFilter(name, value)
	return nil
}

func (ec2fetcher *EC2Fetcher) GetInstances(resource *MonitoredResource) (ListInstances, error) {
	var li ListInstances
	var nexttoken *string = nil
	reservation := make([]*ec2.Reservation, 0)
	ec2list := make([]*ec2.Instance, 0)

	for {
		result, err := ec2fetcher.instanceService.DescribeInstances(getInput(ec2fetcher.filters, nexttoken))
		if err != nil {
			fmt.Println("Error", err)
			return nil, err
		}

		if len(result.Reservations) == 0 {
			fmt.Println("Not found any instances")
			return nil, fmt.Errorf("Error: Can not find any instances with this input params")
		}
		reservation = append(reservation, result.Reservations...)
		nexttoken = result.NextToken
		if nexttoken == nil {
			break
		}
	}

	for _, r := range reservation { //result.Reservations {
		ec2list = append(ec2list, r.Instances...)
	}

	for _, i := range ec2list {
		var vpcId string
		if *i.State.Code != 16 {
			fmt.Printf("Instance %s in not running state: %s \n", *i.InstanceId, *i.State.Name)
			continue
		}
		lifecycle := "normal"
		if i.InstanceLifecycle != nil {
			lifecycle = *i.InstanceLifecycle
		}

		if i.VpcId == nil {
			vpcId = "None"
		} else {
			vpcId = *i.VpcId
		}
		li = append(li, Instance{
			CoreCount:          *i.CpuOptions.CoreCount,
			ThreadsPerCore:     *i.CpuOptions.ThreadsPerCore,
			InstanceId:         *i.InstanceId,
			InstanceType:       *i.InstanceType,
			Tags:               i.Tags,
			Monitoring:         *i.Monitoring.State,
			AvailabilityZone:   *i.Placement.AvailabilityZone,
			GroupName:          *i.Placement.GroupName,
			State:              *i.State.Name,
			VpcId:              vpcId,
			VirtualizationType: *i.VirtualizationType,
			Region:             ec2fetcher.region,
			Lifecycle:          lifecycle,
			DimensionTags:      resource.DimensionTags,
		})

	}
	return ListInstances(li), nil
}

func getInput(fl Filters, nexttoken *string) *ec2.DescribeInstancesInput {
	maxresult := int64(1000)
	if len(fl) < 0 {
		return &ec2.DescribeInstancesInput{}
	}
	input := &ec2.DescribeInstancesInput{
		Filters:    fl,
		MaxResults: &maxresult,
	}
	if nexttoken != nil {
		input.NextToken = nexttoken
	}
	return input
}

func GetEc2Dimensions(resource *MonitoredResource) []string {
	dims := []string{
		"service",
		"instance_id",
		"instance_type",
		"monitoring",
		"availability_zone",
		"group_name",
		"state",
		"vpc_id",
		"virtualization_type",
		"threads_per_core",
		"region",
		"lifecycle",
		"anodot-collector",
	}
	return append(dims, resource.DimensionTags...)
}

func GetEc2CustomMetrics() []CustomMetricDefinition {
	return []CustomMetricDefinition{
		{
			Name:       "cpu_count",
			Alias:      "CoreCount",
			TargetType: "sum",
		},
		{
			Name:       "vcpu_count",
			Alias:      "CoreCount",
			TargetType: "sum",
		},
	}
}

func GetEc2MetricProperties(ins Instance) map[string]string {
	properties := map[string]string{
		"service":             "ec2",
		"instance_id":         ins.InstanceId,
		"instance_type":       ins.InstanceType,
		"monitoring":          ins.Monitoring,
		"availability_zone":   ins.AvailabilityZone,
		"group_name":          ins.GroupName,
		"state":               ins.State,
		"vpc_id":              ins.VpcId,
		"virtualization_type": ins.VirtualizationType,
		"threads_per_core":    strconv.Itoa(int(ins.ThreadsPerCore)),
		"region":              ins.Region,
		"lifecycle":           ins.Lifecycle,
		"anodot-collector":    "aws",
	}

	for _, v := range ins.Tags {
		for _, dt := range ins.DimensionTags {
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

func getCpuCountMetric30(ins []Instance) []metrics3.AnodotMetrics30 {
	metrics := make([]metrics3.AnodotMetrics30, 0)
	for _, i := range ins {

		/*if accountId != "" {
			properties["account_id"] = accountId
		}*/

		metric := metrics3.AnodotMetrics30{
			Dimensions:   GetEc2MetricProperties(i),
			Timestamp:    metrics3.AnodotTimestamp{time.Now()},
			Measurements: map[string]float64{"cpu_count": float64(i.CoreCount)},
		}
		metrics = append(metrics, metric)
	}

	return metrics
}

func getVirtualCpuCountMetric30(ins []Instance) []metrics3.AnodotMetrics30 {
	metrics := make([]metrics3.AnodotMetrics30, 0)

	for _, i := range ins {
		vcpu := i.CoreCount * i.ThreadsPerCore
		metric := metrics3.AnodotMetrics30{
			Dimensions:   GetEc2MetricProperties(i),
			Timestamp:    metrics3.AnodotTimestamp{time.Now()},
			Measurements: map[string]float64{"vcpu_count": float64(vcpu)},
		}
		metrics = append(metrics, metric)
	}

	return metrics
}

func GetEc2CloudwatchMetrics(resource *MonitoredResource, instances []Instance) ([]MetricToFetch, error) {
	metrics := make([]MetricToFetch, 0)

	for _, mstat := range resource.Metrics {
		for _, i := range instances {
			m := MetricToFetch{}
			m.Dimensions = []Dimension{
				Dimension{
					Name:  "InstanceId",
					Value: i.InstanceId,
				},
			}
			m.Resource = i
			mstatCopy := mstat
			mstatCopy.Id = "ec2" + strconv.Itoa(len(metrics))
			m.MStat = mstatCopy
			metrics = append(metrics, m)
		}
	}

	return metrics, nil
}

func GetEc2Metrics30(session *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metrics3.AnodotMetrics30, error) {
	metrics := make([]metrics3.AnodotMetrics30, 0)

	instanceFetcher := CreateEC2Fetcher(session)

	cloudWatchFetcher := CloudWatchFetcher{
		cloudwatchSvc: cloudwatchSvc,
	}
	instances, err := instanceFetcher.GetInstances(resource)
	if err != nil {
		log.Printf("Could not fetch EC2 instances from AWS %v", err)
		return metrics, err
	}

	log.Printf("Found %d instances to process \n", len(instances))
	cmetrics, err := GetEc2CloudwatchMetrics(resource, instances)
	if err != nil {
		log.Printf("Error: %v", err)
		return metrics, err
	}

	if len(cmetrics) > 0 {
		metricdatainput := NewGetMetricDataInput(cmetrics)
		metricdataresults, err := cloudWatchFetcher.FetchMetrics(metricdatainput)
		if err != nil {
			log.Printf("Error during EC2 metrics processing: %v", err)
			return metrics, err
		}

		for _, m := range cmetrics {
			for _, mr := range metricdataresults {
				if *mr.Id == m.MStat.Id {
					i := m.Resource.(Instance)
					metrics = append(
						metrics,
						GetAnodotMetric30(m.MStat.Name, mr.Timestamps, mr.Values, GetEc2MetricProperties(i))...,
					)
				}
			}
		}
	}

	if len(resource.CustomMetrics) > 0 {
		for _, cm := range resource.CustomMetrics {
			if cm == "CoreCount" {
				log.Printf("Processing EC2 custom metric CoreCount\n")
				metrics = append(metrics, getCpuCountMetric30(instances)...)
			}
			if cm == "VCpuCount" {
				log.Printf("Processing EC2 custom metric VCpuCount\n")
				metrics = append(metrics, getVirtualCpuCountMetric30(instances)...)
			}
		}
	}

	return metrics, nil
}
