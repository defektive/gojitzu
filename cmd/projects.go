package cmd

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ProjectSearchResponse struct {
	Total  int `json:"total"`
	Values []struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"values"`
}

func fetchProjectsManually(client *http.Client, base, user, token string) ([]byte, error) {

	//req, err := http.NewRequest("GET", base+"/rest/api/3/project/search", nil)
	req, err := http.NewRequest("GET", base+"/rest/api/3/project/search?maxResults=100", nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(user, token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List projects",
	Long:  `List all projects`,
	Run: func(cmd *cobra.Command, args []string) {

		base := strings.TrimRight(strings.TrimSpace(viper.GetString("baseurl")), "/")
		username := strings.TrimSpace(viper.GetString("username"))
		password := strings.TrimSpace(viper.GetString("password"))

		fmt.Println("✅ JIRA Base:", base)

		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			Proxy: http.ProxyFromEnvironment,
		}

		tp := jira.BasicAuthTransport{
			Username: username,
			Password: password,
		}

		authClient := tp.Client()
		authClient.Transport = transport

		// Manual Cloud-safe test first
		req, _ := http.NewRequest("GET", base+"/rest/api/3/myself", nil)
		req.SetBasicAuth(username, password)

		respCheck, err := authClient.Do(req)
		if err != nil {
			fmt.Println("AUTH CHECK REQUEST FAILED:", err)
			return
		}
		defer respCheck.Body.Close()

		if respCheck.StatusCode != 200 {
			fmt.Println("🚨 Jira authentication failed. Stopping here.")
			return
		}

		fmt.Println("✅ Jira authentication confirmed")

		raw, err := fetchProjectsManually(authClient, base, username, password)
		if err != nil {
			fmt.Println("Project search failed:", err)
			return
		}

		var response ProjectSearchResponse
		err = json.Unmarshal(raw, &response)
		if err != nil {
			fmt.Println("JSON parse error:", err)
			fmt.Println("Raw response:", string(raw))
			return
		}
		fmt.Printf("Jira reported %d total projects\n", response.Total)

		if len(response.Values) == 0 {
			fmt.Println("No projects returned by Jira Cloud")
			fmt.Println("Raw response:", string(raw))
			return
		}

		fmt.Println("✅ Projects found:")
		for _, p := range response.Values {
			fmt.Printf(" - %s : %s\n", p.Key, p.Name)
		}
	},
}

func init() {
	rootCmd.AddCommand(projectsCmd)
}
