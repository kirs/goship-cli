package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v1"
)

type DeployOutputEntry struct {
	Project     string
	Environment string
	StdoutLine  string
}

type DeployConfig struct {
	Host      string
	Project   string
	RepoOwner string `yaml:"repo_owner"`
	RepoName  string `yaml:"repo_name"`
	User      string
}

const (
	CONFIG_NAME = ".goship.yml"
)

type GoshipProjectStatus struct {
	Name         string
	RepoOwner    string
	RepoName     string
	Environments []GoshipProjectEnvironment
}

type GoshipProjectEnvironment struct {
	Name               string
	Deploy             string
	RepoPath           string
	LatestGitHubCommit string
	IsDeployable       bool
	Hosts              []struct {
		URI          string
		LatestCommit string
	}
}

func startDeployRequest(finished chan int, deployEnv string, config *DeployConfig) {
	commits_url := fmt.Sprintf("http://%s/commits/%s", config.Host, config.Project)
	commits_request, err := http.Get(commits_url)
	if err != nil {
		log.Fatal(err)
	}

	var status GoshipProjectStatus
	var project_env *GoshipProjectEnvironment

	commits_response, _ := ioutil.ReadAll(commits_request.Body)
	err = json.Unmarshal(commits_response, &status)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range status.Environments {
		if v.Name == deployEnv {
			project_env = &v
		}
	}

	if !project_env.IsDeployable {
		log.Fatalf("%s is not deployable", deployEnv)
	}

	v := url.Values{}
	v.Set("project", config.Project)
	v.Add("repo_owner", status.RepoOwner)
	v.Add("repo_name", status.RepoName)
	v.Add("from_revision", project_env.Hosts[0].LatestCommit)
	v.Add("to_revision", project_env.LatestGitHubCommit)
	v.Add("environment", deployEnv)
	v.Add("user", config.User)

	_, err = http.PostForm(fmt.Sprintf("http://%s/deploy_handler", config.Host), v)

	if err != nil {
		log.Fatal(err)
	}

	finished <- 1
}

func deploy(deployEnv string) {
	finished := make(chan int)

	if len(deployEnv) == 0 {
		log.Fatal("empty environment")
	}

	config := DeployConfig{}

	configData, err := ioutil.ReadFile(CONFIG_NAME)
	if err != nil {
		log.Fatal("failed to read %s: %s", CONFIG_NAME, err)
	}

	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Printf("Deploying project `%s` to `%s`", config.Project, deployEnv)

	var handshakeDialer = &websocket.Dialer{
		Subprotocols:    []string{"chat"},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	origin_url := fmt.Sprintf("http://%s/", config.Host)
	ws_url := fmt.Sprintf("ws://%s/web_push", config.Host)

	ws, resp, err := handshakeDialer.Dial(ws_url, http.Header{"Origin": {origin_url}})
	if err != nil {
		log.Printf("Dial: %v", err)
	} else {
		defer ws.Close()
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	log.Printf("Connected: %s\n", resp.Status)

	// launch deploy
	go startDeployRequest(finished, deployEnv, &config)

	go func(ch chan int) {
		<-ch
		os.Exit(0)
	}(finished)

	var result DeployOutputEntry

	for {
		_, r, err := ws.NextReader()

		if err != nil {
			log.Fatal(err)
			break
		}
		rbuf, err := ioutil.ReadAll(r)

		json.Unmarshal(rbuf, &result)

		log.Printf("%s\n", result.StdoutLine)
	}
}
