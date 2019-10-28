package idp

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/redhat-developer/odo-fork/pkg/config"
)

// IDP constants
const (
	IDPYaml     = "idp.yaml"
	RuntimeTask = "Runtime"
	SharedTask  = "Shared"
	IDPYamlPath = "/.udo/" + IDPYaml
)

// IDPArtifact holds the IDP artifacts info
type IDPArtifact struct {
	FileBytes []byte
	FileName  string
}

// Get loads in the project's idp.yaml from disk
func Get() (*IDP, error) {
	// Retrieve the IDP.yaml file
	udoDir, err := config.GetUDOFolder("")
	if err != nil {
		return nil, fmt.Errorf("unabled to find .udo folder in current directory")
	}
	idpFile := path.Join(udoDir, IDPYaml)

	// Load it into memory
	idpBytes, err := readIDPFile(idpFile)
	if err != nil {
		return nil, err
	}

	// Unmarshall the yaml into the IDP struct and return it
	var idp IDP
	err = yaml.Unmarshal(idpBytes, &idp)
	return &idp, err
}

// DownloadIDP downloads the idp.yaml for the iterative-dev pack at the given URL
func DownloadIDP(idpURL string, artifactsURL []string) error {
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

	var idpArtifacts []IDPArtifact

	for _, artifactURL := range artifactsURL {
		resp, err := httpClient.Get(artifactURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		artifactBytes, err := ioutil.ReadAll(resp.Body)
		fileName := filepath.Base(artifactURL)

		artifact := IDPArtifact{
			FileBytes: artifactBytes,
			FileName:  fileName,
		}

		idpArtifacts = append(idpArtifacts, artifact)
	}

	fmt.Println("MJF calling writeToUDOFolder")
	// Write the idp.yaml and it's artifacts to disk
	return writeToUDOFolder(idpBytes, idpArtifacts)
}

// CopyLocalIDP reads in a local idp from disk and copies it to the UDO config folder
func CopyLocalIDP(idpFile string, artifactFiles []string) error {
	idpBytes, err := readIDPFile(idpFile)
	if err != nil {
		return err
	}

	// Before writing to the UDO config folder, verify the idp.yaml is valid
	_, err = parseIDPYaml(idpBytes)
	if err != nil {
		return err
	}

	var idpArtifacts []IDPArtifact

	for _, artifactFile := range artifactFiles {
		artifactBytes, err := readIDPFile(artifactFile)
		if err != nil {
			return err
		}
		fileName := filepath.Base(artifactFile)

		artifact := IDPArtifact{
			FileBytes: artifactBytes,
			FileName:  fileName,
		}

		idpArtifacts = append(idpArtifacts, artifact)
	}

	// Write the idp.yaml to the local UDO config folder
	return writeToUDOFolder(idpBytes, idpArtifacts)
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

// readIDPFile reads in an IDP file from disk at the specified path
func readIDPFile(idpFile string) ([]byte, error) {
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
func writeToUDOFolder(idpBytes []byte, idpArtifacts []IDPArtifact) error {
	// Write the idp.yaml to disk
	udoDir, err := config.GetUDOFolder("")
	if err != nil {
		return err
	}
	artifactDir := path.Join(udoDir, "bin")
	if _, err := os.Stat(artifactDir); os.IsNotExist(err) {
		os.Mkdir(artifactDir, 0755)
	}

	idpPath := path.Join(udoDir, IDPYaml)
	err = ioutil.WriteFile(idpPath, idpBytes, 0644)
	if err != nil {
		return err
	}

	for _, idpArtifact := range idpArtifacts {
		artifactPath := path.Join(artifactDir, idpArtifact.FileName)
		err = ioutil.WriteFile(artifactPath, idpArtifact.FileBytes, 0755)
		if err != nil {
			return err
		}

	}

	return nil
}
