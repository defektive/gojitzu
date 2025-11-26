package cmd

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type JiraSearchResponse struct {
	StartAt    int         `json:"startAt"`
	MaxResults int         `json:"maxResults"`
	Total      int         `json:"total"`
	Issues     []JiraIssue `json:"issues"`
}

type JiraIssue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Summary     string      `json:"summary"`
		Description interface{} `json:"description"`
		Parent      *struct {
			Key    string `json:"key"`
			Fields struct {
				Summary string `json:"summary"`
			} `json:"fields"`
		} `json:"parent,omitempty"`
	} `json:"fields"`
}

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "List issues",
	Run: func(cmd *cobra.Command, args []string) {

		jql, _ := cmd.Flags().GetString("jql")

		if jql == "" {
			fmt.Println("❌ JQL is required. Example:")
			fmt.Println(`gojitzu issues -j "project = OS ORDER BY updated DESC"`)
			return
		}

		base := viper.GetString("baseurl")
		username := viper.GetString("username")
		password := viper.GetString("password")

		fmt.Println("🔥 issues command called")
		fmt.Println("Running JQL:", jql)

		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}

		encodedJql := url.QueryEscape(jql)

		api := base + "/rest/api/3/search/jql?jql=" + encodedJql + "&maxResults=100&fields=summary,parent"

		req, err := http.NewRequest("GET", api, nil)
		if err != nil {
			fmt.Println("Failed to build request:", err)
			return
		}

		req.SetBasicAuth(username, password)

		client := &http.Client{}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Request failed:", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != 200 {
			fmt.Println("❌ JIRA ERROR BODY:", string(body))
			fmt.Println("Status:", resp.Status)
			return
		}

		var result JiraSearchResponse

		err = json.Unmarshal(body, &result)
		if err != nil {
			fmt.Println("JSON Parsing Error:", err)
			fmt.Println("Raw Body:", string(body))
			return
		}

		//fmt.Printf("✅ Issues returned: %d\n", len(result.Issues.([]interface{})))
		//fmt.Printf("✅ Issues returned: %d\n", len(result.Issues))
		//fmt.Printf("✅ Jira says total issues: %d\n", result.Total)

		if result.Total == 0 {
			fmt.Println("⚠️ WARNING: No issues returned for JQL:", jql)
		}

		pretty, _ := json.MarshalIndent(result.Issues, "", "  ")
		fmt.Println(string(pretty))
	},
}

func init() {
	rootCmd.AddCommand(issuesCmd)
	issuesCmd.Flags().StringP("jql", "j", "", "JQL to search")
}
