package idp

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/redhat-developer/odo-fork/pkg/config"
)

// Get loads in the project's idp.yaml from disk
func Get() (*IDP, error) {
	// Retrieve the IDP.yaml file
	udoDir, err := config.GetUDOFolder("")
	if err != nil {
		return nil, fmt.Errorf("unabled to find .udo folder in current directory")
	}
	idpFile := path.Join(udoDir, "idp.yaml")

	// Load it into memory
	var idpBytes []byte
	var idp IDP
	if _, err := os.Stat(idpFile); os.IsNotExist(err) {
		//jsonBytes, err = downloadIDPs()
		if err != nil {
			return nil, fmt.Errorf("Unable to find idp.yaml at %s", idpFile)
		}
	} else {
		file, err := os.Open(idpFile)
		if err != nil {
			return nil, err
		}
		idpBytes, err = ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
	}

	err = yaml.Unmarshal(idpBytes, &idp)
	return &idp, err
}

// DownloadIDPYaml downloads the idp.yaml for the iterative-dev pack at the given URL
func DownloadIDPYaml(idpURL string) error {
	// Download the IDP index.json
	var httpClient = &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(idpURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	idpBytes, err := ioutil.ReadAll(resp.Body)

	// Write the idp.yaml to disk
	udoDir, err := config.GetUDOFolder("")
	if err != nil {
		return err
	}
	idpPath := path.Join(udoDir, "idp.yaml")
	return ioutil.WriteFile(idpPath, idpBytes, 0644)
}

// GetPorts returns a list of ports that were set in the IDP. Unset ports will not be returned
func (i *IDP) GetPorts() []string {
	var portList []string
	if i.Spec.Runtime.Ports.InternalHTTPPort != "" {
		portList = append(portList, i.Spec.Runtime.Ports.InternalHTTPPort)
	}
	if i.Spec.Runtime.Ports.InternalHTTPSPort != "" {
		portList = append(portList, i.Spec.Runtime.Ports.InternalHTTPSPort)
	}
	if i.Spec.Runtime.Ports.InternalDebugPort != "" {
		portList = append(portList, i.Spec.Runtime.Ports.InternalDebugPort)
	}
	if i.Spec.Runtime.Ports.InternalPerformancePort != "" {
		portList = append(portList, i.Spec.Runtime.Ports.InternalPerformancePort)
	}

	return portList
}
