package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"gopkg.in/yaml.v3"
)

const green = "\u001b[32m"
const red = "\u001b[31m"
const reset = "\u001b[0m"

type Params struct {
	anodotUrl string
}

var metricsButtons = []string{"Default (All metrics above)", "Done"}

//var serviceButtons = []string{"Default (All services above)", "Done"}

var metrics = map[string][]string{
	"ElastiCache": []string{"CacheNodesCount", "CPUUtilization"},
	"EC2":         []string{"CoreCount", "VCpuCount", "NetworkOut", "NetworkIn"},
	"EBS":         []string{"Size"},
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

var servicesWithTags = map[string]bool{
	"EC2":        true,
	"EBS":        true,
	"Efs":        true,
	"ELB":        true,
	"NatGateway": true,
}

var services = []string{"EC2", "EBS", "S3", "NatGateway", "ELB", "Efs", "DynamoDB", "Cloudfront", "ElastiCache", "Default (All services above)", "Done"}

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

func GetAllMetricsAllServices() []ServiceN {
	services := make([]ServiceN, 0)
	for s, m := range metrics {
		cms := make([]string, 0)
		cwms := make([]CloudWatchMetric, 0)
		for _, metric := range m {
			if metric == "Done" {
				break
			}
			allCwms, ok := cloudwatchMetrics[s]
			if !ok {
				cms = append(cms, metric)
			} else {
				if cwm, ok := allCwms[metric]; ok {
					cwms = append(cwms, cwm)
				}
			}

		}
		services = append(services,
			ServiceN{
				Name:              s,
				CloudWatchMetrics: cwms,
				CustomMetrics:     cms,
			},
		)
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

func removeDuplicatesMetrics(list []string) []string {
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

func removeDuplicatesService(list []ServiceN) []ServiceN {
	new := make([]ServiceN, 0)
	ifPresent := false
	var s ServiceN
	for i := len(list) - 1; i >= 0; i-- {
		s = list[i]
		for _, snew := range new {
			if s.Name == snew.Name {
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

func SetDimenionsTags() ([]string, error) {
	tags := make([]string, 0)
	options := []string{
		"Add tag based dimension",
		"Skip",
	}
	v := func(s string) error {
		if len(s) == 0 {
			return fmt.Errorf("Tag name can't be blank")
		}
		return nil
	}

	for {
		r, err := CustomSelect("Do you want to set which tags will be used as dimensions ?", options)
		if err != nil {
			return tags, err
		}

		if r == "Continue" || r == "Skip" {
			break
		}

		s, err := CustomPrompt(v, "What tag you want to use as a dimension ?")
		tags = append(tags, s)
		options[0] = "Add one more tag based dimension"
		options[1] = "Continue"
	}

	return tags, nil
}

func ChooseMetrics(service string) ([]string, error) {
	metrics_ := make([]string, 0)

	metricmsg := "What metrics do you need"
	for {
		metric, err := CustomSelect(metricmsg, append(metrics[service], metricsButtons...))
		if err != nil {
			return make([]string, 0), err
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
			metrics_ = make([]string, 0)
			for _, m := range metrics[service] {
				metrics_ = append(metrics_, m)
			}
			fmt.Printf("You chose next metrics for service: %s", service)
			fmt.Println(ListToString(metrics[service]))

			return metrics_, nil
		}

		metrics_ = append(metrics_, metric)
		fmt.Printf("You chose next metrics for service: %s", service)
		fmt.Println(ListToString(metrics_))
	}
	return removeDuplicatesMetrics(metrics_), nil
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

func ChoseService(region string) ([]ServiceN, error) {
	chosenservices := make([]ServiceN, 0)
	servicenames := make([]string, 0)
	selectmsg := "What services do you need"

	//services = append(services, serviceButtons...)
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

		if service == "Default (All services above)" {
			chosenservices = make([]ServiceN, 0)
			tags, err := SetDimenionsTags()
			if err != nil {
				return make([]ServiceN, 0), err
			}

			for _, srv := range GetAllMetricsAllServices() {
				if _, ok := servicesWithTags[srv.Name]; ok {
					srv.Tags = tags
				}
				chosenservices = append(chosenservices, srv)
			}
			services = removeCloudfront(services)
			return chosenservices, nil
		}

		metrics_, err := ChooseMetrics(service)
		if err != nil {
			return make([]ServiceN, 0), err
		}

		cwms := make([]CloudWatchMetric, 0)
		cms := make([]string, 0)
		for _, m := range metrics_ {
			allCwms, ok := cloudwatchMetrics[service]
			if !ok {
				cms = metrics_
				break
			}

			if cwm, ok := allCwms[m]; ok {
				cwms = append(cwms, cwm)
			} else {
				cms = append(cms, m)
			}
		}

		if service == "Cloudfront" {
			services = removeCloudfront(services)
		}

		tags, err := SetDimenionsTags()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
		}

		chosenservices = append(chosenservices, ServiceN{
			Tags:              tags,
			Name:              service,
			CustomMetrics:     cms,
			CloudWatchMetrics: cwms,
		},
		)
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

	params, err := ChoseParams()

	if err != nil {
		fmt.Printf("Something went wrong %v", err)
		os.Exit(3)
	}
	c := Config{
		AnodotUrl:      params.anodotUrl,
		RegionsConfigs: make(map[string]map[string]ServiceN),
	}
	for {

		region, err := ChoseRegion()
		if err != nil {
			fmt.Printf("Error occured %v", err)
			break
		}

		c.RegionsConfigs[region] = make(map[string]ServiceN)

		fmt.Println("Hello please choose services you need to monitor: ")
		chosenservices, err := ChoseService(region)
		if err != nil {
			break
		}

		for _, service := range chosenservices {
			c.RegionsConfigs[region][service.Name] = service
		}

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

	b, err := yaml.Marshal(c)
	if err != nil {
		panic(err)
	}

	if err := WriteConfig(b); err != nil {
		fmt.Println(err)
		os.Exit(127)
	}
	fmt.Println(string(b))
}
