package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type Ext struct {
	Name      string `json:"name"`
	Publisher string `json:"publisher"`
	Version   string `json:"version"`
	Sha256    string `json:"sha256"`
}

func getVersion(publisher, name string) string {

	r, err := http.Get(fmt.Sprintf("https://marketplace.visualstudio.com/items?itemName=%s.%s", publisher, name))
	if err != nil {
		panic(err)
	}

	defer r.Body.Close()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	var re = regexp.MustCompile(`(?m)"VersionValue":"([^"]+)"`)

	m := re.FindAllStringSubmatch(string(b), -1)

	if m[0][1] == "" {
		panic(string(b))
	}

	return m[0][1]
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

	path := flag.String("path", "ext.json", "")
	flag.Parse()

	b, err := os.ReadFile(*path)
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

			version := getVersion(ext.Publisher, ext.Name)
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

	output, err := json.MarshalIndent(exts, "", "    ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(output))

	if modified {
		err = os.WriteFile("./ext.json", output, 0664)
		if err != nil {
			panic(err)
		}
	}
}
