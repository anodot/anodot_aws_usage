package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/anodot/anodot-common/pkg/metrics"
	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
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

func (ec2fetcher *EC2Fetcher) GetInstances() (ListInstances, error) {
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
		if *i.State.Code != 16 {
			fmt.Printf("Instance %s in not running state: %s \n", *i.InstanceId, *i.State.Name)
			continue
		}
		lifecycle := "normal"
		if i.InstanceLifecycle != nil {
			lifecycle = *i.InstanceLifecycle
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
			VpcId:              *i.VpcId,
			VirtualizationType: *i.VirtualizationType,
			Region:             ec2fetcher.region,
			Lifecycle:          lifecycle,
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
	}

	for _, v := range ins.Tags {
		if len(*v.Key) > 50 || len(*v.Value) < 2 {
			continue
		}
		if len(properties) == 18 {
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

func getCpuCountMetric(ins []Instance) []metricsAnodot.Anodot20Metric {
	metricList := make([]metricsAnodot.Anodot20Metric, 0)
	for _, i := range ins {
		properties := GetEc2MetricProperties(i)
		if accountId != "" {
			properties["account_id"] = accountId
		}
		properties["metric_version"] = metricVersion
		properties["what"] = "cpu_count"
		metric := metrics.Anodot20Metric{
			Properties: properties,
			Value:      float64(i.CoreCount),
			Timestamp: metrics.AnodotTimestamp{
				time.Now(),
			},
		}
		metricList = append(metricList, metric)
	}

	return metricList
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
