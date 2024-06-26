package config

type CustomField struct {
	JiraField string `json:"JiraField" yaml:"jira_field"`
	Name      string `json:"Name" yaml:"name"`
}

type ConfigMap struct {
	CustomFields []CustomField `json:"CustomFields" yaml:"custom_fields"`
}
