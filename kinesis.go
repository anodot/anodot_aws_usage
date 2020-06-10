package main

import (
	"fmt"
	"log"
	"strconv"

	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
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
