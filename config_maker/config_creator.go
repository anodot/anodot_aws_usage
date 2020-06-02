package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
)

const green = "\u001b[32m"
const red = "\u001b[31m"
const reset = "\u001b[0m"

var metricsButtons = []string{"Default (All metrics above)", "Done"}

var metrics = map[string][]string{
	"EC2": []string{"CoreCount", "VCpuCount", "NetworkOut", "NetworkIn"},
	"EBS": []string{"Size", "Done"},
	"S3": []string{"BucketSizeBytes", "NumberOfObjects", "AllRequests", "GetRequests",
		"PutRequests", "DeleteRequests", "HeadRequests",
		"SelectRequests", "ListRequests"},
	"Cloudfront": []string{"BytesDownloaded", "Requests", "TotalErrorRate"},
	"ELB":        []string{"RequestCount", "EstimatedProcessedBytes"},
	"NatGateway": []string{"BytesOutToSource", "BytesOutToDestination", "BytesInFromSource",
		"BytesInFromDestination", "ActiveConnectionCount",
		"ConnectionEstablishedCount"},
	"Efs": []string{"Size_All", "Size_Infrequent", "Size_Standard", "DataWriteIOBytes", "DataReadIOBytes"},
	"DynamoDB": []string{"SuccessfulRequestLatency", "ReturnedItemCount", "ConsumedWriteCapacityUnits",
		"ProvisionedWriteCapacityUnits", "ConsumedReadCapacityUnits", "ProvisionedReadCapacityUnits"},
}

var services = []string{"Default (All services above)", "EC2", "EBS", "S3", "Cloudfront", "NatGateway", "ELB", "Efs", "DynamoDB", "Done"}

var regions = []string{
	"eu-north-1",
	"ap-south-1",
	"eu-west-3",
	"eu-west-2",
	"eu-west-1",
	"ap-northeast-2",
	"ap-northeast-1",
	"sa-east-1",
	"ca-central-1",
	"ap-southeast-1",
	"ap-southeast-2",
	"eu-central-1",
	"us-east-1",
	"us-east-2",
	"us-west-1",
	"us-west-2",
}

var selectedRegions []string

type Metric struct {
	name string
}

type Service struct {
	name    string
	metrics []Metric
}

type RegionConfig struct {
	region   string
	services []Service
}

type Params struct {
	token       string
	anodotUrl   string
	accountName string
}

func GetAllMetricsAllServices() []Service {
	services := make([]Service, 0)
	for s, m := range metrics {
		metrics_ := make([]Metric, 0)
		for _, metric := range m {
			if metric == "Done" {
				break
			}
			metrics_ = append(metrics_, Metric{name: metric})
		}
		services = append(services, Service{name: s, metrics: metrics_})
	}
	return services
}

func ValidateUrl(input string) error {
	if !strings.HasPrefix(input, "http") {
		return fmt.Errorf("Please provide correct url")
	}
	return nil
}

func ValidateToken(input string) error {
	if len(input) < 2 {
		return fmt.Errorf("Token len shoud be > 1")
	}
	return nil
}

func ValidateAccountName(input string) error {
	if len(input) < 2 {
		return fmt.Errorf("Account name len shoud be > 1")
	}
	return nil
}

func ValidateRegion(input string) error {
	valid := false
	for _, r := range regions {
		if input == r {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("Please provide correct region name")
	}

	for _, sr := range selectedRegions {
		if sr == input {
			return fmt.Errorf("Region %s already selected please choose another one", input)
		}
	}
	return nil
}

func CustomSelect(label string, items []string) (string, error) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   " > {{ . | cyan }} ",
		Inactive: "  {{ . | magenta }} ",
		Selected: "", //"{{ . | red | cyan }}",
	}
	prompt := promptui.Select{
		Label:     label,
		Items:     items,
		Templates: templates,
	}

	_, result, err := prompt.Run()

	if err != nil {
		return "", err
	}
	return result, nil
}

func CustomPrompt(validate func(string) error, name string) (string, error) {
	prompt := promptui.Prompt{
		Label:    name,
		Validate: validate,
	}
	result, err := prompt.Run()

	if err != nil {
		return "", err
	}
	return result, nil
}

func removeDuplicates(list []string) []string {
	new := make([]string, 0)
	ifPresent := false
	for _, s := range list {
		for _, snew := range new {
			if s == snew {
				ifPresent = true
			}
		}
		if !ifPresent {
			new = append(new, s)
		} else {
			ifPresent = false
		}
	}
	return new
}

func removeDuplicatesMetrics(list []Metric) []Metric {
	new := make([]Metric, 0)
	ifPresent := false
	for _, s := range list {
		for _, snew := range new {
			if s.name == snew.name {
				ifPresent = true
			}
		}
		if !ifPresent {
			new = append(new, s)
		} else {
			ifPresent = false
		}
	}
	return new
}

func removeDuplicatesService(list []Service) []Service {
	new := make([]Service, 0)
	ifPresent := false
	for _, s := range list {
		for _, snew := range new {
			if s.name == snew.name {
				ifPresent = true
			}
		}
		if !ifPresent {
			new = append(new, s)
		} else {
			ifPresent = false
		}
	}
	return new
}

func ListToString(list []string) string {
	list = removeDuplicates(list)
	var r string
	for _, s := range list {
		r = r + "\n\t" + green + s + reset
	}
	return r
}

func ChooseMetrics(service string) ([]Metric, error) {
	metrics_ := make([]Metric, 0)
	metricsname := make([]string, 0)
	metricmsg := "What metrics do you need"
	for {
		metric, err := CustomSelect(metricmsg, append(metrics[service], metricsButtons...))
		if err != nil {
			return make([]Metric, 0), err
		}
		metricmsg = "What metrics do you need"

		if metric == "Done" {
			if len(metrics_) > 0 {
				break
			} else {
				metricmsg = "You didn't choose any metrics. What metrics do you need"
				continue
			}
		}

		if metric == "Default (All metrics above)" {

			metrics_ = make([]Metric, 0)
			for _, m := range metrics[service] {
				metrics_ = append(metrics_, Metric{name: m})
			}
			fmt.Printf("You chose next metrics for service: %s", service)
			fmt.Println(ListToString(metrics[service]))

			return metrics_, nil
		}

		metricsname = append(metricsname, metric)
		metrics_ = append(metrics_, Metric{name: metric})
		fmt.Printf("You chose next metrics for service: %s", service)
		fmt.Println(ListToString(metricsname))
	}
	return removeDuplicatesMetrics(metrics_), nil
}

func removeCloudfront(list []string) []string {
	services := make([]string, 0)

	for _, s := range list {
		if s != "Cloudfront" {
			services = append(services, s)
		}
	}
	delete(metrics, "Cloudfront")
	return services
}

func removeUsedRegion(regions []string, region string) []string {
	regions_ := make([]string, 0)
	for _, r := range regions {
		if r != region {
			regions_ = append(regions_, r)
		}
	}
	return regions_
}

func ChoseService(region string) ([]Service, error) {
	chosenservices := make([]Service, 0)
	servicenames := make([]string, 0)
	selectmsg := "What services do you need"
	for {
		service, err := CustomSelect(selectmsg, services)
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
		}
		selectmsg = "What services do you need"
		if service == "Done" {
			if len(chosenservices) > 0 {
				break
			} else {
				selectmsg = "You didn't choose any services. What services do you need"
				continue
			}

		}

		// If Cloudfront has been chosen remove it from service list because it global service
		// and can be chosen only once
		if service == "Default (All services above)" {
			chosenservices = GetAllMetricsAllServices()
			services = removeCloudfront(services)
			return chosenservices, nil
		}

		metrics_, err := ChooseMetrics(service)
		if err != nil {
			return make([]Service, 0), err
		}

		if service == "Cloudfront" {
			services = removeCloudfront(services)
		}

		chosenservices = append(chosenservices, Service{name: service, metrics: metrics_})
		servicenames = append(servicenames, service)

		fmt.Printf("You chose next services for region: %s", region)
		fmt.Println(ListToString(servicenames))
	}
	return removeDuplicatesService(chosenservices), nil
}

func ChoseParams() (Params, error) {
	p := Params{}
	url, err := CustomPrompt(ValidateUrl, "Anodot URL")
	if err != nil {
		return p, err
	}
	p.anodotUrl = url

	token, err := CustomPrompt(ValidateToken, "Anodot Token")
	if err != nil {
		return p, err
	}
	p.token = token

	accountName, err := CustomPrompt(ValidateAccountName, "Account name")
	if err != nil {
		return p, err
	}

	p.accountName = accountName
	return p, nil
}

func ChoseRegion() (string, error) {
	result, err := CustomSelect("Choose your region", regions)
	if err != nil {
		return "", err
	}
	if ValidateRegion(result) != nil {
		return "", err
	}
	selectedRegions = append(selectedRegions, result)
	regions = removeUsedRegion(regions, result)

	return result, nil
}

func WriteConfig(data []byte) error {
	fo, err := os.Create("cloudwatch_metrics.yaml")
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	if _, err := fo.Write(data); err != nil {
		return err
	}
	return nil
}

func main() {
	configs := make([]RegionConfig, 0)
	params, err := ChoseParams()

	if err != nil {
		fmt.Printf("Something went wrong %v", err)
		os.Exit(3)
	}

	for {
		region, err := ChoseRegion()
		if err != nil {
			fmt.Printf("Error occured %v", err)
			break
		}
		regionconfig := RegionConfig{region: region}

		fmt.Println("Hello please choose services you need to monitor: ")
		chosenservices, err := ChoseService(region)
		if err != nil {
			break
		}
		regionconfig.services = chosenservices
		configs = append(configs, regionconfig)
		prompt := promptui.Select{
			Label: "Do you want do add another region ?",
			Items: []string{"Yes", "No"},
		}

		_, newregion, err := prompt.Run()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			break
		}
		if newregion == "No" {
			break
		}
	}

	confstr := RenderConfig(params, configs)
	fmt.Println(confstr)

	if err := WriteConfig([]byte(confstr)); err != nil {
		fmt.Println(err)
		os.Exit(127)
	}
}
