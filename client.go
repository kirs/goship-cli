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

func startDeployRequest(finished chan int, deployEnv string, config *DeployConfig) {
	v := url.Values{}
	v.Set("project", config.Project)
	v.Add("repo_owner", config.RepoOwner)
	v.Add("repo_name", config.RepoName)
	v.Add("from_revision", "9922f9fd0c751e6071d50858a09c1fa9fb410bd0")
	v.Add("to_revision", "0269450e29e3690dbe984963dfdb991edd872fba")
	v.Add("environment", deployEnv)
	v.Add("user", config.User)

	_, err := http.PostForm(fmt.Sprintf("http://%s/deploy_handler", config.Host), v)

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
