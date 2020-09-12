package main

import (
	"strconv"
	"time"

	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/elasticache"
)

type CacheCluster struct {
	CacheClusterId, Engine, CacheClusterStatus, NumCacheNodes, ReplicationGroupId, Region, CacheNodeType string
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
		repId := ""
		nodenum := strconv.FormatInt(*cluster.NumCacheNodes, 10)
		if *cluster.Engine != "memcached" {
			repId = *cluster.ReplicationGroupId
		}

		clusters = append(clusters, CacheCluster{*cluster.CacheClusterId,
			*cluster.Engine,
			*cluster.CacheClusterStatus,
			nodenum, repId,
			*region, *cluster.CacheNodeType,
		})
	}
	return clusters, nil
}

func GetElasticacheMetricProperties(c CacheCluster) map[string]string {
	return map[string]string{
		"service":              "elasticache",
		"cache_cluster_id":     c.CacheClusterId,
		"engine":               c.Engine,
		"cache_cluster_status": c.CacheClusterStatus,
		//"num_cache_nodes":      c.NumCacheNodes,
		"region":           c.Region,
		"anodot-collector": "aws",
		"cache_node_type":  c.CacheNodeType,
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
	nodegroups, err := GetNodeGroups(ses)
	if err != nil {
		return anodotMetrics, err
	}

	if len(resource.CustomMetrics) > 0 {
		for _, cm := range resource.CustomMetrics {
			if cm == "CacheNodesCount" {
				anodotMetrics = getCacheNodesCount(clusters, nodegroups)
			}
		}
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

type NodeGroup struct {
	ReplicationGroupId, NodeGroupId string
}

func GetNodeGroups(session *session.Session) ([]NodeGroup, error) {
	nodegroups := make([]NodeGroup, 0)
	svc := elasticache.New(session)
	input := &elasticache.DescribeReplicationGroupsInput{}
	result, err := svc.DescribeReplicationGroups(input)
	if err != nil {
		return nodegroups, err
	}
	for _, rg := range result.ReplicationGroups {
		for _, ng := range rg.NodeGroups {
			nodegroups = append(nodegroups, NodeGroup{NodeGroupId: *ng.NodeGroupId,
				ReplicationGroupId: *rg.ReplicationGroupId,
			})
		}
	}
	return nodegroups, nil
}

func getCacheNodesCount(cacheclusters []CacheCluster, nodegroups []NodeGroup) []metricsAnodot.Anodot20Metric {
	metrics := make([]metricsAnodot.Anodot20Metric, 0)
	for _, cluster := range cacheclusters {
		if cluster.Engine == "memcached" {
			props := GetElasticacheMetricProperties(cluster)
			props["what"] = "cache_nodes_count"
			nodenum, _ := strconv.Atoi(cluster.NumCacheNodes)
			metric := metricsAnodot.Anodot20Metric{
				Properties: props,
				Value:      float64(nodenum),
				Timestamp: metricsAnodot.AnodotTimestamp{
					time.Now(),
				},
			}
			metrics = append(metrics, metric)
			continue
		}

		for _, ng := range nodegroups {
			if ng.ReplicationGroupId == cluster.ReplicationGroupId {
				props := GetElasticacheMetricProperties(cluster)
				props["node_group_id"] = ng.NodeGroupId
				props["what"] = "cache_nodes_count"
				nodenum, _ := strconv.Atoi(cluster.NumCacheNodes)
				metric := metricsAnodot.Anodot20Metric{
					Properties: props,
					Value:      float64(nodenum),
					Timestamp: metricsAnodot.AnodotTimestamp{
						time.Now(),
					},
				}
				metrics = append(metrics, metric)
			}
		}
	}
	return metrics
}
