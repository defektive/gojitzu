package cmd

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/andygrunwald/go-jira"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"net/http"
)

// projectsCmd represents the projects command
var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List projects",
	Long:  `List all projects`,
	Run: func(cmd *cobra.Command, args []string) {

		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		base := viper.GetString("baseurl")
		username := viper.GetString("username")
		password := viper.GetString("password")

		tp := jira.BasicAuthTransport{
			Username: username,
			Password: password,
		}

		jiraClient, err := jira.NewClient(tp.Client(), base)
		if err != nil {
			panic(err)
		}

		jiraProjects, resp, err := jiraClient.Project.GetList()
		if err != nil {
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
			panic(err)
		}

		jsonBytes, err := json.MarshalIndent(*jiraProjects, "", "  ")
		if err != nil {
			panic(err)
		}

		fmt.Println(string(jsonBytes))
	},
}

func init() {
	RootCmd.AddCommand(projectsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// projectsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// projectsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
