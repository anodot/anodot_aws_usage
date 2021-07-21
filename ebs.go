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

type EBS struct {
	Id            string
	Tags          []*ec2.Tag
	Type          string
	State         string
	AZ            string
	IOPS          int64
	Region        string
	Size          int64
	DimensionTags []string
}

func DescribeVolumes(deafaultfilters map[string]string, ec2svc *ec2.EC2) ([]*ec2.Volume, error) {
	filters := make([]*ec2.Filter, 0)
	var nexttoken *string = nil
	volumes := make([]*ec2.Volume, 0)

	for name, value := range deafaultfilters {
		filters = append(filters, &ec2.Filter{
			Name: aws.String(name),
			Values: []*string{
				aws.String(value),
			},
		})
	}

	for {
		resultavailable, err := ec2svc.DescribeVolumes(getDecribeInput(filters, nexttoken))
		if err != nil {
			return nil, err
		}
		volumes = append(volumes, resultavailable.Volumes...)
		nexttoken = resultavailable.NextToken
		if nexttoken == nil {
			break
		}
	}

	return volumes, nil
}

func getDecribeInput(filters []*ec2.Filter, nexttoken *string) *ec2.DescribeVolumesInput {
	maxresult := int64(500)
	input := &ec2.DescribeVolumesInput{
		Filters:    filters,
		MaxResults: &maxresult,
	}
	if nexttoken != nil {
		input.NextToken = nexttoken
	}
	return input
}

func GetEBSVolumes(session *session.Session, customtags []Tag, resource *MonitoredResource) ([]EBS, error) {
	ebslist := make([]EBS, 0)
	ec2svc := ec2.New(session)
	volumes := make([]*ec2.Volume, 0)
	region := session.Config.Region

	deafultfilters := []map[string]string{
		map[string]string{
			"status": "available",
		},
		map[string]string{
			"status": "in-use",
		},
	}

	for _, filter := range deafultfilters {
		for _, tag := range customtags {
			filter["tag:"+tag.Name] = tag.Value
		}

		result, err := DescribeVolumes(filter, ec2svc)

		if err != nil {
			return ebslist, err
		}
		volumes = append(volumes, result...)
	}

	for _, v := range volumes {
		ebs := EBS{
			Id:            *v.VolumeId,
			Type:          *v.VolumeType,
			Size:          *v.Size,
			AZ:            *v.AvailabilityZone,
			Region:        *region,
			IOPS:          0,
			State:         *v.State,
			DimensionTags: resource.DimensionTags,
		}

		if v.Iops != nil {
			ebs.IOPS = *v.Iops
		}

		ebslist = append(ebslist, ebs)
	}
	return ebslist, nil
}

func GetEBSDimensions(resource *MonitoredResource) []string {
	dims := []string{
		"service",
		"volume_id",
		"ebs_type",
		"state",
		"availability_zone",
		"iops",
		"region",
		"anodot-collector",
	}
	return removeDuplicates(append(dims, resource.DimensionTags...))
}

func GetEBSCustomMetrics() []CustomMetricDefinition {

	return []CustomMetricDefinition{
		CustomMetricDefinition{
			Name:       "size",
			Alias:      "Size",
			TargetType: "sum",
		},
	}
}

func GetEBSMetricProperties(ebs EBS) map[string]string {
	properties := map[string]string{
		"service":           "ebs",
		"volume_id":         ebs.Id,
		"ebs_type":          ebs.Type,
		"state":             ebs.State,
		"availability_zone": ebs.AZ,
		"iops":              strconv.Itoa(int(ebs.IOPS)),
		"region":            ebs.Region,
		"anodot-collector":  "aws",
	}

	for _, v := range ebs.Tags {
		for _, dt := range ebs.DimensionTags {
			fmt.Println(*v.Key)
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

func getEBSSizeMetric(ebs []EBS) []metrics3.AnodotMetrics30 {
	metrics := make([]metrics3.AnodotMetrics30, 0)
	for _, e := range ebs {

		/*if accountId != "" {
			properties["account_id"] = accountId
		}*/

		metric := metrics3.AnodotMetrics30{
			Dimensions:   GetEBSMetricProperties(e),
			Timestamp:    metrics3.AnodotTimestamp{time.Now()},
			Measurements: map[string]float64{"size": float64(e.Size)},
		}
		metrics = append(metrics, metric)
	}
	return metrics
}

func GetEBSMetrics30(session *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metrics3.AnodotMetrics30, error) {
	metrics := make([]metrics3.AnodotMetrics30, 0)
	ebss, err := GetEBSVolumes(session, resource.Tags, resource)

	if err != nil {
		log.Printf("Cloud not describe EBS volumes %v", err)
		return metrics, err
	}
	log.Printf("Got %d EBS volumes to process", len(ebss))
	if len(resource.CustomMetrics) > 0 {
		for _, cm := range resource.CustomMetrics {
			if cm == "Size" {
				log.Printf("Processing EBS custom metric Size\n")
				metrics = append(metrics, getEBSSizeMetric(ebss)...)
			}
		}
	}
	return metrics, nil
}
