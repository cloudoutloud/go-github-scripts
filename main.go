package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type Spec struct {
	appID          *int64
	installationID *int64
	privateKeyFile string
	dirName        string
}

func NewGraphClient(ctx context.Context, s Spec) *githubv4.Client {
	return githubv4.NewClient(newGitHubAuth(s, &ctx))
}

// Uses Github app or PAT
func newGitHubAuth(s Spec, ctx *context.Context) *http.Client {
	if s.privateKeyFile != "" {
		log.Println("Pulling credentials for GitHub App")
		if s.appID != nil && s.installationID != nil {
			itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, *s.appID, *s.installationID, s.privateKeyFile)
			if err != nil {
				log.Println("Error creating GitHub App installation client:", err)
				return &http.Client{}
			}
			return &http.Client{Transport: itr}
		} else {
			log.Println("App ID or Installation ID is nil")
		}
	} else {
		log.Println("GITHUB_APP_PRIVATE_KEY is empty or not set")
	}

	// Interactive run by a user PAT
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		log.Println("Pulling credentials for GitHub Personal Access Token")
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		return oauth2.NewClient(*ctx, ts)
	}

	return &http.Client{}
}

// Loop through and clone each repository in new directory
func cloneRepositories(s Spec, client *githubv4.Client) error {
	ctx := context.Background()

	var query struct {
		Viewer struct {
			Repositories struct {
				Nodes []struct {
					Name   string
					SSHURL string `graphql:"sshUrl"`
				}
			} `graphql:"repositories(first: 100)"`
		}
	}

	err := client.Query(ctx, &query, nil)
	if err != nil {
		return fmt.Errorf("unable to list repositories: %v", err)
	}

	err = os.MkdirAll(s.dirName, 0755)
	if err != nil {
		return fmt.Errorf("unable to create directory: %v", err)
	}

	err = os.Chdir(s.dirName)
	if err != nil {
		return fmt.Errorf("unable to change directory: %v", err)
	}

	for _, repo := range query.Viewer.Repositories.Nodes {

		fmt.Printf("Cloning Repo: %s\n", repo.Name)
		cmd := exec.Command("git", "clone", repo.SSHURL)
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error cloning repository %s: %v\n", repo.Name, err)
		}
	}

	return nil
}

func main() {
	ctx := context.Background()

	appID := int64(1234)          // GH app ID
	installationID := int64(1234) // GH app instalion ID
	privateKeyFile := ""          // pem key file
	dirName := ""                 // Directory name you want to create

	s := Spec{
		appID:          &appID,
		installationID: &installationID,
		privateKeyFile: privateKeyFile,
		dirName:        dirName,
	}

	client := NewGraphClient(ctx, s)

	err := cloneRepositories(s, client)
	if err != nil {
		log.Fatalf("Error cloning repositories: %v", err)
	}
}
