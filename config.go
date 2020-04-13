package main

import (
	"bytes"
	"encoding/json"
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

type Config struct {
	Region      string
	AnodotUrl   string
	AnodotToken string
	Resources   []MonitoredResource
}

func (ic *Config) GetConfigJson() string {
	out, _ := json.MarshalIndent(ic, " ", " ")
	return string(out)
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	config := map[string]interface{}{}
	if err := unmarshal(&config); err != nil {
		return err
	}
	resources := make([]MonitoredResource, 0)
	for rname, rkey := range config {
		mr := MonitoredResource{
			Name: rname,
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
	log.Printf("Input config:%v", c.GetConfigJson())
	return nil
}

func GetConfig() (Config, error) {
	anodotUrl := os.Getenv("anodotUrl")
	token := os.Getenv("token")
	region := os.Getenv("region")
	lambda_bucket := os.Getenv("lambda_bucket")

	if anodotUrl == "" || token == "" || region == "" || lambda_bucket == "" {
		return Config{}, errors.New("Need to define env vars anodotUrl, token, region, lambda_bucket")
	}

	c := Config{
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
