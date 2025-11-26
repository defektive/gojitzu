package cmd

import (
	"fmt"
	"github.com/andygrunwald/go-jira"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
)

// addIssueCmd represents the addIssue command
var addIssueCmd = &cobra.Command{
	Use:   "addIssue",
	Short: "Add a new issue",
	Long:  `Add a new issue.`,
	Run: func(cmd *cobra.Command, args []string) {
		summary, _ := cmd.Flags().GetString("summary")
		description, _ := cmd.Flags().GetString("description")
		labels, _ := cmd.Flags().GetStringArray("label")
		projectKey := viper.GetString("project")

		//http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

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

		jiraProject, resp, err := jiraClient.Project.Get(projectKey)
		if err != nil {
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
			panic(err)
		}

		i := jira.Issue{
			Fields: &jira.IssueFields{
				Description: description,
				Type: jira.IssueType{
					Name: "Task",
				},
				Project: jira.Project{
					Key: jiraProject.Key,
				},
				Summary: summary,
				Labels:  labels,
			},
		}
		newIssue, resp, err := jiraClient.Issue.Create(&i)
		if err != nil {
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
			panic(err)
		}

		fmt.Printf("Created %s %s\n", newIssue.ID, newIssue.Key)
	},
}

func init() {
	issuesCmd.AddCommand(addIssueCmd)
	addIssueCmd.Flags().StringP("summary", "s", "", "Summary of the issue")
	addIssueCmd.Flags().StringP("type", "t", "task", "Type of issue. EG: task, sub-task, epic, bug")
	addIssueCmd.Flags().StringSliceP("label", "l", []string{}, "Labels of the issue")
	addIssueCmd.Flags().StringP("description", "d", "", "Description of the issue")
}
