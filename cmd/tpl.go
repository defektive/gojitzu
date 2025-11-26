package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Base tpl command template
var tplCmd = &cobra.Command{
	Use:   "tpl",
	Short: "create issues based on templates",
	Long:  `Create issues using templates`,
	Run:   runTplCommand,
}

func NewTplCommand() *cobra.Command {
	localTpl := *tplCmd

	flags := localTpl.Flags()

	flags.BoolP("nextgen", "n", false, "specify next gen projects")
	flags.StringSliceP("templates", "t", []string{}, "templates to use")
	flags.StringP("duedate", "d", "", "due date")
	flags.StringP("desc", "D", "", "Description")
	flags.StringP("epic", "e", "", "epic key to add issues to existing epic")
	flags.StringP("title", "T", "", "Title for the new epic")
	flags.String("prefix", "", "prefix for tasks that are prefixable")
	flags.StringP("project", "p", "", "Project override")

	localTpl.RegisterFlagCompletionFunc("templates", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		templatesPath := viper.GetString("templatepath")
		templatesPath, _ = homedir.Expand(templatesPath)

		var templates []string

		filepath.WalkDir(templatesPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if !d.IsDir() && (ext == ".yaml" || ext == ".yml") {
				templatePath, _ := filepath.Rel(templatesPath, path)
				templates = append(templates, templatePath)
			}
			return nil
		})

		return templates, cobra.ShellCompDirectiveDefault
	})

	return &localTpl
}

func runTplCommand(cmd *cobra.Command, args []string) {

	fmt.Println("=== GUI EXECUTION CONTEXT ===")
	fmt.Println("Working Dir:", func() string { d, _ := os.Getwd(); return d }())
	fmt.Println("Config Path:", viper.ConfigFileUsed())
	fmt.Println("Template Path:", viper.GetString("templatepath"))
	fmt.Println("Project:", viper.GetString("project"))
	fmt.Println("Username:", viper.GetString("username"))
	fmt.Println("🚀 tpl command EXECUTED")
	fmt.Println("==============================")

	templates, _ := cmd.Flags().GetStringSlice("templates")
	templatesPath := viper.GetString("templatepath")
	templatesPath, _ = homedir.Expand(templatesPath)

	var templateTasks []Task
	for _, templateName := range templates {
		var template Template
		template.load(templatesPath, templateName)

		for _, task := range template.Tasks {
			templateTasks = append(templateTasks, task)
		}
	}

	if len(templateTasks) == 0 {
		fmt.Println("Nothing to do")
		return
	}

	base := viper.GetString("baseurl")
	username := viper.GetString("username")
	password := viper.GetString("password")

	projectKey, _ := cmd.Flags().GetString("project")
	if projectKey == "" {
		projectKey = viper.GetString("project")
	}

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
		runClassicJiraFlow(cmd, jiraClient, jiraProject, epicKey, templateTasks)
	} else {
		runNextgenJiraFlow(cmd, jiraClient, jiraProject, epicKey, templateTasks)
	}
}

func runClassicJiraFlow(cmd *cobra.Command, jiraClient *jira.Client, jiraProject *jira.Project, epicKey string, templateTasks []Task) {

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

		const dateFmt = "2006-01-02"
		dueDateTime, _ := time.Parse(dateFmt, due)

		i := jira.Issue{
			Fields: &jira.IssueFields{
				Description: description,
				Type:        jira.IssueType{Name: "Epic"},
				Project:     jira.Project{Key: jiraProject.Key},
				Summary:     title,
				Duedate:     jira.Date(dueDateTime),
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

	for _, task := range templateTasks {

		title := task.Title
		if prefix != "" && task.Prefixable {
			title = fmt.Sprintf("%s %s", prefix, title)
		}

		i := jira.Issue{
			Fields: &jira.IssueFields{
				Description: task.Description,
				Type:        jira.IssueType{Name: "Task"},
				Project:     jira.Project{Key: jiraProject.Key},
				Summary:     title,
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

		fmt.Printf("Created (%s) %s\n", newIssue.Key, task.Title)
	}
}

func runNextgenJiraFlow(cmd *cobra.Command, jiraClient *jira.Client, jiraProject *jira.Project, epicKey string, templateTasks []Task) {

	log.Println("Using NextGen Jira project workflow")

	var jiraEpic *jira.Issue

	if len(epicKey) > 0 {
		jiraEpic, _, _ = jiraClient.Issue.Get(epicKey, nil)
	} else {
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("desc")
		due, _ := cmd.Flags().GetString("duedate")

		const dateFmt = "2006-01-02"
		dueDateTime, _ := time.Parse(dateFmt, due)

		i := jira.Issue{
			Fields: &jira.IssueFields{
				Description: description,
				Type:        jira.IssueType{Name: "Epic"},
				Project:     jira.Project{Key: jiraProject.Key},
				Summary:     title,
				Duedate:     jira.Date(dueDateTime),
			},
		}

		createdEpic, _, err := jiraClient.Issue.Create(&i)
		if err != nil {
			panic(err)
		}

		jiraEpic = createdEpic
	}

	// ✅ Now jiraEpic is ALWAYS set before use
	if jiraEpic == nil {
		panic("jiraEpic is nil — cannot continue NextGen workflow")
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
				Type:        jira.IssueType{Name: "Task"},
				Project:     jira.Project{Key: jiraProject.Key},
				Summary:     title,
				Labels:      task.Labels,
			},
		}

		newIssue, resp, err := jiraClient.Issue.Create(&i)
		if err != nil {
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Println(string(body))
			panic(err)
		}

		intID, _ := strconv.Atoi(newIssue.ID)
		newIssues = append(newIssues, intID)

		fmt.Printf("Created (%s) %s\n", newIssue.Key, task.Title)
	}

	epicPath := fmt.Sprintf(
		"/rest/internal/simplified/1.0/projects/%s/issues/%s/children",
		jiraProject.ID,
		jiraEpic.ID,
	)

	epicIssues := map[string][]int{
		"issueIds": newIssues,
	}

	req, err := jiraClient.NewRequest("POST", epicPath, epicIssues)
	resp, err := jiraClient.Do(req, nil)
	if err != nil {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(body))
		panic(err)
	}

	fmt.Printf("Done %s\n", jiraEpic.Key)
}
