package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

const (
	ENGINE = "Microsoft.VisualStudio.Code.Engine"
)

type Ext struct {
	Name      string `json:"name"`
	Publisher string `json:"publisher"`
	Version   string `json:"version"`
	Sha256    string `json:"sha256"`
}

type QueryResultVersion struct {
	Version     string    `json:"version"`
	LastUpdated time.Time `json:"lastUpdated"`
	Properties  []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"properties"`
}

type QueryResult struct {
	Results []struct {
		Extensions []struct {
			Publisher struct {
				PublisherName string `json:"publisherName"`
			} `json:"publisher"`
			ExtensionName string               `json:"extensionName"`
			Versions      []QueryResultVersion `json:"versions"`
		} `json:"extensions"`
		// PagingToken    any `json:"pagingToken"`
	} `json:"results"`
}

func getVersion(publisher, name, engine string) string {

	targetVersion, err := semver.NewVersion(engine)
	if err != nil {
		panic(err)
	}

	// https://github.com/NixOS/nixpkgs/blob/014ba34da35c14b6cad82b07a1245f3251e78645/pkgs/applications/editors/vscode/extensions/ms-python.python/default.nix#L64
	req, err := http.NewRequest(http.MethodPost,
		"https://marketplace.visualstudio.com/_apis/public/gallery/extensionquery",
		strings.NewReader(fmt.Sprintf(`{"filters":[{"criteria":[{"filterType":7,"value":"%s.%s"}]}],"flags":16}`, publisher, name)))
	if err != nil {
		panic(err)
	}

	req.Header.Set("accept", "application/json;api-version=3.0-preview.1")
	req.Header.Set("content-type", "application/json")

	c := &http.Client{
		Timeout: time.Second * 120,
	}

	res, err := c.Do(req)
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()

	var result QueryResult
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	var versions []QueryResultVersion

LABEL:
	for i := range result.Results {
		for j := range result.Results[i].Extensions {
			if result.Results[i].Extensions[j].Publisher.PublisherName == publisher && result.Results[i].Extensions[j].ExtensionName == name {
				versions = result.Results[i].Extensions[j].Versions
				break LABEL
			}
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].LastUpdated.After(versions[j].LastUpdated)
	})

	for _, version := range versions {
		for _, property := range version.Properties {
			if property.Key == ENGINE {
				v, err := semver.NewConstraint(property.Value)
				if err != nil {
					panic(err)
				}

				if v.Check(targetVersion) {
					return version.Version
				}

				break
			}
		}
	}

	panic("failed")
}

func getHash(publisher, name, version string) string {

	u := fmt.Sprintf("https://%s.gallery.vsassets.io/_apis/public/gallery/publisher/%s/extension/%s/%s/assetbyname/Microsoft.VisualStudio.Services.VSIXPackage",
		publisher,
		publisher,
		name,
		version)

	cmd := exec.Command("bash", "-c",
		fmt.Sprintf("nix shell nixpkgs#nix-prefetch -c nix-prefetch-url %s", u))
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	outputs := bytes.Split(output, []byte("\n"))

	hash := strings.TrimSpace(string(outputs[0]))
	if len(hash) != 52 {
		panic(hash)
	}

	return hash

}

func main() {

	list := flag.String("list", "ext.json", "")
	output := flag.String("output", "ext.json", "")
	engine := flag.String("engine", "", "")
	force := flag.Bool("force", false, "")
	flag.Parse()

	b, err := os.ReadFile(*list)
	if err != nil {
		panic(err)
	}

	exts := map[string]map[string]Ext{}
	err = json.Unmarshal(b, &exts)
	if err != nil {
		panic(err)
	}

	fmt.Println(exts)

	modified := false

	for publisher := range exts {
		for name, ext := range exts[publisher] {
			fmt.Println(modified, publisher, name, ext)

			if ext.Publisher != publisher {
				panic("publisher")
			}

			if ext.Name != name {
				panic("name")
			}

			version := getVersion(ext.Publisher, ext.Name, *engine)
			if version != ext.Version {
				modified = true
				ext.Version = version
			}

			hash := getHash(ext.Publisher, ext.Name, ext.Version)
			if hash != ext.Sha256 {
				modified = true
				ext.Sha256 = hash
			}

			fmt.Println(modified, publisher, name, ext)

			exts[publisher][name] = ext
		}
	}

	result, err := json.MarshalIndent(exts, "", "    ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(result))

	if modified || *force {
		err = os.WriteFile(*output, result, 0664)
		if err != nil {
			panic(err)
		}
	}
}
