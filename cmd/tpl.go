package cmd

import (
	"fmt"
	"github.com/andygrunwald/go-jira"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// tplCmd represents the create command
var tplCmd = &cobra.Command{
	Use:   "tpl",
	Short: "create issues based on templates",
	Long:  `Create issues using templates`,
	Run: func(cmd *cobra.Command, args []string) {

		templates, _ := cmd.Flags().GetStringSlice("templates")
		templatesPath := viper.GetString("templatepath")
		var templateTasks []Task
		for _, templateName := range templates {
			var template Template
			template.load(templatesPath, templateName)

			for _, task := range template.Tasks {
				fmt.Println(task.Title)
				templateTasks = append(templateTasks, task)
			}
		}

		if len(templateTasks) == 0 {
			fmt.Println("Nothing to do")
			return
		}

		//http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		base := viper.GetString("baseurl")
		username := viper.GetString("username")
		password := viper.GetString("password")
		projectKey := viper.GetString("project")
		epicKey, _ := cmd.Flags().GetString("epic")
		nextGen, _ := cmd.Flags().GetBool("nextgen")

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

		if !nextGen {
			log.Println("Using normal Jira project workflow")
			fieldList, _, _ := jiraClient.Field.GetList()

			var customFieldID string
			for _, v := range fieldList {
				if v.Name == "Epic Link" {
					customFieldID = v.ID
					break
				}
			}

			if len(epicKey) == 0 {
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
				jiraEpic, res, err := jiraClient.Issue.Create(&i)
				if err != nil {
					body, _ := io.ReadAll(res.Body)
					fmt.Println(string(body))
					panic(err)
				}
				epicKey = jiraEpic.Key
			}

			prefix, _ := cmd.Flags().GetString("prefix")
			var newIssues []int
			var newKeys []string
			for _, task := range templateTasks {
				title := task.Title
				if prefix != "" && task.Prefixable {
					title = fmt.Sprintf("%s %s", prefix, title)
				}

				i := jira.Issue{
					Fields: &jira.IssueFields{
						Description: task.Description,
						Type: jira.IssueType{
							Name: "Task",
						},
						Project: jira.Project{
							Key: jiraProject.Key,
						},
						Summary: title,
						Unknowns: map[string]interface{}{
							customFieldID: epicKey,
						},
						Labels: task.Labels,
					},
				}
				newIssue, resp, err := jiraClient.Issue.Create(&i)
				if err != nil {
					body, _ := io.ReadAll(resp.Body)
					fmt.Println(string(body))
					panic(err)
				}

				intID, err := strconv.Atoi(newIssue.ID)
				fmt.Printf("Created %s\n", task.Title)
				newIssues = append(newIssues, intID)
				newKeys = append(newKeys, newIssue.Key)

				if len(task.SubTasks) > 0 {
					// Create subTasks
					for _, subTask := range task.SubTasks {

						title := subTask.Title
						if prefix != "" && subTask.Prefixable {
							title = fmt.Sprintf("%s %s", prefix, title)
						}

						i := jira.Issue{
							Fields: &jira.IssueFields{
								Description: subTask.Description,
								Type: jira.IssueType{
									Name: "Sub-task",
								},
								Project: jira.Project{
									Key: jiraProject.Key,
								},
								Summary: title,
								Labels:  subTask.Labels,
								Parent: &jira.Parent{
									ID:  newIssue.ID,
									Key: newIssue.Key,
								},
							},
						}
						newSubTask, resp, err := jiraClient.Issue.Create(&i)
						if err != nil {
							body, _ := io.ReadAll(resp.Body)
							fmt.Println(string(body))
							panic(err)
						}

						//intID, err := strconv.Atoi(newSubTask.ID)
						fmt.Printf("Created (%s) %s\n", newSubTask.Key, subTask.Title)
						//newIssues = append(newIssues, intID)
						//newKeys = append(newKeys, newSubTask.Key)
					}

				}

			}

			fmt.Printf("Done %s\n", epicKey)
		} else {
			log.Println("Using NextGen Jira project workflow")

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

			prefix, _ := cmd.Flags().GetString("prefix")
			var newIssues []int
			for _, task := range templateTasks {
				title := task.Title
				if prefix != "" && task.Prefixable {
					title = fmt.Sprintf("%s %s", prefix, title)
				}

				i := jira.Issue{
					Fields: &jira.IssueFields{
						Description: task.Description,
						Type: jira.IssueType{
							Name: "Task",
						},
						Project: jira.Project{
							Key: jiraProject.Key,
						},
						Summary: title,
						Labels:  task.Labels,
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

				if len(task.SubTasks) > 0 {
					// Create subTasks
					for _, subTask := range task.SubTasks {

						title := subTask.Title
						if prefix != "" && subTask.Prefixable {
							title = fmt.Sprintf("%s %s", prefix, title)
						}

						i := jira.Issue{
							Fields: &jira.IssueFields{
								Description: subTask.Description,
								Type: jira.IssueType{
									Name: "Sub-task",
								},
								Project: jira.Project{
									Key: jiraProject.Key,
								},
								Summary: title,
								Labels:  subTask.Labels,
								Parent: &jira.Parent{
									ID:  newIssue.ID,
									Key: newIssue.Key,
								},
							},
						}
						newSubTask, resp, err := jiraClient.Issue.Create(&i)
						if err != nil {
							body, _ := io.ReadAll(resp.Body)
							fmt.Println(string(body))
							panic(err)
						}

						//intID, err := strconv.Atoi(newSubTask.ID)
						fmt.Printf("Created (%s) %s\n", newSubTask.Key, subTask.Title)
						//newIssues = append(newSubTask, intID)
					}

				}
			}

			//add issues to epic
			//for some reason, jira wouldn't let me set the epic link when creating issues. so this is what i am doing instead
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
		}
	},
}

func init() {

	RootCmd.AddCommand(tplCmd)

	tplCmd.PersistentFlags().BoolP("nextgen", "n", false, "specify next gen projects")

	tplCmd.Flags().StringSliceP("templates", "t", []string{}, "templates to use")
	tplCmd.RegisterFlagCompletionFunc("templates", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		templatesPath := viper.GetString("templatepath")
		var templates []string
		filepath.WalkDir(templatesPath, func(path string, d fs.DirEntry, err error) error {
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if !d.IsDir() && (ext == ".yaml" || ext == ".yml") {
				templatePath, _ := filepath.Rel(templatesPath, path)
				templates = append(templates, templatePath)
			}
			return nil
		})
		return templates, cobra.ShellCompDirectiveDefault
	})

	tplCmd.Flags().StringP("duedate", "d", "", "due date")
	tplCmd.Flags().StringP("desc", "D", "", "Description")
	tplCmd.Flags().StringP("epic", "e", "", "epic key to add issues to existing epic")
	tplCmd.Flags().StringP("title", "T", "", "Title for the new epic")
	tplCmd.Flags().String("prefix", "", "prefix for tasks that are prefixable")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
