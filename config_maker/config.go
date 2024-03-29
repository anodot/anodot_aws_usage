package main

type CloudWatchMetric struct {
	Id        string `yaml:"Id"`
	Name      string `yaml:"Name"`
	Namespace string `yaml:"Namespace"`
	Period    string `yaml:"Period"`
	Unit      string `yaml:"Unit"`
	Stat      string `yaml:"Stat"`
}

type ServiceN struct {
	Name              string             `yaml:"-"`
	Tags              []string           `yaml:"DimensionsFromTags,omitempty"`
	CloudWatchMetrics []CloudWatchMetric `yaml:"CloudWatchMetrics,omitempty"`
	CustomMetrics     []string           `yaml:"CustomMetrics,omitempty"`
	CustomRegion      string             `yaml:"Region,omitempty"`
}

type Config struct {
	Region         string                         `yaml:"-"`
	AnodotUrl      string                         `yaml:"anodotUrl"`
	RegionsConfigs map[string]map[string]ServiceN `yaml:",inline"`
}
