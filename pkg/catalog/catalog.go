package catalog

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const DefaultIDPRepo = "https://raw.githubusercontent.com/johnmcollier/iterative-dev-packs/master"
const DefaultIDPCatalog = DefaultIDPRepo + "/index.json"

// CatalogEntry represents an entry in the index.json file for IDPs
type CatalogEntry struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Language    string            `json:"language"`
	Framework   string            `json:"framework"`
	Devpack     map[string]string `json:"devpack"`
}

// List lists all the available Iterative-Dev Packs
func List(localIndexJSON string) ([]CatalogEntry, error) {

	var idpList []CatalogEntry

	// If a local index.json file wasn't passed in, see if one is cached
	var indexJSONFile string
	if localIndexJSON == "" {
		indexJSONFile = path.Join(os.TempDir(), ".udo", "index.json")
	} else {
		indexJSONFile = localIndexJSON
	}

	// Load the index.json file into memory
	var jsonBytes []byte
	if _, err := os.Stat(indexJSONFile); os.IsNotExist(err) {
		jsonBytes, err = downloadIndexJSON()
		if err != nil {
			return nil, fmt.Errorf("unable to download index.json from %s", DefaultIDPCatalog)
		}
	} else {
		file, err := os.Open(indexJSONFile)
		if err != nil {
			return nil, err
		}
		jsonBytes, err = ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
	}

	err := json.Unmarshal(jsonBytes, &idpList)
	return idpList, err
}

// Search searches for the IDP in the catalog
func Search(name string, localIndexJSON string) ([]string, error) {
	var result []string
	idpList, err := List(localIndexJSON)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list Iterative-Dev Packs")
	}

	// do a partial search in all the Iterative-Dev Packs
	for _, idp := range idpList {
		// we only show IDP entries that contain the search term in their name and that have at least one devpack yaml linked
		if strings.Contains(idp.Name, name) {
			result = append(result, idp.Name)
		}
	}

	return result, nil
}

// Exists returns true if the given iterative-dev pack type is valid, false if not
func Exists(idpName string, localIndexJSON string) (bool, error) {

	catalogList, err := List(localIndexJSON)
	if err != nil {
		return false, errors.Wrapf(err, "unable to list catalog")
	}

	for _, supported := range catalogList {
		if idpName == supported.Name {
			return true, nil
		}
	}
	return false, nil
}

// Get returns the first IDP matching the specified name, if none are found, an error is returned
func Get(name string, localIndexJSON string) (*CatalogEntry, error) {
	idpList, err := List(localIndexJSON)
	if err != nil {
		return nil, err
	}

	for _, idp := range idpList {
		if idp.Name == name {
			return &idp, nil
		}
	}
	return nil, fmt.Errorf("Could not find an IDP matching the name %s", name)
}

// downloadIndexJSON downloads the index.json to a temp directory for the CLI to access
func downloadIndexJSON() ([]byte, error) {
	// Download the IDP index.json
	var httpClient = &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(DefaultIDPCatalog)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	jsonBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return jsonBytes, err
}
