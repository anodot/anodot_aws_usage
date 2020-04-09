package main

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"strconv"
)

var pageSize int64 = 250

type ELB struct {
	Name  string
	Az    string
	VPCId string
	//Type  string
	Tags   []*elb.Tag
	Region string
}

func GetELBs(session *session.Session) ([]ELB, error) {
	region := session.Config.Region
	elbSvc := elb.New(session)
	input := &elb.DescribeLoadBalancersInput{
		PageSize: &pageSize,
	}
	result, err := elbSvc.DescribeLoadBalancers(input)
	if err != nil {
		return nil, err
	}
	elbs := make([]ELB, 0)
	elbnames := make([]*string, 0)
	for _, elb := range result.LoadBalancerDescriptions {
		elbnames = append(elbnames, elb.LoadBalancerName)
	}

	tagsdescriptions := make([]*elb.TagDescription, 0)
	inew := 0
	for i := 0; i < len(elbnames); i++ {
		inew = i + 20
		if len(elbnames) <= inew {
			inew = len(elbnames)
		}

		desctagsoutput, err := elbSvc.DescribeTags(&elb.DescribeTagsInput{
			LoadBalancerNames: elbnames[i:inew],
		})
		if err != nil {
			return nil, err
		}
		tagsdescriptions = append(tagsdescriptions, desctagsoutput.TagDescriptions...)
		i = inew
	}

	for _, elb := range result.LoadBalancerDescriptions {
		for _, td := range tagsdescriptions {
			if *elb.LoadBalancerName == *td.LoadBalancerName {
				elbs = append(elbs, ELB{
					Name:   *elb.LoadBalancerName,
					Az:     *elb.AvailabilityZones[0],
					VPCId:  *elb.VPCId,
					Tags:   td.Tags,
					Region: *region,
				})
			}
		}
	}
	return elbs, nil
}

func GetELBMetricProperties(elb ELB) map[string]string {
	properties := map[string]string{
		"service": "elb",
		"name":    elb.Name,
		"az":      elb.Az,
		"vpcid":   elb.VPCId,
		"region":  elb.Region,
	}

	for _, v := range elb.Tags {
		if len(*v.Key) > 50 || len(*v.Value) < 2 {
			continue
		}
		properties[*v.Key] = *v.Value
	}

	for k, v := range properties {
		if len(v) > 50 || len(v) < 2 {
			delete(properties, k)
		}
	}
	return properties
}

func GetELBCloudwatchMetrics(resource MonitoredResource, elbs []ELB) ([]MetricToFetch, error) {
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
