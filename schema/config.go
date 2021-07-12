package main

type CloudwatchMetric struct {
	Id        string `yaml:"Id"`
	Name      string `yaml:"Name"`
	Namespace string `yaml:"Namespace"`
	Period    int64  `yaml:"Period"`
	Unit      string `yaml:"Unit"`
	Stat      string `yaml:"Stat"`
}

type Service struct {
	CloudwatchMetrics []CloudwatchMetric `yaml:"CloudWatchMetrics"`
	CustomMetrics     []string           `yaml:"CustomMetrics"`
}

type ConfigForSchema struct {
	Regions map[string]map[string]Service
}

func (c *ConfigForSchema) UnmarshalYAML(unmarshal func(interface{}) error) error {
	regions := make(map[string]map[string]Service)
	var config map[string]map[string]Service

	if err := unmarshal(&config); err != nil {
		return err
	}
	for region, services := range config {
		if region == "anodotUrl" || region == "token" || region == "accountName" {
			continue
		}
		regions[region] = make(map[string]Service)

		for name, service := range services {
			regions[region][name] = service
		}
	}
	c.Regions = regions
	return nil
}
