package main

var metric_templates = map[string]MetricTemplate{
	"BytesDownloaded": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "Cloudfront", Template: CloudwatchMetricTemplate{
		Metricname: "BytesDownloaded",
		Period:     "3600",
		Unit:       "None",
		Namespace:  "AWS/CloudFront",
		Id:         "test1",
		Stat:       "Average",
	},
	},
	"Requests": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "Cloudfront", Template: CloudwatchMetricTemplate{
		Metricname: "Requests",
		Period:     "3600",
		Unit:       "None",
		Namespace:  "AWS/CloudFront",
		Id:         "test1",
		Stat:       "Average",
	},
	},
	"TotalErrorRate": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "Cloudfront", Template: CloudwatchMetricTemplate{
		Metricname: "TotalErrorRate",
		Period:     "3600",
		Unit:       "None",
		Namespace:  "AWS/CloudFront",
		Id:         "test1",
		Stat:       "Average",
	},
	},
	"BucketSizeBytes": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "S3", Template: CloudwatchMetricTemplate{
		Metricname: "BucketSizeBytes",
		Period:     "86400",
		Unit:       "Bytes",
		Namespace:  "AWS/S3",
		Id:         "test1",
		Stat:       "Average",
	},
	},
	"NetworkIn": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "EC2", Template: CloudwatchMetricTemplate{
		Metricname: "NetworkIn",
		Period:     "3600",
		Unit:       "Bytes",
		Namespace:  "AWS/EC2",
		Id:         "test1",
		Stat:       "Average",
	},
	},
	"NetworkOut": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "EC2", Template: CloudwatchMetricTemplate{
		Metricname: "NetworkOut",
		Period:     "3600",
		Unit:       "Bytes",
		Namespace:  "AWS/EC2",
		Id:         "test1",
		Stat:       "Average",
	},
	},
	"NumberOfObjects": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "S3", Template: CloudwatchMetricTemplate{
		Metricname: "NumberOfObjects",
		Period:     "86400",
		Unit:       "Count",
		Namespace:  "AWS/S3",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"AllRequests": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "S3", Template: CloudwatchMetricTemplate{
		Metricname: "AllRequests",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/S3",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"GetRequests": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "S3", Template: CloudwatchMetricTemplate{
		Metricname: "GetRequests",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/S3",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"PutRequests": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "S3", Template: CloudwatchMetricTemplate{
		Metricname: "PutRequests",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/S3",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"DeleteRequests": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "S3", Template: CloudwatchMetricTemplate{
		Metricname: "DeleteRequests",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/S3",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"HeadRequests": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "S3", Template: CloudwatchMetricTemplate{
		Metricname: "HeadRequests",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/S3",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"SelectRequests": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "S3", Template: CloudwatchMetricTemplate{
		Metricname: "SelectRequests",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/S3",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"ListRequests": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "S3", Template: CloudwatchMetricTemplate{
		Metricname: "ListRequests",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/S3",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"RequestCount": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "ELB", Template: CloudwatchMetricTemplate{
		Metricname: "RequestCount",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/ELB",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"EstimatedProcessedBytes": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "ELB", Template: CloudwatchMetricTemplate{
		Metricname: "EstimatedProcessedBytes",
		Period:     "3600",
		Unit:       "Bytes",
		Namespace:  "AWS/ELB",
		Id:         "test1",
		Stat:       "Average",
	},
	},
	"BytesOutToSource": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "NatGateway", Template: CloudwatchMetricTemplate{
		Metricname: "BytesOutToSource",
		Period:     "3600",
		Unit:       "Bytes",
		Namespace:  "AWS/NATGateway",
		Id:         "test1",
		Stat:       "Average",
	},
	},
	"BytesOutToDestination": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "NatGateway", Template: CloudwatchMetricTemplate{
		Metricname: "BytesOutToDestination",
		Period:     "3600",
		Unit:       "Bytes",
		Namespace:  "AWS/NATGateway",
		Id:         "test1",
		Stat:       "Average",
	},
	},
	"BytesInFromSource": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "NatGateway", Template: CloudwatchMetricTemplate{
		Metricname: "BytesInFromSource",
		Period:     "3600",
		Unit:       "Bytes",
		Namespace:  "AWS/NATGateway",
		Id:         "test1",
		Stat:       "Average",
	},
	},
	"BytesInFromDestination": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "NatGateway", Template: CloudwatchMetricTemplate{
		Metricname: "BytesInFromDestination",
		Period:     "3600",
		Unit:       "Bytes",
		Namespace:  "AWS/NATGateway",
		Id:         "test1",
		Stat:       "Average",
	},
	},
	"ActiveConnectionCount": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "NatGateway", Template: CloudwatchMetricTemplate{
		Metricname: "ActiveConnectionCount",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/NATGateway",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"ConnectionEstablishedCount": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "NatGateway", Template: CloudwatchMetricTemplate{
		Metricname: "ConnectionEstablishedCount",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/NATGateway",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"CoreCount": MetricTemplate{Metrictype: "CustomMetrics", ServiceName: "EC2", Template: CustomMetricTemplate{
		Metricname: "CoreCount",
	},
	},
	"VCpuCount": MetricTemplate{Metrictype: "CustomMetrics", ServiceName: "EC2", Template: CustomMetricTemplate{
		Metricname: "VCpuCount",
	},
	},
	"Size": MetricTemplate{Metrictype: "CustomMetrics", ServiceName: "EBS", Template: CustomMetricTemplate{
		Metricname: "Size",
	},
	},
	"Size_All": MetricTemplate{Metrictype: "CustomMetrics", ServiceName: "Efs", Template: CustomMetricTemplate{
		Metricname: "Size_All",
	},
	},
	"Size_Infrequent": MetricTemplate{Metrictype: "CustomMetrics", ServiceName: "Efs", Template: CustomMetricTemplate{
		Metricname: "Size_Infrequent",
	},
	},
	"Size_Standard": MetricTemplate{Metrictype: "CustomMetrics", ServiceName: "Efs", Template: CustomMetricTemplate{
		Metricname: "Size_Standard",
	},
	},
	"DataWriteIOBytes": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "Efs", Template: CloudwatchMetricTemplate{
		Metricname: "DataWriteIOBytes",
		Period:     "3600",
		Unit:       "Bytes",
		Namespace:  "AWS/EFS",
		Id:         "test1",
		Stat:       "Average",
	},
	},
	"DataReadIOBytes": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "Efs", Template: CloudwatchMetricTemplate{
		Metricname: "DataReadIOBytes",
		Period:     "3600",
		Unit:       "Bytes",
		Namespace:  "AWS/EFS",
		Id:         "test1",
		Stat:       "Average",
	},
	},
	"SuccessfulRequestLatency": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "DynamoDB", Template: CloudwatchMetricTemplate{
		Metricname: "SuccessfulRequestLatency",
		Period:     "3600",
		Unit:       "Milliseconds",
		Namespace:  "AWS/DynamoDB",
		Id:         "test1",
		Stat:       "Average",
	},
	},
	"ReturnedItemCount": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "DynamoDB", Template: CloudwatchMetricTemplate{
		Metricname: "ReturnedItemCount",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/DynamoDB",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"ConsumedWriteCapacityUnits": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "DynamoDB", Template: CloudwatchMetricTemplate{
		Metricname: "ConsumedWriteCapacityUnits",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/DynamoDB",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"ProvisionedWriteCapacityUnits": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "DynamoDB", Template: CloudwatchMetricTemplate{
		Metricname: "ProvisionedWriteCapacityUnits",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/DynamoDB",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"ConsumedReadCapacityUnits": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "DynamoDB", Template: CloudwatchMetricTemplate{
		Metricname: "ConsumedReadCapacityUnits",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/DynamoDB",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
	"ProvisionedReadCapacityUnits": MetricTemplate{Metrictype: "CloudWatchMetrics", ServiceName: "DynamoDB", Template: CloudwatchMetricTemplate{
		Metricname: "ProvisionedReadCapacityUnits",
		Period:     "3600",
		Unit:       "Count",
		Namespace:  "AWS/DynamoDB",
		Id:         "test1",
		Stat:       "Sum",
	},
	},
}
