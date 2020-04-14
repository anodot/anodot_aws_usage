package main

import (
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"log"
	"time"
)

const offset time.Duration = time.Hour

type Tag struct {
	Name  string
	Value string
}

type MonitoredResource struct {
	Name          string
	Tags          []Tag
	Metrics       []MetricStat
	Ids           []string
	CustomMetrics []string // List of fields which will be used as measurement
}

type Dimension struct {
	Name  string
	Value string
}

type MetricStat struct {
	Id        string
	Name      string
	Namespace string
	Period    int64
	Unit      string
	Stat      string
	Label     string
}

type MetricToFetch struct {
	Resource   interface{}
	MStat      MetricStat
	Dimensions []Dimension
}

func NewGetMetricDataInput(mTofetch []MetricToFetch) []*cloudwatch.GetMetricDataInput {
	// General way:
	// []Dimension -> Metric -> MetricStat -> []MetricDataQuery -> GetMetricDataInput
	datainputs := make([]*cloudwatch.GetMetricDataInput, 0)
	endTime := time.Now()
	startTime := endTime.Add(-offset)

	mQueries := make([]*cloudwatch.MetricDataQuery, 0)
	di := &cloudwatch.GetMetricDataInput{}

	for index, metric := range mTofetch {

		m := metric.MStat
		dimensions := make([]*cloudwatch.Dimension, 0)

		for i := 0; i < len(metric.Dimensions); i++ {
			name := &metric.Dimensions[i].Name
			value := &metric.Dimensions[i].Value
			dimensions = append(dimensions, &cloudwatch.Dimension{
				Name:  name,
				Value: value,
			})
		}

		mStat := &cloudwatch.MetricStat{
			Metric: &cloudwatch.Metric{
				Dimensions: dimensions,
				MetricName: &m.Name,
				Namespace:  &m.Namespace,
			},
			Period: &m.Period,
			Unit:   &m.Unit,
			Stat:   &m.Stat,
		}
		dimensions = make([]*cloudwatch.Dimension, 0)
		mdatQuery := &cloudwatch.MetricDataQuery{
			Id:         &m.Id,
			MetricStat: mStat,
		}

		mQueries = append(mQueries, mdatQuery)

		if index == len(mTofetch)-1 || index%400 == 0 && index != 0 {

			di.SetMetricDataQueries(mQueries)
			di.SetEndTime(endTime)
			di.SetStartTime(startTime)
			datainputs = append(datainputs, di)
			di = &cloudwatch.GetMetricDataInput{}
			mQueries = make([]*cloudwatch.MetricDataQuery, 0)
		}
	}

	return datainputs
}

type CloudWatchFetcher struct {
	cloudwatchSvc *cloudwatch.CloudWatch
}

func (cf *CloudWatchFetcher) FetchMetrics(metricinputs []*cloudwatch.GetMetricDataInput) ([]*cloudwatch.MetricDataResult, error) {

	metricdataresults := make([]*cloudwatch.MetricDataResult, 0)
	for _, mi := range metricinputs {
		mo, err := cf.cloudwatchSvc.GetMetricData(mi)
		if err != nil {
			log.Printf("Cloud not fetch metrics from CLoudWatch : %v", err)
			return metricdataresults, err
		}
		metricdataresults = append(metricdataresults, mo.MetricDataResults...)
	}
	return metricdataresults, nil
}
