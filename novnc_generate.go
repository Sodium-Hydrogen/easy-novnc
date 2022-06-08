// +build novnc_generate

package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/shurcooL/vfsgen"
	"github.com/spkg/zipfs"
)

const targetRootDir = "noVNC/"
const githubAPI = "https://api.github.com/repos/novnc/noVNC/"
const vncScript = `
	try {
		function parseQuery(e){for(var o=e.split("&"),n={},t=0;t<o.length;t++){var d=o[t].split("="),p=decodeURIComponent(d[0]),r=decodeURIComponent(d[1]);if(void 0===n[p])n[p]=decodeURIComponent(r);else if("string"==typeof n[p]){var i=[n[p],decodeURIComponent(r)];n[p]=i}else n[p].push(decodeURIComponent(r))}return n};
		fetch(parseQuery(window.location.search.replace(/^\?/, ""))["path"]).then(function(resp) {
			return resp.text();
		}).then(function (txt) {
			if (txt.indexOf("not websocket") == -1) alert(txt);
		});
	} catch (ex) {
		console.log(ex);
	}
`
type release_data struct {
	Tag_name string
	Zipball_url string
}

func main() {

	// Allow dynamic referencing to any release or latest commit
	var zipURL string
	var comment string

	if len(os.Args) == 1{
		resp, err := http.Get(githubAPI+"releases/latest")
		if err != nil {
			panic(err)
		} else if resp.StatusCode != 200 {
			panic("Invalid url");
		}

		var data release_data
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(body, &data)
		if err != nil {
			panic(err)
		}
		zipURL = data.Zipball_url
		comment = "noVNC is the latest release of noVNC from GitHub as a http.FileSystem"
	} else if len(os.Args) == 2 && os.Args[1] != "nightly" {
		zipURL = githubAPI+"zipball/"+os.Args[1]
		comment = "noVNC is the tagged version ("+os.Args[1]+") of noVNC from GitHub as a http.FileSystem"
	} else if len(os.Args) == 2 {
		zipURL = githubAPI+"zipball"
		comment = "noVNC is the latest commit of noVNC from GitHub as a http.FileSystem"
	} else {
		os.Exit(1)
	}

	resp, err := http.Get(zipURL)
	if err != nil {
		panic(err)
	}

	f, err := ioutil.TempFile("", "novnc*.zip")
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		panic(err)
	}

	f.Close()
	resp.Body.Close()

	err = modifyZip(f.Name())
	if err != nil {
		panic(err)
	}

	zfs, err := zipfs.New(f.Name())
	if err != nil {
		panic(err)
	}

	err = vfsgen.Generate(zfs, vfsgen.Options{
		Filename:        "novnc_generated.go",
		PackageName:     "main",
		VariableName:    "noVNC",
		VariableComment: comment,
	})
	if err != nil {
		panic(err)
	}
}

// modifyZip adds the custom easy-novnc code into the noVNC zip file.
func modifyZip(zf string) error {
	buf, err := ioutil.ReadFile(zf)
	if err != nil {
		return err
	}

	zr, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		return err
	}

	f, err := os.Create(zf)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	var found bool
	// The root file to rename all to
	parent := ""
	for _, e := range zr.File {
		var w io.Writer

		rc, err := e.Open()
		if err != nil {
			return err
		}

		fbuf, err := ioutil.ReadAll(rc)
		if err != nil {
			return err
		}

		if e.Name[len(e.Name)-1] == '/' {
			if strings.Count(e.Name, "/") == 1{
				parent = e.Name
			}
		}

		if parent != "" {
			e.FileHeader.Name = strings.Replace(e.FileHeader.Name, parent, targetRootDir, 1)
		}

		if filepath.Base(e.Name) == "vnc.html" {
			found = true

			fbuf = bytes.ReplaceAll(fbuf, []byte("</body>"), []byte(fmt.Sprintf("<script>%s</script></body>", vncScript)))
			fi, err := os.Stat("novnc_generate.go")
			if err != nil {
				return err
			}
			w, err = zw.CreateHeader(&zip.FileHeader{
				Name:          e.Name,
				Flags:         e.Flags,
				Method:        e.Method,
				Modified:      fi.ModTime(),
				Extra:         e.Extra,
				ExternalAttrs: e.ExternalAttrs,
			})
		} else {
			w, err = zw.CreateHeader(&e.FileHeader)
		}

		if err != nil {
			return err
		}

		_, err = io.Copy(w, bytes.NewReader(fbuf))
		if err != nil {
			return err
		}
		rc.Close()
	}

	if !found {
		return errors.New("could not find vnc.html")
	}

	return nil
}
