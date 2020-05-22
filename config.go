package main

import (
	"bytes"
	"fmt"
	"os"

	metricsAnodot "github.com/anodot/anodot-common/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/yaml.v2"

	"log"
)

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
	MFunction     MetricFunction
	CustomRegion  string // in case metrics should be fetched from some different region where lambda installed
}

type MetricFunction func(*session.Session, *cloudwatch.CloudWatch, *MonitoredResource) ([]metricsAnodot.Anodot20Metric, error)

func (mr *MonitoredResource) String() string {
	s := fmt.Sprintf("	Name: %s\n", mr.Name)
	s = s + "	Tags:\n"
	for _, tag := range mr.Tags {
		s = s + fmt.Sprintf("	%s: %s", tag.Name, tag.Value)
	}
	s = s + "	Metric stat:\n"
	for _, ms := range mr.Metrics {
		s = s + "	" + ms.String()
	}
	s = s + "	Cutom metrics:\n"
	for _, cm := range mr.CustomMetrics {
		s = s + "	" + cm + "\n"
	}
	return s
}

type Config struct {
	LambdaBucket   string
	AccountId      string
	Region         string
	AnodotUrl      string
	AnodotToken    string
	RegionsConfigs map[string]RegionConfig
}

func (c *Config) String() string {
	s := fmt.Sprintf("Region: %s\nAnodot URL: %s\nAonodot Token: %s\nAccountId: %s\n", c.Region, c.AnodotUrl, c.AnodotToken, c.AccountId)
	for region, rconf := range c.RegionsConfigs {
		s = s + fmt.Sprintf("Region %s has following config: \n", region)
		s = s + rconf.String()
	}
	return s
}

type RegionConfig struct {
	Resources []*MonitoredResource
}

func (c *RegionConfig) String() string {
	s := "Monitored resources:\n"
	for _, r := range c.Resources {
		s = s + "\n"
		s = s + r.String()
	}
	return s
}

func GetMetricFunction(rname string) (MetricFunction, error) {
	switch rname {
	case "EC2":
		return GetEc2Metrics, nil
	case "EBS":
		return GetEBSMetrics, nil
	case "ELB":
		return GetELBMetrics, nil
	case "S3":
		return GetS3Metrics, nil
	case "Cloudfront":
		return GetCloudfrontMetrics, nil
	case "NatGateway":
		return GetNatGatewayMetrics, nil
	case "Efs":
		return GetEfsMetrics, nil
	default:
		return nil, fmt.Errorf("Unknown resource type: %s", rname)
	}
}

func getCustomRegion(rname string, rconfig map[interface{}]interface{}) string {
	r := ""
	if region, ok := rconfig["Region"].(string); ok {
		log.Printf("Resource %s has custom region for Cloudwatch. Metrics will fetched from %s.", rname, region)
		r = region
	}
	return r
}

func getCloudwatchMetrics(rname string, rconfig map[interface{}]interface{}) ([]MetricStat, error) {
	metrics := make([]MetricStat, 0)
	if rmetrics, ok := rconfig["CloudWatchMetrics"].([]interface{}); ok {
		for _, m := range rmetrics {
			mtf := MetricStat{}
			mmap := m.(map[interface{}]interface{})

			if id, ok := mmap["Id"]; ok {
				i, ok := id.(string)
				if !ok {
					return metrics, fmt.Errorf("Metric config should have Id")
				}
				mtf.Id = i
			} else {
				return metrics, fmt.Errorf("Metric config should have Id")
			}

			if name, ok := mmap["Name"]; ok {
				n, ok := name.(string)
				if !ok {
					return metrics, fmt.Errorf("Metric config should have Name")
				}
				mtf.Name = n
			} else {
				return metrics, fmt.Errorf("Metric config should have Name")
			}

			if namespace, ok := mmap["Namespace"]; ok {
				ns, ok := namespace.(string)
				if !ok {
					return metrics, fmt.Errorf("Metric config should have Namespace")
				}
				mtf.Namespace = ns
			} else {
				return metrics, fmt.Errorf("Metric config should have Namespace")
			}

			if unit, ok := mmap["Unit"]; ok {
				u, ok := unit.(string)
				if !ok {
					return metrics, fmt.Errorf("Metric config should have Unit")
				}
				mtf.Unit = u
			} else {
				return metrics, fmt.Errorf("Metric config should have Unit")
			}

			if stat, ok := mmap["Stat"]; ok {
				s, ok := stat.(string)
				if !ok {
					return metrics, fmt.Errorf("Metric config should have Stat")
				}
				mtf.Stat = s
			} else {
				return metrics, fmt.Errorf("Metric config should have Stat")
			}

			if period, ok := mmap["Period"]; ok {
				p, ok := period.(int)
				if !ok {
					return metrics, fmt.Errorf("Metric config should have Period")
				}
				mtf.Period = int64(p)
			} else {
				return metrics, fmt.Errorf("Metric config should have Period")
			}

			metrics = append(metrics, mtf)
		}

	} else {
		log.Printf("Config for  %s doesn't have  CLoudwatch metrics", rname)
	}
	return metrics, nil
}

func getCustomMetrics(rconfig map[interface{}]interface{}) []string {
	custommetrics := make([]string, 0)
	if custommetricsRaw, ok := rconfig["CustomMetrics"].([]interface{}); ok {
		for _, m := range custommetricsRaw {
			custommetrics = append(custommetrics, m.(string))
		}
	}
	return custommetrics
}

func getTags(rname string, rconfig map[interface{}]interface{}) ([]Tag, error) {
	tagsList := make([]Tag, 0)
	if tags, ok := rconfig["Tags"].([]interface{}); ok {
		for _, t := range tags {
			tag := Tag{}
			rtag := t.(map[interface{}]interface{})
			n, ok := rtag["Name"].(string)
			if !ok {
				return tagsList, fmt.Errorf("Tag should have Name field")
			}
			v, ok := rtag["Value"].(string)
			if !ok {
				return tagsList, fmt.Errorf("Tag should have Value field")
			}

			tag.Name = n
			tag.Value = v
			tagsList = append(tagsList, tag)
		}
	} else {
		log.Printf("Config for %s doesn't have filed Tag ", rname)
	}
	return tagsList, nil
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	regionconfigs := make(map[string]RegionConfig)
	config := map[string]interface{}{}
	if err := unmarshal(&config); err != nil {
		return err
	}

	for region, rconfig := range config {
		if region == "anodotUrl" {
			c.AnodotUrl = rconfig.(string)
			continue
		}

		if region == "token" {
			c.AnodotToken = rconfig.(string)
			continue
		}

		if region == "accountName" {
			c.AccountId = rconfig.(string)
			continue
		}

		resources := make([]*MonitoredResource, 0)
		conf := rconfig.(map[interface{}]interface{})

		for rn, rkey := range conf {
			rname := rn.(string)
			cmap := rkey.(map[interface{}]interface{})
			mr := &MonitoredResource{
				Name: rname,
			}
			mfunc, err := GetMetricFunction(rname)
			if err != nil {
				return err
			}
			mr.MFunction = mfunc

			mr.CustomMetrics = getCustomMetrics(cmap)

			tags, err := getTags(rname, cmap)
			if err != nil {
				return err
			}
			mr.Tags = tags

			metrics, err := getCloudwatchMetrics(rname, cmap)
			if err != nil {
				return err
			}
			mr.Metrics = metrics

			mr.CustomRegion = getCustomRegion(rname, cmap)

			resources = append(resources, mr)
		}

		regionconfigs[region] = RegionConfig{
			Resources: resources,
		}
		c.RegionsConfigs = regionconfigs
	}

	return nil
}

func GetConfig() (Config, error) {
	anodotUrl := os.Getenv("anodotUrl")
	token := os.Getenv("token")
	region := os.Getenv("region")
	lambda_bucket := os.Getenv("lambda_bucket")
	accountId := os.Getenv("accountName")
	c := Config{}

	if region == "" || lambda_bucket == "" {
		return Config{}, fmt.Errorf("Please provide region and lambda_bucket (lambda s3 bucket) as lambda functions env var")
	}
	c.Region = region

	/*fileData, err := ioutil.ReadFile("cloudwatch_metrics.yaml")
	if err != nil {
		log.Fatalf("error: %v", err)
		return c, err
	}*/

	fileData, err := GetConfigFromS3(lambda_bucket, region)
	if err != nil {
		fmt.Printf("Can not get config from s3 : %v\n", err)
		return c, err
	}

	err = yaml.Unmarshal([]byte(fileData), &c)
	if err != nil {
		fmt.Printf("Can not Unmarshal config : %v\n", err)
		return c, err
	}

	if anodotUrl != "" {
		c.AnodotUrl = anodotUrl
	}

	if token != "" {
		c.AnodotToken = token
	}

	if accountId != "" {
		c.AccountId = accountId
	}

	if c.AnodotToken == "" || c.AnodotUrl == "" {
		return c, fmt.Errorf("Too few arguments for lambda function. Please set token, anodotUrl with config file or with lambda env vars.")
	}

	log.Printf("Input config:")
	fmt.Print(c.String())

	return c, nil
}

func GetConfigFromS3(bucket_name, region string) ([]byte, error) {
	//session := session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
	session := session.New()
	svc := s3.New(session)

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket_name),
		Key:    aws.String("usage_lambda/cloudwatch_metrics.yaml"),
	}

	buf := new(bytes.Buffer)
	result, err := svc.GetObject(input)
	if err != nil {
		return nil, err
	}
	data := make([]byte, int(*result.ContentLength))
	buf.ReadFrom(result.Body)
	_, err = buf.Read(data)

	if err != nil {
		return nil, err
	}
	return data, nil
}
