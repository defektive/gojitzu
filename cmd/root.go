/*
Copyright © 2021 defektive

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"crypto/tls"
	"fmt"
	"github.com/andygrunwald/go-jira"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

var cfgFile string
var labelsFlag []string

type Task struct {
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Labels      []string `yaml:"labels"`
}
type Template struct {
	Version string `yaml:"version"`
	Tasks []Task   `yaml:"tasks"`
}

func (tpl *Template) load(templatePath string) *Template {
	yamlFile, err := ioutil.ReadFile(templatePath)
	if err != nil {
		log.Printf("yamlFile.Get err #%v ", err)
	}

	err = yaml.Unmarshal(yamlFile, tpl)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return tpl
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gojitzu",
	Short: "create ",
	Long:  `create test`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {

		templates, _ := cmd.Flags().GetStringSlice("templates")
		var templateTasks []Task
		for _, templateName := range templates {
			templatePath := path.Join(viper.GetString("templatepath"), templateName)
			var template Template
			template.load(templatePath)

			for _, task := range template.Tasks {
				fmt.Println(task.Title)
				templateTasks = append(templateTasks, task)
			}
		}


		if len(templateTasks) == 0 {
			fmt.Println("Nothing to do")
			return
		}

		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		base := viper.GetString("baseurl")
		username := viper.GetString("username")
		password:= viper.GetString("password")
		projectKey :=  viper.GetString("project")
		epicKey, _ := cmd.Flags().GetString("epic")

		tp := jira.BasicAuthTransport{
			Username: username,
			Password: password,
		}

		jiraClient, err := jira.NewClient(tp.Client(), base)
		if err != nil {
			panic(err)
		}

		me, _, _ := jiraClient.User.GetSelf()
		jiraProject, resp, err := jiraClient.Project.Get(projectKey)
		if err != nil {
			//body, _ := ioutil.ReadAll(resp.Body)
			//fmt.Println(string(body))
			panic(err)
		}

		fmt.Println(jiraProject.Key)
		var jiraEpic *jira.Issue
		if len(epicKey) > 0 {
			jiraEpic, _, _ = jiraClient.Issue.Get(epicKey, nil)
		} else {
			title, _ := cmd.Flags().GetString("title")
			description, _ := cmd.Flags().GetString("desc")
			due, _ := cmd.Flags().GetString("duedate")
			fmt.Println(title, description)
			const dateFmt = "2006-01-02"
			dueDateTime, _ := time.Parse(dateFmt, due)
			i := jira.Issue{
				Fields: &jira.IssueFields{
					Description: description,
					Type: jira.IssueType{
						Name: "Epic",
					},
					Project: jira.Project{
						Key: jiraProject.Key,
					},
					Summary: title,
					Duedate: jira.Date(dueDateTime),
				},
			}
			jiraEpic, _, err = jiraClient.Issue.Create(&i)
			if err != nil {
				panic(err)
			}
		}

		var newIssues []int
		for _, task := range templateTasks {
			i := jira.Issue{
				Fields: &jira.IssueFields{
					Description: task.Description,
					Type: jira.IssueType{
						Name: "Task",
					},
					Project: jira.Project{
						Key: jiraProject.Key,
					},
					Summary: task.Title,
					Labels: task.Labels,
					Reporter:    me,
				},
			}
			newIssue, resp, err := jiraClient.Issue.Create(&i)
			if err != nil {
				body, _ := ioutil.ReadAll(resp.Body)
				fmt.Println(string(body))
				panic(err)
			}

			intID, err := strconv.Atoi(newIssue.ID)
			fmt.Printf("Created %s\n", task.Title)
			newIssues = append(newIssues, intID)
		}

		// add issues to epic
		// for some reason, jira wouldn't let me set the epic link when creating issues. so this is what i am doing instead
		epicPath := fmt.Sprintf("/rest/internal/simplified/1.0/projects/%s/issues/%s/children", jiraProject.ID, jiraEpic.ID)
		epicIssues := make(map[string][]int)
		epicIssues["issueIds"] = newIssues
		req, err := jiraClient.NewRequest("POST", epicPath, epicIssues)

		resp, err = jiraClient.Do(req, nil)
		if err != nil {
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Println(string(body))
			panic(err)
		}
		fmt.Printf("Done %s\n", jiraEpic.Key)
	},
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gojitzu.yaml)")
	rootCmd.PersistentFlags().StringP("baseurl", "b", "", "base url for jira")
	rootCmd.PersistentFlags().StringP("project", "p", "", "project key")
	rootCmd.PersistentFlags().StringP("templatepath", "g", path.Join(home, ".gojitzu-templates"), "$HOME/.gojitzu-templates")
	rootCmd.PersistentFlags().StringP("username", "U", "", "username to use")
	rootCmd.PersistentFlags().StringP("password", "P", "", "password/token")

	rootCmd.Flags().StringSliceP("templates", "t", []string{}, "templates to use")
	rootCmd.Flags().StringP("duedate", "d", "", "due date")
	rootCmd.Flags().StringP("desc", "D", "", "Description")
	rootCmd.Flags().StringP("epic", "e", "", "epic key to add issues to existing epic")
	rootCmd.Flags().StringP("title", "T", "", "Title for the new epic")
	//rootCmd.Flags().StringSliceVarP(&labelsFlag, "labels", "l", []string{},"template file")

	viper.BindPFlag("baseurl", rootCmd.PersistentFlags().Lookup("baseurl"))
	viper.BindPFlag("project", rootCmd.PersistentFlags().Lookup("project"))
	viper.BindPFlag("username", rootCmd.PersistentFlags().Lookup("username"))
	viper.BindPFlag("password", rootCmd.PersistentFlags().Lookup("password"))
	viper.BindPFlag("templatepath", rootCmd.PersistentFlags().Lookup("templatepath"))
}

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
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
