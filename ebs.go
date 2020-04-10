package main

import (
	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"strconv"
	"time"
)

type EBS struct {
	Id     string
	Tags   []*ec2.Tag
	Type   string
	State  string
	AZ     string
	IOPS   int64
	Region string
	Size   int64
}

func GetEBSVolumes(session *session.Session) ([]EBS, error) {
	ebslist := make([]EBS, 0)
	ec2svc := ec2.New(session)
	volumes := make([]*ec2.Volume, 0)
	region := session.Config.Region

	availablefilter := []*ec2.Filter{
		&ec2.Filter{
			Name: aws.String("status"),
			Values: []*string{
				aws.String("available"),
			},
		},
	}
	inusefilter := []*ec2.Filter{
		&ec2.Filter{
			Name: aws.String("status"),
			Values: []*string{
				aws.String("in-use"),
			},
		},
	}

	resultavailable, err := ec2svc.DescribeVolumes(&ec2.DescribeVolumesInput{
		Filters: availablefilter,
	})
	if err != nil {
		return ebslist, err
	}

	resultinuse, err := ec2svc.DescribeVolumes(&ec2.DescribeVolumesInput{
		Filters: inusefilter,
	})
	if err != nil {
		return ebslist, err
	}
	volumes = append(volumes, resultinuse.Volumes...)
	volumes = append(volumes, resultavailable.Volumes...)

	for _, v := range volumes {
		ebs := EBS{
			Id:     *v.VolumeId,
			Type:   *v.VolumeType,
			Size:   *v.Size,
			AZ:     *v.AvailabilityZone,
			Region: *region,
			IOPS:   0,
			State: *v.State,
		}

		if *v.VolumeType != "standard" {
			ebs.IOPS = *v.Iops
		}

		ebslist = append(ebslist, ebs)
	}
	return ebslist, nil
}

func GetEBSMetricProperties(ebs EBS) map[string]string {
	properties := map[string]string{
		"service":          "ebs",
		"volume_id":        ebs.Id,
		"ebs_type":         ebs.Type,
		"state":            ebs.State,
		"availability_zone": ebs.AZ,
		"iops":             strconv.Itoa(int(ebs.IOPS)),
		"region":           ebs.Region,
	}

	for _, v := range ebs.Tags {
		if len(*v.Key) > 50 || len(*v.Value) < 2 {
			continue
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

func getEBSSizeMetric(ebs []EBS) []metricsAnodot.Anodot20Metric {
	metricList := make([]metricsAnodot.Anodot20Metric, 0)
	for _, e := range ebs {
		properties := GetEBSMetricProperties(e)
		properties["what"] = "size"
		properties["metric_version"] = metricVersion
		metric := metricsAnodot.Anodot20Metric{
			Properties: properties,
			Value:      float64(e.Size),
			Timestamp: metricsAnodot.AnodotTimestamp{
				time.Now(),
			},
		}
		metricList = append(metricList, metric)
	}
	return metricList
}
