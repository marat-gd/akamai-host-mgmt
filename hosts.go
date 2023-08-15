package main

//for WAP, App&API Protector
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/comcast/go-edgegrid/edgegrid"
	"github.com/kataras/golog"
)

type hostnameList struct {
	Items []hostname `json:"hostnameList"`
	Mode  string     `json:"mode"`
}

type hostname struct {
	Item string `json:"hostname"`
}

type hostRecord struct {
	ActiveInProduction bool `json:"activeInProduction"`
	ActiveInStaging    bool `json:"activeInStaging"`
	hostname
}

type selectableHostnames struct {
	AvailableSet []hostRecord `json:"availableSet"`
	//SelectedSet []hostRecord
}

type cloneConfig struct {
	CreateFromVersion string `json:"createFromVersion"`
	RuleUpdate        bool   `json:"ruleUpdate"`
}

type securityConfig struct {
	Items []config `json:"configurations"`
}
type config struct {
	ID                int `json:"id"`
	ProductionVersion int `json:"productionVersion"`
}
type clonedConfig struct {
	ConfigID int `json:"configId"`
	Version  int `json:"version"`
}

var AkamaiHost string = "https://" + os.Getenv("AKAMAI_EDGEGRID_HOST")
var configID string
var version string
var policyID string
var mode = "append"

func main() {

	//get configuration ID, version and policyID for WAP product
	//function calls...
	configJson := GetConfig(AkamaiHost)
	config := new(securityConfig)
	err := json.Unmarshal(configJson, config)
	if err != nil {
		golog.Fatal("Error! ", err)
		return
	}
	golog.Info("Security config data: ")
	fmt.Printf("%+v", config)

	configID = strconv.Itoa(config.Items[0].ID)
	version = strconv.Itoa(config.Items[0].ProductionVersion)
	//overwrite for testing purposes

	configID = "47313"
	version = "10"
	//Check for new hostnames

	//result := ListSelected(AkamaiHost, configID, version, policyID)

	configHostnames := ListSelectableOnConfig(AkamaiHost, configID, version)

	policy := new(selectableHostnames)

	err = json.Unmarshal(configHostnames, policy)
	if err != nil {
		golog.Fatal("Error!", err)
		return
	}
	//grab available hostnames
	h := hostnameList{}
	for _, v := range policy.AvailableSet {
		h.Items = append(h.Items, v.hostname)
	}
	//If no new hostnames available - exit
	if len(h.Items) < 1 {
		golog.Warn("Available hostnames set is empty: No new hostnames present.\n Exiting...")
		return
	}
	//If there are, then create new hostname list to append

	h.Mode = mode
	newSet, err := json.Marshal(h)
	if err != nil {
		golog.Fatal("Error!", err)
		return
	}
	golog.Info(string(newSet))

	//for testing purposes use fixed array
	newSet = []byte(`{"hostnameList":[{"hostname":"marat.akamaized.net"}], "mode":"append"}`)

	//clone latest config

	cloneData := cloneConfig{CreateFromVersion: version, RuleUpdate: false}
	cloneJson, err := json.Marshal(cloneData)
	if err != nil {
		golog.Fatal("Error!", err)
		return
	}
	clone := CloneConfig(AkamaiHost, configID, version, cloneJson)
	v := new(clonedConfig)
	err = json.Unmarshal(clone, v)
	if err != nil {
		golog.Fatal("Error!", err)
		return
	}
	fmt.Printf("%+v", clone)
	fmt.Printf("%+v", v)
	// increase version for modify operations

	//sent update selectedSet request
	version = strconv.Itoa(v.Version)
	ModifySelectedHostnamesOnConfig(AkamaiHost, configID, version, mode, newSet)

}

func Send(method, url string, data []byte) []byte {
	client := &http.Client{}
	payload := bytes.NewReader(data)
	golog.Info("Ready to send data " + string(data) + "\n to URL " + url)

	req, _ := http.NewRequest(method, url, payload)
	accessToken := os.Getenv("AKAMAI_EDGEGRID_ACCESS_TOKEN")
	clientToken := os.Getenv("AKAMAI_EDGEGRID_CLIENT_TOKEN")
	clientSecret := os.Getenv("AKAMAI_EDGEGRID_CLIENT_SECRET")

	params := edgegrid.NewAuthParams(req, accessToken, clientToken, clientSecret)
	auth := edgegrid.Auth(params)

	req.Header.Add("Authorization", auth)
	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	resp, _ := client.Do(req)
	contents, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		golog.Error("Error!")
		return make([]byte, 0)
	}
	golog.Info("Request status code :")
	golog.Info(resp.StatusCode)
	golog.Info("Received data: " + string(contents))
	if resp.StatusCode > 201 {
		golog.Fatal("Response code is not OK!")
	}
	return contents
}

// only for KSD and Advanced AAP
func ListSelectableOnConfig(hostname, configID, version string) []byte {
	golog.Info("Starting ListSelectable func")
	url := hostname + "/appsec/v1/configs/" + configID + "/versions/" + version + "/selectable-hostnames"
	return Send("GET", url, []byte{})

}

func ListSelectedOnPolicy(hostname, configID, version, policyID string) []byte {
	golog.Info("Starting ListSelected func")
	url := hostname + "/appsec/v1/configs/" + configID + "/versions/" + version + "/security-policies/" + policyID + "/selected-hostnames"
	return Send("GET", url, []byte{})
}

func GetConfig(hostname string) []byte {
	golog.Info("Starting GetConfig func")
	url := hostname + "/appsec/v1/configs/"
	return Send("GET", url, []byte{})
}

func CloneConfig(hostname, configID, version string, data []byte) []byte {
	golog.Info("Starting CloneConfig func")
	url := hostname + "/appsec/v1/configs/" + configID + "/versions"
	return Send("POST", url, data)
}

func ModifySelectedHostnamesOnConfig(hostname, configID, version, mode string, data []byte) {
	golog.Info("Starting Modify func")
	url := hostname + "/appsec/v1/configs/" + configID + "/versions/" + version + "/selected-hostnames"

	Send("PUT", url, data)
}
