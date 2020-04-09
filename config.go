package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

const conf = `
S3:
  CloudWatchMetrics:
  - Name: BucketSizeBytes
    Id: test1
    Namespace: AWS/S3
    Period: 3600
    Unit: Bytes
    Stat:  Average
  - Name: NumberOfObjects
    Id: test1
    Namespace: AWS/S3
    Period: 3600
    Unit: Count
    Stat:  Average

EBS:
  CustomMetrics:
    - Size
EC2:
  CustomMetrics:
    - CoreCount

ELB:
  CloudWatchMetrics:   
  - Name: RequestCount
    Id: test1
    Namespace: AWS/ELB
    Period: 600
    Unit: Count
    Stat:  Average
  - Name: EstimatedProcessedBytes
    Id: test2
    Namespace: AWS/ELB
    Period: 600
    Unit: Bytes
    Stat:  Average
`

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
	region:= os.Getenv("region")

	if anodotUrl == "" || token == "" || region == "" {
		return Config{}, errors.New("Need to define env vars anodotUrl, token, region")
	}

	c := Config{
		AnodotUrl:   anodotUrl,
		AnodotToken: token,
		Region: region,
	}
	/*fileData, err := ioutil.ReadFile("cloudwatch_metrics.yaml")
	if err != nil {
		log.Fatalf("error: %v", err)
		return c, err
	}*/

	err := yaml.Unmarshal([]byte(conf), &c)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return c, err
	}

	return c, nil
}
