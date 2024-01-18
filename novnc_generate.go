//go:build novnc_generate

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
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

var patchNotice = map[string]string{
	"js": `
/**begin easy-novnc patch**/
%s
/***end easy-novnc patch***/
`,

	"html": `
<!-- begin easy-novnc patch !-->
%s
<!-- end easy-novnc patch !-->
`,
	"css": `
/**begin easy-novnc patch**/
%s
/***end easy-novnc patch***/
`}
var mobilePatches = [][]string{
	{"noVNC/vnc.html", "</body>", "</body>", `
<script>
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
</script>
</body>
`},
	{"noVNC/core/rfb.js", "this.dragViewport = false;", "this*;", `
	\1

	let _evnc_params = new URLSearchParams(document.location.search);
	let _evnc_min_width = parseInt(_evnc_params.get("min_width") || 1);
	let _evnc_min_height = parseInt(_evnc_params.get("min_height") || 1);

	if (_evnc_min_width > window.innerWidth || _evnc_min_height > window.innerHeight){
		this.dragViewport = true;
	}
`},
	{"noVNC/core/rfb.js", " _requestRemoteResize() {", "const size = this._screenSize();", `
	let _evnc_params = new URLSearchParams(document.location.search);
	let _evnc_min_width = parseInt(_evnc_params.get("min_width") || 1);
	let _evnc_min_height = parseInt(_evnc_params.get("min_height") || 1);
	const size = this._screenSize();

	size.w = _evnc_min_width < size.w ? size.w: _evnc_min_width;
	size.h = _evnc_min_height < size.h ? size.h: _evnc_min_height;
`},
	{"noVNC/core/rfb.js", "_handleGesture(ev) {", "let pos = clientToElement*;", `
	\1

	let _evnc_ev_detail_type = ev.detail.type;
	let _evnc_viewportDragging = this.dragViewport;

	if (_evnc_ev_detail_type == "twodrag" && _evnc_viewportDragging){
		ev.detail._evnc_overridden = true;
		_evnc_ev_detail_type = "drag";
		this.dragViewport = false;
		pos = clientToElement(ev.detail._evnc_lastClientX, ev.detail._evnc_lastClientY, this._canvas);
	}	
`},
	{"noVNC/core/rfb.js", "switch (ev.detail.type) {", "switch (ev.detail.type) {", `switch (_evnc_ev_detail_type) {`},
	{"noVNC/core/rfb.js", "switch (ev.detail.type) {", "switch (ev.detail.type) {", `switch (_evnc_ev_detail_type) {`},
	{"noVNC/core/rfb.js", "switch (ev.detail.type) {", "switch (ev.detail.type) {", `switch (_evnc_ev_detail_type) {`},
	{"noVNC/core/rfb.js", "case 'gestureend':", "}*}", `
\1
this.dragViewport = _evnc_viewportDragging;
`},
	{"noVNC/core/rfb.js", "_fakeMouseMove(ev, elementX, elementY) {", "this._cursor.move*;", `
	if(ev.detail._evnc_overridden !== undefined){
		this._cursor.move(ev.detail._evnc_lastClientX, ev.detail._evnc_lastClientY)
	}else{
		\1
	}

`},
	{"noVNC/core/input/gesturehandler.js", " _pushEvent(type) {", "detail['clientY'] = pos.y;", `
	\1
	detail['_evnc_lastClientX'] = pos.x;
	detail['_evnc_lastClientY'] = pos.y;
	if (!isNaN(avg.last.x)){
		detail['_evnc_lastClientX'] = avg.last.x;
		detail['_evnc_lastClientY'] = avg.last.y;
	}
`},
	{"noVNC/app/styles/base.css", "@media screen and (max-width: 640px){", "}", `}
	#noVNC_control_bar > .noVNC_scroll{
		display: flex;
	}
	#noVNC_control_bar_anchor,
	#noVNC_control_bar_anchor .noVNC_vcenter{
		right: 40px;
		justify-content: start;
	}
	#noVNC_control_bar_anchor.noVNC_right,
	#noVNC_control_bar_anchor.noVNC_right .noVNC_vcenter{
		left: 40px;
		justify-content: end;
	}
	#noVNC_control_bar_handle{
		height: unset;
		transform: unset !important;
		bottom: 0;
		width: calc(100% + 60px);
		left: -30px;
		transition: 0.2s;
	}
	#noVNC_control_bar.noVNC_open #noVNC_control_bar_handle{
		height: 50px;
		top: 30px;
		transform: rotate(90deg)!important;
		left: calc(100% - 75px);
		width: 40px;
		bottom: 0;
	}
	.noVNC_right #noVNC_control_bar.noVNC_open #noVNC_control_bar_handle{
		top: -30px;
		left: 25px;
	}
	#noVNC_modifiers{
		display: flex;
	}
	.noVNC_panel.noVNC_open{
		transform: translateY(60px);
	}
	.noVNC_right .noVNC_panel.noVNC_open{
		transform: translateY(-60px);
	}
	#noVNC_hint_anchor{
		justify-content: center;
	}
	#noVNC_control_bar_hint {
		display: flex;
		align-items: center;
		justify-content: start;
	}
	#noVNC_control_bar_hint::after {
		content: "▼";
		font-size: xxx-large;
		color: rgba(0, 0, 0, 0);
		text-shadow: rgba(110, 132, 163, 0.8) 0 0 4px;
	}
	#noVNC_control_bar_anchor.noVNC_right + #noVNC_hint_anchor #noVNC_control_bar_hint{
		justify-content: end;
	}
	#noVNC_control_bar_anchor.noVNC_right + #noVNC_hint_anchor #noVNC_control_bar_hint::after{
		content: "▲";
	}
`}}

type release_data struct {
	Tag_name    string
	Zipball_url string
}

func main() {

	// Allow dynamic referencing to any release or latest commit
	var zipURL string
	var comment string

	if len(os.Args) == 1 {
		resp, err := http.Get(githubAPI + "releases/latest")
		if err != nil {
			panic(err)
		} else if resp.StatusCode != 200 {
			panic("Invalid url")
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
		zipURL = githubAPI + "zipball/" + os.Args[1]
		comment = "noVNC is the tagged version (" + os.Args[1] + ") of noVNC from GitHub as a http.FileSystem"
	} else if len(os.Args) == 2 {
		zipURL = githubAPI + "zipball"
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

	var mobilePatchCount int
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
			if strings.Count(e.Name, "/") == 1 {
				parent = e.Name
			}
		}

		if parent != "" {
			e.FileHeader.Name = strings.Replace(e.FileHeader.Name, parent, targetRootDir, 1)
		}
		fi, err := os.Stat("novnc_generate.go")
		if err != nil {
			return err
		}

		var editedFile bool
		var patchesOnFile int

		for _, replacement := range mobilePatches {
			if e.Name == replacement[0] {
				var needle = []byte(replacement[1])
				var remove = replacement[2]
				var endRemove []byte
				var newData = replacement[3]

				var location = bytes.Index(fbuf, needle)
				if location != -1 {
					if strings.Index(remove, "*") != -1 && strings.Index(remove, "\\*") == -1 {
						endRemove = []byte(remove[strings.Index(remove, "*")+1:])
						remove = remove[:strings.Index(remove, "*")]
					}
					var beginning = bytes.Index(fbuf[location:], []byte(remove))
					if beginning != -1 {
						editedFile = true
						mobilePatchCount++
						patchesOnFile++

						beginning += location

						if remove != replacement[2] {
							remove = string(fbuf[beginning : bytes.Index(fbuf[beginning+1:], endRemove)+beginning+len(endRemove)+1])
						}

						newData = strings.ReplaceAll(newData, "\\1", remove)

						var newBuff = append(
							[]byte(fmt.Sprintf(patchNotice[filepath.Ext(e.Name)[1:]], newData)),
							fbuf[beginning+len(remove):]...,
						)
						fbuf = append(fbuf[:beginning], newBuff...)
					}
				}
			}
		}
		if editedFile {
			fmt.Printf("Applied %d patch(es) to:  %s\n", patchesOnFile, e.Name)
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

	if mobilePatchCount != len(mobilePatches) {
		return errors.New(
			fmt.Sprintf(
				"Mismatch in count of applied patches (%d) and available patches (%d)",
				mobilePatchCount,
				len(mobilePatches)))
	}

	return nil
}
