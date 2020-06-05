package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

type KinesisStream struct {
	Name   string
	Region string
}

func GetStreams(session *session.Session) ([]KinesisStream, error) {
	streams := make([]KinesisStream, 0)
	region := session.Config.Region

	input := &kinesis.ListStreamsInput{}
	kinesisSvc := kinesis.New(session)
	output, err := kinesisSvc.ListStreams(input)
	if err != nil {
		log.Printf("Error occured during Kinesis stream fetching %v", err)
		return streams, err
	}
	if len(output.StreamNames) == 0 {
		return streams, fmt.Errorf("Can't find anys Kinesis streams ")
	}
	for _, stream := range output.StreamNames {
		streams = append(streams, KinesisStream{Name: *stream, Region: *region})
	}
	return streams, nil
}

func GetStreamMetricProperties(stream KinesisStream) map[string]string {
	return map[string]string{
		"service":          "kinesis",
		"StreamName":       stream.Name,
		"anodot-collector": "aws",
		"region":           stream.Region,
	}
}

func GetKinesisStreamCloudwatchMetrics(resource *MonitoredResource, streams []KinesisStream) ([]MetricToFetch, error) {
	metrics := make([]MetricToFetch, 0)

	for _, mstat := range resource.Metrics {
		for _, stream := range streams {
			m := MetricToFetch{}
			m.Dimensions = []Dimension{
				Dimension{
					Name:  "StreamName",
					Value: stream.Name,
				},
			}
			m.Resource = streams
			mstatCopy := mstat
			mstatCopy.Id = "stream" + strconv.Itoa(len(metrics))
			m.MStat = mstatCopy
			metrics = append(metrics, m)
		}
	}
	return metrics, nil
}
