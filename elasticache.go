package main

import (
	"strconv"

	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/elasticache"
)

type CacheCluster struct {
	CacheClusterId, Engine, CacheClusterStatus, NumCacheNodes, ReplicationGroupId, Region string
}

func GetCacheClusters(session *session.Session) ([]CacheCluster, error) {
	svc := elasticache.New(session)
	input := &elasticache.DescribeCacheClustersInput{}
	result, err := svc.DescribeCacheClusters(input)
	region := session.Config.Region
	clusters := make([]CacheCluster, 0)
	if err != nil {
		return clusters, err
	}
	for _, cluster := range result.CacheClusters {
		nodenum := strconv.FormatInt(*cluster.NumCacheNodes, 10)
		clusters = append(clusters, CacheCluster{*cluster.CacheClusterId,
			*cluster.Engine,
			*cluster.CacheClusterStatus,
			nodenum, *cluster.ReplicationGroupId,
			*region,
		})
	}
	return clusters, nil
}

func GetElasticacheMetricProperties(c CacheCluster) map[string]string {
	return map[string]string{
		"cache_cluster_id":     c.CacheClusterId,
		"engine":               c.Engine,
		"cache_cluster_status": c.CacheClusterStatus,
		"num_cache_nodes":      c.NumCacheNodes,
		"replication_group_id": c.ReplicationGroupId,
		"region":               c.Region,
		"anodot-collector":     "aws",
	}
}

func GetElasticacheCloudwatchMetrics(resource *MonitoredResource, clusters []CacheCluster) ([]MetricToFetch, error) {
	metrics := make([]MetricToFetch, 0)
	for _, mstat := range resource.Metrics {
		for _, cluster := range clusters {
			m := MetricToFetch{}
			m.Dimensions = []Dimension{
				Dimension{
					Name:  "CacheClusterId",
					Value: cluster.CacheClusterId,
				},
			}
			m.Resource = cluster
			mstatCopy := mstat
			mstatCopy.Id = "ecache" + strconv.Itoa(len(metrics))
			m.MStat = mstatCopy
			metrics = append(metrics, m)
		}
	}
	return metrics, nil
}

func GetElasticacheMetrics(ses *session.Session, cloudwatchSvc *cloudwatch.CloudWatch, resource *MonitoredResource) ([]metricsAnodot.Anodot20Metric, error) {
	anodotMetrics := make([]metricsAnodot.Anodot20Metric, 0)

	cloudWatchFetcher := CloudWatchFetcher{
		cloudwatchSvc: cloudwatchSvc,
	}
	clusters, err := GetCacheClusters(ses)
	if err != nil {
		return anodotMetrics, err
	}
	mfetch, err := GetElasticacheCloudwatchMetrics(resource, clusters)
	if err != nil {
		return anodotMetrics, err
	}
	metricdatainput := NewGetMetricDataInput(mfetch)
	metricdataresults, err := cloudWatchFetcher.FetchMetrics(metricdatainput)
	for _, m := range mfetch {
		for _, mr := range metricdataresults {
			if *mr.Id == m.MStat.Id {
				ecache := m.Resource.(CacheCluster)
				anodot_ecache_metrics := GetAnodotMetric(m.MStat.Name, mr.Timestamps, mr.Values, GetElasticacheMetricProperties(ecache))
				anodotMetrics = append(anodotMetrics, anodot_ecache_metrics...)
			}
		}
	}
	return anodotMetrics, nil
}
