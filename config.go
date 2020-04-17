package main

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/yaml.v2"

	//"io/ioutil"
	"log"
	"os"
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
}

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
	AccountId   string
	Region      string
	AnodotUrl   string
	AnodotToken string
	Resources   []*MonitoredResource
}

func (c *Config) String() string {
	s := fmt.Sprintf("Region: %s\nAnodot URL:%s\nAonodot Token%s\nAccountId: %s\n", c.Region, c.AnodotUrl, c.AnodotToken, c.AccountId)
	s = s + "Monitored resources:\n"
	for _, r := range c.Resources {
		s = s + "\n"
		s = s + r.String()
	}
	return s
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	config := map[string]interface{}{}
	if err := unmarshal(&config); err != nil {
		return err
	}
	resources := make([]*MonitoredResource, 0)
	for rname, rkey := range config {
		mr := &MonitoredResource{
			Name: rname,
		}

		if rname == "EC2" {
			mr.MFunction = GetEc2Metrics
		} else if rname == "EBS" {
			mr.MFunction = GetEBSMetrics
		} else if rname == "ELB" {
			mr.MFunction = GetELBMetrics
		} else if rname == "S3" {
			mr.MFunction = GetS3Metrics
		} else if rname == "Cloudfront" {
			mr.MFunction = GetCloudfrontMetrics
		}

		cmap := rkey.(map[interface{}]interface{})
		if custommetricsRaw, ok := cmap["CustomMetrics"].([]interface{}); ok {
			custommetrics := make([]string, 0)
			for _, m := range custommetricsRaw {
				custommetrics = append(custommetrics, m.(string))
			}
			mr.CustomMetrics = custommetrics
		}
		tagsList := make([]Tag, 0)
		if tags, ok := cmap["Tags"].([]interface{}); ok {
			for _, t := range tags {
				tag := Tag{}
				rtag := t.(map[interface{}]interface{})
				n, ok := rtag["Name"].(string)
				if !ok {
					return fmt.Errorf("Tag should have Name field")
				}
				v, ok := rtag["Value"].(string)
				if !ok {
					return fmt.Errorf("Tag should have Value field")
				}

				tag.Name = n
				tag.Value = v
				tagsList = append(tagsList, tag)
			}
		} else {
			log.Printf("Config for %s doesn't have filed Tag ", rname)
		}
		mr.Tags = tagsList

		if rmetrics, ok := cmap["CloudWatchMetrics"].([]interface{}); ok {
			metrics := make([]MetricStat, 0)
			for _, m := range rmetrics {
				mtf := MetricStat{}
				mmap := m.(map[interface{}]interface{})

				if id, ok := mmap["Id"]; ok {
					i, ok := id.(string)
					if !ok {
						return fmt.Errorf("Metric config should have Id")
					}
					mtf.Id = i
				} else {
					return fmt.Errorf("Metric config should have Id")
				}

				if name, ok := mmap["Name"]; ok {
					n, ok := name.(string)
					if !ok {
						return fmt.Errorf("Metric config should have Name")
					}
					mtf.Name = n
				} else {
					return fmt.Errorf("Metric config should have Name")
				}

				if namespace, ok := mmap["Namespace"]; ok {
					ns, ok := namespace.(string)
					if !ok {
						return fmt.Errorf("Metric config should have Namespace")
					}
					mtf.Namespace = ns
				} else {
					return fmt.Errorf("Metric config should have Namespace")
				}

				if unit, ok := mmap["Unit"]; ok {
					u, ok := unit.(string)
					if !ok {
						return fmt.Errorf("Metric config should have Unit")
					}
					mtf.Unit = u
				} else {
					return fmt.Errorf("Metric config should have Unit")
				}

				if stat, ok := mmap["Stat"]; ok {
					s, ok := stat.(string)
					if !ok {
						return fmt.Errorf("Metric config should have Stat")
					}
					mtf.Stat = s
				} else {
					return fmt.Errorf("Metric config should have Stat")
				}

				if period, ok := mmap["Period"]; ok {
					p, ok := period.(int)
					if !ok {
						return fmt.Errorf("Metric config should have Period")
					}
					mtf.Period = int64(p)
				} else {
					return fmt.Errorf("Metric config should have Period")
				}

				metrics = append(metrics, mtf)
			}
			mr.Metrics = metrics
		} else {
			log.Printf("Config for  %s doesn't have  CLoudwatch metrics", rname)
		}
		resources = append(resources, mr)
	}
	c.Resources = resources
	log.Printf("Input config:")
	fmt.Print(c.String())
	return nil
}

func GetConfig() (Config, error) {
	anodotUrl := os.Getenv("anodotUrl")
	token := os.Getenv("token")
	region := os.Getenv("region")
	lambda_bucket := os.Getenv("lambda_bucket")
	accountId := os.Getenv("accountId")

	if anodotUrl == "" || token == "" || region == "" || lambda_bucket == "" {
		return Config{}, errors.New("Need to define env vars anodotUrl, token, region, lambda_bucket")
	}

	c := Config{
		AccountId:   accountId,
		AnodotUrl:   anodotUrl,
		AnodotToken: token,
		Region:      region,
	}

	/*fileData, err := ioutil.ReadFile("cloudwatch_metrics.yaml")
	if err != nil {
		log.Fatalf("error: %v", err)
		return c, err
	}*/

	fileData, err := GetConfigFromS3(lambda_bucket, region)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return c, err
	}

	err = yaml.Unmarshal([]byte(fileData), &c)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return c, err
	}

	return c, nil
}

func GetConfigFromS3(bucket_name, region string) ([]byte, error) {
	session := session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
	//session := session.New()
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
