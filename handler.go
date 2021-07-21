package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/anodot/anodot-common/pkg/metrics3"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

type SyncMetricList struct {
	mux     sync.Mutex
	metrics []metrics3.AnodotMetrics30
}

func (ml *SyncMetricList) Append(newmetrics []metrics3.AnodotMetrics30) {
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

func Handle(resources map[string]*MonitoredResource, wg *sync.WaitGroup, sess *session.Session, cloudwatchsvc *cloudwatch.CloudWatch, ml *SyncMetricList, el *ErrorList) {

	for resourceName, resource := range resources {
		wg.Add(1)
		rs := resource
		session_copy := sess
		rname := resourceName

		go func(wg *sync.WaitGroup, ss *session.Session, rs *MonitoredResource, rname string) {
			defer wg.Done()
			mfunc := GetMetricsFunction(rname)

			metrics, err := mfunc(ss, cloudwatchsvc, rs)
			if err != nil {
				log.Printf("ERROR encoutered during processing %s metrics ", rname)
				el.Append(err)
				return
			}
			sId, ok := schemaIds[rname]
			if !ok {
				el.Append(fmt.Errorf("failed to get schema ID for %s", rname))
				return
			}
			metricsUpdated := make([]metrics3.AnodotMetrics30, 0)
			for _, m := range metrics {
				m.SchemaId = sId
				metricsUpdated = append(metricsUpdated, m)
			}

			log.Printf("Got %d metrics for %s", len(metrics), rname)
			ml.Append(metricsUpdated)
		}(wg, session_copy, rs, rname)
	}
}
