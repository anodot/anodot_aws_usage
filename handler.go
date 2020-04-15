package main

import (
	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"log"
	"sync"
)

type SyncMetricList struct {
	mux     sync.Mutex
	metrics []metricsAnodot.Anodot20Metric
}

func (ml *SyncMetricList) Append(newmetrics []metricsAnodot.Anodot20Metric) {
	ml.mux.Lock()
	defer ml.mux.Unlock()
	ml.metrics = append(ml.metrics, newmetrics...)
}

type ErrorList struct {
	mux    sync.Mutex
	errors []error
}

func (el *ErrorList) Append(err error) {
	el.mux.Lock()
	defer el.mux.Unlock()
	el.errors = append(el.errors, err)
}

type MetricFunction func(*session.Session, *cloudwatch.CloudWatch, *MonitoredResource) ([]metricsAnodot.Anodot20Metric, error)

func Handle(resources []*MonitoredResource, wg *sync.WaitGroup, session *session.Session, cloudwatchsvc *cloudwatch.CloudWatch, ml *SyncMetricList, el *ErrorList) {

	for _, resource := range resources {
		wg.Add(1)
		rs := resource

		go func(wg *sync.WaitGroup, rs *MonitoredResource) {
			defer wg.Done()

			metrics, err := rs.MFunction(session, cloudwatchsvc, rs)
			if err != nil {
				log.Printf("ERROR encoutered during processing %s metrics ", rs.Name)
				el.Append(err)
				return
			}
			log.Printf("Got %d metrics for %s", len(metrics), rs.Name)
			ml.Append(metrics)
		}(wg, rs)
	}
}
