package main

import (
	"bytes"
	"fmt"
	"html/template"
)

type Template interface {
	Render() (string, error)
}

type MetricTemplate struct {
	Metrictype  string
	Template    Template
	ServiceName string
}

type CustomMetricTemplate struct {
	Metricname string
}

func (ct CustomMetricTemplate) Render() (string, error) {
	return "    - " + ct.Metricname, nil
}

type CloudwatchMetricTemplate struct {
	Metricname string
	Period     string
	Unit       string
	Namespace  string
	Id         string
	Stat       string
}

func (ct CloudwatchMetricTemplate) Render() (string, error) {
	var b bytes.Buffer
	//foo := bufio.NewWriter(&b)

	var cloudwatchmetric = "    - Name: {{ .Metricname}}\n        Id: {{ .Id}}\n        Namespace: {{.Namespace}}\n        Period: {{.Period}}\n        Unit: {{.Unit}}\n        Stat: {{.Stat}}"
	t, err := template.New("cloudwatchmetric").Parse(cloudwatchmetric)
	if err != nil {
		return "", err
	}

	err = t.Execute(&b, ct)
	return string(b.Bytes()), nil
}

func RenderConfig(configs []RegionConfig) string {
	renderedRegions := make([]string, 0)
	config := ""
	for _, c := range configs {
		renderedServices := make([]string, 0)
		for _, s := range c.services {
			custom_metrics := make([]string, 0)
			cloudwatch_metrics := make([]string, 0)

			for _, m := range s.metrics {
				template := metric_templates[m.name]
				if template.Metrictype == "CloudWatchMetrics" {
					ms, err := template.Template.Render()
					if err != nil {
						fmt.Println(err)
					}
					cloudwatch_metrics = append(cloudwatch_metrics, ms)
				}

				if template.Metrictype == "CustomMetrics" {
					ms, err := template.Template.Render()
					if err != nil {
						fmt.Println(err)
					}
					custom_metrics = append(custom_metrics, ms)
				}
			}
			servicestr, err := RenderServices(s.name, custom_metrics, cloudwatch_metrics)
			if err != nil {
				fmt.Println(err)
			}
			renderedServices = append(renderedServices, servicestr)
		}
		regionstr, err := RenderRegion(renderedServices, c.region)
		if err != nil {
			fmt.Println(err)
		}
		renderedRegions = append(renderedRegions, regionstr)
	}

	for _, r := range renderedRegions {
		config = config + "\n" + r
	}
	return config
}

func RenderRegion(services []string, region string) (string, error) {
	var b bytes.Buffer
	data := struct {
		Services []string
		Region   string
	}{
		services,
		region,
	}
	regiontemplate := "{{- .Region -}}:{{ range .Services}} \n  {{.}} \n  {{end}}"
	t, err := template.New("regiontemplate").Parse(regiontemplate)
	if err != nil {
		return "", err
	}
	err = t.Execute(&b, data)
	return string(b.Bytes()), nil
}

func RenderServices(name string, custom_metrics []string, cloudwatch_metrics []string) (string, error) {
	var b bytes.Buffer
	data := struct {
		Name              string
		CustomMetrics     []string
		CloudwatchMetrics []string
	}{
		name,
		custom_metrics,
		cloudwatch_metrics,
	}
	servicetemplate := "{{ .Name}}:"
	if len(data.CloudwatchMetrics) > 0 {
		servicetemplate = servicetemplate + "\n    CloudwatchMetrics:{{ range .CloudwatchMetrics }}\n  {{.}}{{- end -}}"
	}

	if len(data.CustomMetrics) > 0 {
		servicetemplate = servicetemplate + "\n    CustomMetrics:{{ range .CustomMetrics}}\n  {{.}}{{- end -}}"
	}

	t, err := template.New("servicetemplate").Parse(servicetemplate)
	if err != nil {
		return "", err
	}
	err = t.Execute(&b, data)
	return string(b.Bytes()), nil
}
