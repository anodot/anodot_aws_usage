package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/anodot/anodot-common/pkg/metrics3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	"gopkg.in/yaml.v2"
)

type CustomMetricDefinition struct {
	Name       string
	TargetType string
	Alias      string
}

type Tag struct {
	Name  string
	Value string
}

type MonitoredResource struct {
	Tags          []Tag
	DimensionTags []string     `yaml:"DimensionsFromTags,omitempty"`
	Metrics       []MetricStat `yaml:"CloudWatchMetrics"`
	CustomMetrics []string     `yaml:"CustomMetrics"`
	CustomRegion  string       `yaml:"Region,omitempty"`
}

type MetricFunction func(*session.Session, *cloudwatch.CloudWatch, *MonitoredResource) ([]metrics3.AnodotMetrics30, error)

type Config struct {
	AccessKey      string `yaml:"accessKey"`
	AccountId      string `yaml:"accountName"`
	Region         string
	AnodotUrl      string                                   `yaml:"anodotUrl"`
	AnodotToken    string                                   `yaml:"token"`
	RegionsConfigs map[string]map[string]*MonitoredResource `yaml:",inline"`
}

func GetMetricsFunction(servicName string) MetricFunction {
	switch servicName {
	case "EC2":
		return GetEc2Metrics30
	case "EBS":
		return GetEBSMetrics30
	case "ELB":
		return GetELBMetrics30
	case "S3":
		return GetS3Metrics30
	case "Cloudfront":
		return GetCloudfrontMetrics30
	case "NatGateway":
		return GetNatGatewayMetrics30
	case "Efs":
		return GetEfsMetrics30
	case "DynamoDB":
		return GetDynamoDbMetrics30
	case "Kinesis":
		return GetKinesisMetrics30
	case "ElastiCache":
		return GetElasticacheMetrics30
	}
	return nil
}

func GetSecretValue(secretId, region string) (*string, error) {
	//session := session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
	session := session.New()
	i := secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretId),
	}
	svc := secretsmanager.New(session)
	o, err := svc.GetSecretValue(&i)
	if err != nil {
		return nil, err
	}
	return o.SecretString, nil
}

func GetConfig() (Config, error) {
	anodotUrl := os.Getenv("anodotUrl")

	token := os.Getenv("token")
	region := os.Getenv("region")
	lambda_bucket := os.Getenv("lambda_bucket")

	accountId := os.Getenv("accountId")

	c := Config{}

	if region == "" || lambda_bucket == "" || accountId == "" {
		return Config{}, fmt.Errorf("Please provide region, accountId and lambda_bucket (lambda s3 bucket) as lambda functions env var")
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

	accessKey, err := GetSecretValue(accountId+"_anodot_access_key", c.Region)
	if err != nil {
		log.Fatalf("failed to fetch accessKey from  secrets manager: %v", err)
	}

	if accessKey == nil {
		log.Fatalf("failed to fetch accessKey from  secrets manager: accesKey can't be blank")
	}

	c.AccessKey = *accessKey

	dataToken, err := GetSecretValue(accountId+"_anodot_data_token", c.Region)
	if err != nil {
		log.Fatalf("failed to fetch anodot data token from  secrets manager: %v", err)
	}
	if accessKey == nil {
		log.Fatalf("failed to fetch anodot data token from  secrets manager: data token can't be blank")
	}
	c.AnodotToken = *dataToken

	if c.AnodotToken == "" || c.AnodotUrl == "" {
		return c, fmt.Errorf("Too few arguments for lambda function. Please set token, anodotUrl with config file or with lambda env vars.")
	}

	//log.Printf("Input config:")
	//fmt.Print(c.String())

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
