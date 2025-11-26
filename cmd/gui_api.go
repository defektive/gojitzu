package cmd

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type GUIProject struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type GUIEpic struct {
	Key   string `json:"key"`
	Title string `json:"title"`
}

type GUITemplate struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ------------------
// JIRA CLIENT
// ------------------

func getJiraClient() (*http.Client, string, string, string) {

	initConfig()

	base := strings.TrimRight(strings.TrimSpace(viper.GetString("baseurl")), "/")
	username := strings.TrimSpace(viper.GetString("username"))
	password := strings.TrimSpace(viper.GetString("password"))

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: transport}

	return client, base, username, password
}

// ------------------
// GUI API
// ------------------

func GetProjects() ([]GUIProject, string, error) {

	client, base, username, password := getJiraClient()

	req, _ := http.NewRequest("GET", base+"/rest/api/3/project/search?maxResults=100", nil)
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var parsed struct {
		Values []struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"values"`
	}

	json.Unmarshal(body, &parsed)

	out := []GUIProject{}
	for _, p := range parsed.Values {
		out = append(out, GUIProject{Key: p.Key, Name: p.Name})
	}

	return out, fmt.Sprintf("Loaded %d projects", len(out)), nil
}

func GetEpics(project string) ([]GUIEpic, string, error) {

	client, base, username, password := getJiraClient()

	jql := "project = " + project + " AND issuetype = Epic ORDER BY updated DESC"
	encoded := url.QueryEscape(jql)

	req, _ := http.NewRequest("GET", base+"/rest/api/3/search/jql?jql="+encoded+"&fields=summary", nil)
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var parsed struct {
		Issues []struct {
			Key    string `json:"key"`
			Fields struct {
				Summary string `json:"summary"`
			} `json:"fields"`
		} `json:"issues"`
	}

	json.Unmarshal(body, &parsed)

	out := []GUIEpic{}
	for _, e := range parsed.Issues {
		out = append(out, GUIEpic{Key: e.Key, Title: e.Fields.Summary})
	}

	return out, fmt.Sprintf("Loaded %d epics", len(out)), nil
}

func GetDefaultProject() string {
	return viper.GetString("project")
}

// ------------------
// Templates
// ------------------

func GetTemplates() ([]GUITemplate, error) {

	path := viper.GetString("templatepath")

	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}

	path = filepath.Clean(path)

	var templates []GUITemplate

	filepath.Walk(path, func(fullPath string, info os.FileInfo, err error) error {

		if err != nil || info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))
		if ext == ".yaml" || ext == ".yml" {

			rel, _ := filepath.Rel(path, fullPath)

			templates = append(templates, GUITemplate{
				Name: rel,
				Path: fullPath,
			})
		}

		return nil
	})

	return templates, nil
}

// ------------------
// GUI RUNNER
// ------------------

// ✅ FIX: No more template stacking

func resetTplState(root *cobra.Command) {

	for _, c := range root.Commands() {
		if c.Name() == "tpl" {

			flag := c.Flag("templates")
			if flag != nil {
				flag.Value.Set("")
			}

			c.Flags().Set("templates", "")
			return
		}
	}
}

func RunForGUI(args []string) (string, error) {

	viper.Reset()
	initConfig()

	root := NewRootCommand()

	resetTplState(root)

	buf := new(strings.Builder)

	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err := root.Execute()

	return buf.String(), err
}
