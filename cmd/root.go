package cmd

import (
	"fmt"
	"github.com/defektive/gojitzu/pkg/config"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
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

func (tpl *Template) load(baseDir string, templatePath string, includedSoFar ...map[string]bool) *Template {
	fullPath := filepath.Join(baseDir, templatePath)
	yamlFile, err := ioutil.ReadFile(fullPath)
	if err != nil {
		log.Printf("yamlFile.Get err #%v ", err)
	}

	err = yaml.Unmarshal(yamlFile, tpl)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	included := make(map[string]bool)
	for _, inc := range includedSoFar {
		for key := range inc {
			if _, found := included[key]; found {
				continue
			}
			included[key] = true
		}
	}

	if includedSoFar == nil {
		included[fullPath] = true
	}

	for _, includePath := range tpl.Includes {
		fullIncludePath := filepath.Join(baseDir, includePath)
		if _, found := included[fullIncludePath]; found {
			continue
		}
		included[fullIncludePath] = true

		var includedTpl Template
		includedTpl.load(baseDir, includePath, included)
		tpl.Tasks = append(tpl.Tasks, includedTpl.Tasks...)
	}

	return tpl
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gojitzu",
	Short: "Create tickets",
	Long:  `Create test`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//Run: func(cmd *cobra.Command, args []string) {
	//
	//},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//rootCmd.Flags().StringSliceVarP(&labelsFlag, "labels", "l", []string{},"template file")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gojitzu.yaml)")
	rootCmd.PersistentFlags().StringP("baseurl", "b", "", "base url for jira")
	rootCmd.PersistentFlags().StringP("project", "p", "", "project key")
	rootCmd.PersistentFlags().StringP("templatepath", "g", path.Join(home, ".gojitzu-templates"), "$HOME/.gojitzu-templates")
	rootCmd.PersistentFlags().StringP("username", "U", "", "username to use")
	rootCmd.PersistentFlags().StringP("password", "P", "", "password/token")

	viper.BindPFlag("baseurl", rootCmd.PersistentFlags().Lookup("baseurl"))
	viper.BindPFlag("project", rootCmd.PersistentFlags().Lookup("project"))
	viper.BindPFlag("username", rootCmd.PersistentFlags().Lookup("username"))
	viper.BindPFlag("password", rootCmd.PersistentFlags().Lookup("password"))
	viper.BindPFlag("templatepath", rootCmd.PersistentFlags().Lookup("templatepath"))
}

var Config = config.ConfigMap{}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".gojitzu" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".gojitzu")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Error config file:", viper.ConfigFileUsed(), err)
	}

	configByte, err := os.ReadFile(viper.ConfigFileUsed())
	if err != nil {
		fmt.Println("error parseing config", err)
	}

	err = yaml.Unmarshal(configByte, &Config)
	if err != nil {
		fmt.Println("error unmarshalling yaml", err)
	}
}
