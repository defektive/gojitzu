package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/defektive/gojitzu/pkg/config"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var cfgFile string
var labelsFlag []string

type Task struct {
	Title       string    `yaml:"title"`
	Description string    `yaml:"description"`
	Labels      []string  `yaml:"labels"`
	Prefixable  bool      `yaml:"prefixable"`
	SubTasks    []SubTask `yaml:"subtasks"`
}

type SubTask struct {
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Labels      []string `yaml:"labels"`
	Prefixable  bool     `yaml:"prefixable"`
}

type Template struct {
	Version  string   `yaml:"version"`
	Includes []string `yaml:"includes,omitempty"`
	Tasks    []Task   `yaml:"tasks"`
}

var Config = config.ConfigMap{}

func (tpl *Template) load(baseDir string, templatePath string, includedSoFar ...map[string]bool) *Template {

	expandedBase, _ := homedir.Expand(baseDir)
	baseDir = expandedBase

	expandedTemplate, _ := homedir.Expand(templatePath)
	templatePath = expandedTemplate

	fullPath := filepath.Join(baseDir, templatePath)

	yamlFile, err := ioutil.ReadFile(fullPath)
	if err != nil {
		log.Printf("yamlFile.Get err #%v ", err)
		return tpl
	}

	err = yaml.Unmarshal(yamlFile, tpl)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	included := make(map[string]bool)

	for _, inc := range includedSoFar {
		for key := range inc {
			included[key] = true
		}
	}

	if includedSoFar == nil {
		included[fullPath] = true
	}

	for _, includePath := range tpl.Includes {
		fullIncludePath := filepath.Join(baseDir, includePath)

		if included[fullIncludePath] {
			continue
		}

		included[fullIncludePath] = true

		var includedTpl Template
		includedTpl.load(baseDir, includePath, included)
		tpl.Tasks = append(tpl.Tasks, includedTpl.Tasks...)
	}

	return tpl
}

// Standard CLI Entry
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Global root for CLI mode
var rootCmd = &cobra.Command{
	Use:   "gojitzu",
	Short: "Create tickets",
	Long:  `Create test`,
}

func init() {
	cobra.OnInitialize(initConfig)

	home, _ := homedir.Dir()

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.gojitzu.yaml)")
	rootCmd.PersistentFlags().StringP("baseurl", "b", "", "base url for jira")
	rootCmd.PersistentFlags().StringP("project", "p", "", "project key")
	rootCmd.PersistentFlags().StringP("templatepath", "g", path.Join(home, ".gojitzu-templates"), "template path")
	rootCmd.PersistentFlags().StringP("username", "U", "", "username to use")
	rootCmd.PersistentFlags().StringP("password", "P", "", "password/token")

	viper.BindPFlag("baseurl", rootCmd.PersistentFlags().Lookup("baseurl"))
	viper.BindPFlag("project", rootCmd.PersistentFlags().Lookup("project"))
	viper.BindPFlag("username", rootCmd.PersistentFlags().Lookup("username"))
	viper.BindPFlag("password", rootCmd.PersistentFlags().Lookup("password"))
	viper.BindPFlag("templatepath", rootCmd.PersistentFlags().Lookup("templatepath"))
}

// ✅ Fresh command instance used by GUI
func NewRootCommand() *cobra.Command {

	root := &cobra.Command{
		Use:   "gojitzu",
		Short: "Create tickets",
		Long:  `Create test`,
	}

	root.AddCommand(issuesCmd)
	root.AddCommand(projectsCmd)
	root.AddCommand(NewTplCommand())

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
	root.PersistentFlags().StringP("baseurl", "b", "", "base url for jira")
	root.PersistentFlags().StringP("project", "p", "", "project key")
	root.PersistentFlags().StringP("templatepath", "g", "", "template path")
	root.PersistentFlags().StringP("username", "U", "", "username")
	root.PersistentFlags().StringP("password", "P", "", "token")

	viper.BindPFlag("baseurl", root.PersistentFlags().Lookup("baseurl"))
	viper.BindPFlag("project", root.PersistentFlags().Lookup("project"))
	viper.BindPFlag("username", root.PersistentFlags().Lookup("username"))
	viper.BindPFlag("password", root.PersistentFlags().Lookup("password"))
	viper.BindPFlag("templatepath", root.PersistentFlags().Lookup("templatepath"))

	return root
}

// ✅ Config loader
func initConfig() {

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := homedir.Dir()
		viper.AddConfigPath(home)
		viper.SetConfigName(".gojitzu")
	}

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("Error config file:", err)
		return
	}

	fmt.Println("📂 Loaded config:", viper.ConfigFileUsed())
	fmt.Println("🔧 baseurl loaded:", viper.GetString("baseurl"))

	raw, err := os.ReadFile(viper.ConfigFileUsed())
	if err != nil {
		return
	}

	yaml.Unmarshal(raw, &Config)
}
