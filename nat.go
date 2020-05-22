package main

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type NatGateway struct {
	NatGatewayId *string
	VpcId        *string
	SubnetId     *string
	Tags         []*ec2.Tag
	State        *string
}

func DescribeNatGateways(session *session.Session) ([]NatGateway, error) {
	var nexttoken *string = nil
	gateways := make([]NatGateway, 0)
	maxcount := int64(900)
	client := ec2.New(session)
	rawgateways := make([]*ec2.NatGateway, 0)

	input := &ec2.DescribeNatGatewaysInput{
		MaxResults: &maxcount,
	}

	for {
		req, output := client.DescribeNatGatewaysRequest(input)
		err := req.Send()
		if err != nil {
			return gateways, err
		}
		rawgateways = append(rawgateways, output.NatGateways...)
		nexttoken = output.NextToken
		if nexttoken == nil {
			break
		}
	}

	if len(rawgateways) == 0 {
		return gateways, fmt.Errorf("Can not find any nat gateways")
	}

	for _, g := range rawgateways {
		gateway := NatGateway{
			NatGatewayId: g.NatGatewayId,
			VpcId:        g.VpcId,
			SubnetId:     g.SubnetId,
			State:        g.State,
			Tags:         g.Tags,
		}
		gateways = append(gateways, gateway)
	}
	return gateways, nil
}

func GetNatGatewayMetricProperties(gateway NatGateway) map[string]string {
	properties := map[string]string{
		"service":          "natgateway",
		"NatGatewayId":     *gateway.NatGatewayId,
		"VpcId":            *gateway.VpcId,
		"SubnetId":         *gateway.SubnetId,
		"State":            *gateway.State,
		"anodot-collector": "aws",
	}

	for _, v := range gateway.Tags {
		if len(*v.Key) > 50 || len(*v.Value) < 2 {
			continue
		}
		if len(properties) == 17 {
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

func GetNatGatewayCloudwatchMetrics(resource *MonitoredResource, gateways []NatGateway) ([]MetricToFetch, error) {
	metrics := make([]MetricToFetch, 0)

	for _, mstat := range resource.Metrics {
		for _, g := range gateways {
			m := MetricToFetch{}
			m.Dimensions = []Dimension{
				Dimension{
					Name:  "NatGatewayId",
					Value: *g.NatGatewayId,
				},
			}
			m.Resource = g
			mstatCopy := mstat
			mstatCopy.Id = "nat" + strconv.Itoa(len(metrics))
			m.MStat = mstatCopy
			metrics = append(metrics, m)
		}
	}

	return metrics, nil
}
