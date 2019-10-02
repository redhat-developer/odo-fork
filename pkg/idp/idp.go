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

const IDPYaml = "idp.yaml"

// Get loads in the project's idp.yaml from disk
func Get() (*IDP, error) {
	// Retrieve the IDP.yaml file
	udoDir, err := config.GetUDOFolder("")
	if err != nil {
		return nil, fmt.Errorf("unabled to find .udo folder in current directory")
	}
	idpFile := path.Join(udoDir, IDPYaml)

	// Load it into memory
	idpBytes, err := readIDPYaml(idpFile)
	if err != nil {
		return nil, err
	}

	// Unmarshall the yaml into the IDP struct and return it
	var idp IDP
	err = yaml.Unmarshal(idpBytes, &idp)
	return &idp, err
}

// DownloadIDP downloads the idp.yaml for the iterative-dev pack at the given URL
func DownloadIDP(idpURL string) error {
	// Download the IDP index.json
	var httpClient = &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(idpURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	idpBytes, err := ioutil.ReadAll(resp.Body)

	// Before writing to disk, verify that the IDP.yaml is valid (can be unmarshalled), return an error if it failed
	_, err = parseIDPYaml(idpBytes)
	if err != nil {
		return err
	}
	// Write the idp.yaml to disk
	return writeToUDOFolder(idpBytes)
}

// CopyLocalIDP reads in a local idp from disk and copies it to the UDO config folder
func CopyLocalIDP(idpFile string) error {
	idpBytes, err := readIDPYaml(idpFile)
	if err != nil {
		return err
	}

	// Before writing to the UDO config folder, verify the idp.yaml is valid
	_, err = parseIDPYaml(idpBytes)
	if err != nil {
		return err
	}

	// Write the idp.yaml to the local UDO config folder
	return writeToUDOFolder(idpBytes)
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

// parseIDPYaml takes in an array of bytes and tries to unmarshall it into the IDP yaml struct
// Returns the unmarshalle
func parseIDPYaml(idpBytes []byte) (*IDP, error) {
	var idp IDP
	err := yaml.Unmarshal(idpBytes, &idp)
	if err != nil {
		return nil, fmt.Errorf("unable to download Iterative-Dev pack, idp.yaml invalid: %s", err)
	}
	return &idp, nil
}

// readIDPYaml reads in an idp.yaml file from disk at the specified path
func readIDPYaml(idpFile string) ([]byte, error) {
	var idpBytes []byte

	// Check if the file exists, and return an error if it doesn't.
	if _, err := os.Stat(idpFile); os.IsNotExist(err) {
		if err != nil {
			return nil, fmt.Errorf("Unable to find %s at %s", IDPYaml, idpFile)
		}
	} else {
		file, err := os.Open(idpFile)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		idpBytes, err = ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
	}

	return idpBytes, nil
}

// writeToUDOFolder takes in bytes representing an idp.yaml file or tar archive and writes it to disk
func writeToUDOFolder(bytes []byte) error {
	// Write the idp.yaml to disk
	udoDir, err := config.GetUDOFolder("")
	if err != nil {
		return err
	}

	idpPath := path.Join(udoDir, IDPYaml)
	return ioutil.WriteFile(idpPath, bytes, 0644)
}
