package cmd

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/andygrunwald/go-jira"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

// issuesCmd represents the issues command
var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "List issues",
	Long:  `List issues within the project.`,
	Run: func(cmd *cobra.Command, args []string) {
		jql, _ := cmd.Flags().GetString("jql")

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

		last := ""

		opt := &jira.SearchOptionsV2{
			MaxResults:    1000, // Max results can go up to 1000
			NextPageToken: last,
			Fields:        []string{"*all"},
		}

		issues, resp, err := jiraClient.Issue.SearchV2JQL(jql, opt)
		if err != nil {
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
			panic(err)
		}

		jsonBytes, err := json.MarshalIndent(issues, "", "  ")
		if err != nil {
			panic(err)
		}

		fmt.Println(string(jsonBytes))
	},
}

func GetAllIssues(jiraClient *jira.Client, jql string) []jira.Issue {

	opt := &jira.SearchOptionsV2{
		MaxResults:    1000, // Max results can go up to 1000
		NextPageToken: "",
		Fields:        []string{"*all"},
	}
	retIssues := []jira.Issue{}
	for {

		issues, resp, err := jiraClient.Issue.SearchV2JQL(jql, opt)
		if err != nil {
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
			panic(err)
		}

		retIssues = append(retIssues, issues...)

		if resp.IsLast {
			break
		}

		opt.NextPageToken = resp.NextPageToken
	}

	return retIssues
}

func init() {
	RootCmd.AddCommand(issuesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// issuesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	issuesCmd.Flags().StringP("jql", "j", "", "JQL to search")
}
