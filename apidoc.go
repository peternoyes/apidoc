package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cosiner/gohper/lib/types"

	"github.com/cosiner/gohper/lib/sys"
)

type section struct {
	headerLine string
	headers    []string
	datas      []string
}

type APIs struct {
	top []*API
	sub map[string]*API
	sync.Mutex
}

type API struct {
	name  string
	desc  []string
	reqs  []*section
	resps []*section
	subs  []string
}

func (a *API) WriteMarkdown(w io.Writer, index int, sub map[string]*API) {
	if index != -1 && len(a.desc) != 0 {
		fmt.Fprintf(w, "### %d. %s\n", index+1, a.desc[0])
		for _, d := range a.desc[1:] {
			fmt.Fprintf(w, "%s  \n", d)
		}
	}
	writeSection(w, index, "Request", a.reqs)
	writeSection(w, index, "Response", a.resps)
	for _, s := range a.subs {
		if sa := sub[s]; sa != nil {
			sa.WriteMarkdown(w, -1, sub)
		}
	}
}

func writeSection(w io.Writer, i int, name string, secs []*section) {
	if len(secs) > 0 {
		if i != -1 {
			fmt.Fprintf(w, "* **%s**\n", name)
		}
		for _, sec := range secs {
			fmt.Fprintf(w, "    * %s  \n", sec.headerLine)
			for _, h := range sec.headers {
				fmt.Fprintf(w, "      %s  \n", h)
			}
			for _, d := range sec.datas {
				fmt.Fprintf(w, "      %s  \n", d)
			}
		}
	}
}

func (as *APIs) WriteMarkDown(w io.Writer) {
	for i, a := range as.top {
		a.WriteMarkdown(w, i, as.sub)
		fmt.Fprintln(w)
	}
}

func (as *APIs) Add(a *API) {
	as.Lock()
	as.top = append(as.top, a)
	as.Unlock()
}

func (as *APIs) AddSub(name string, a *API) {
	as.Lock()
	if as.sub == nil {
		as.sub = make(map[string]*API)
	}
	as.sub[name] = a
	as.Unlock()
}

var file string
var comment string
var ext string
var outputType string
var overwrite bool

func parseCLI() {
	flag.Usage = func() {
		fmt.Println(os.Args[0] + " [OPTIONS] [FILE/DIR]")
		flag.PrintDefaults()
	}
	flag.StringVar(&file, "f", "", "save result to file")
	flag.BoolVar(&overwrite, "o", false, "overwrite exist file content")
	flag.StringVar(&comment, "c", "//", "comment start")
	flag.StringVar(&ext, "e", "go", "file extension name")
	flag.StringVar(&outputType, "t", "md", "output format, currently only support markdown")
	flag.Parse()
}

var String = types.UnsafeString
var Bytes = types.UnsafeBytes
var Trim = strings.TrimSpace
var StartWith = strings.HasPrefix

var as = APIs{}

func main() {
	parseCLI()
	var path string
	args := flag.Args()
	if len(args) == 0 {
		path = "."
	} else {
		path = args[0]
	}
	ext = "." + ext
	wg := sync.WaitGroup{}
	if err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if !info.IsDir() && filepath.Ext(info.Name()) == ext {
				wg.Add(1)
				go process(path, &wg)
			}
		}
		return err
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	} else {
		wg.Wait()
		if len(as.top) == 0 {
			fmt.Println("No files contains api in this dir or file")
			return
		}
		switch outputType {
		case "md":
			if file != "" {
				if err := sys.OpenOrCreateFor(file, overwrite, func(f *os.File) error {
					as.WriteMarkDown(f)
					return nil
				}); err != nil {
					fmt.Fprintln(os.Stderr, err)
				}
			} else {
				as.WriteMarkDown(os.Stdout)
			}
		default:
			fmt.Fprintln(os.Stderr, "Sorry, currently only support markdown format")
		}
	}
}

const (
	PARSE_INIT = iota
	PARSE_API
	PARSE_REQ
	PARSE_RESP

	PARSE_STATUS
	PARSE_HEADER
	PARSE_DATA

	DEF_API         = "@API"
	DEF_SUBRESP     = "@SubResp"
	DEF_RESPINCLUDE = "@RespIncl"
	DEF_RESP        = "@Resp"
	DEF_REQ         = "@Req"
	DEF_ENDAPI      = "@EndAPI"
	DEF_DATA        = "->"
)

func process(path string, wg *sync.WaitGroup) {
	sectionState := PARSE_INIT
	dataState := PARSE_INIT
	var a *API
	var name string
	var curr *section
	if err := sys.FilterFileContent(path, false, func(linum int, line []byte) ([]byte, error) {
		line = bytes.TrimSpace(line)
		if !StartWith(String(line), comment) {
			sectionState = PARSE_INIT
			return nil, nil
		}
		line = bytes.TrimSpace(line[len(comment):])
		index := bytes.Index(line, Bytes(comment))
		if index > 0 {
			line = bytes.TrimSpace(line[:index])
		}
		if len(line) == 0 {
			return nil, nil
		}
		linestr := string(line)
		switch {
		case StartWith(linestr, DEF_RESPINCLUDE):
			if a != nil {
				a.subs = append(a.subs, Trim(linestr[len(DEF_RESPINCLUDE):]))
			}
		case StartWith(linestr, DEF_SUBRESP):
			name = Trim(linestr[len(DEF_SUBRESP):])
			fallthrough
		case StartWith(linestr, DEF_API):
			sectionState = PARSE_API
			a = &API{}
			if name == "" {
				if desc := Trim(linestr[len(DEF_API):]); desc != "" {
					a.desc = append(a.desc, desc)
				}
				as.Add(a)
			} else {
				as.AddSub(name, a)
			}
		case StartWith(linestr, DEF_REQ):
			if sectionState != PARSE_INIT {
				sectionState = PARSE_REQ
				curr = &section{}
				a.reqs = append(a.reqs, curr)
				dataState = PARSE_STATUS
			}
		case StartWith(linestr, DEF_RESP):
			if sectionState != PARSE_INIT {
				sectionState = PARSE_RESP
				dataState = PARSE_STATUS
				curr = &section{}
				a.resps = append(a.resps, curr)
			}
		case StartWith(linestr, DEF_ENDAPI):
			sectionState = PARSE_INIT
		default:
			if sectionState == PARSE_INIT {
			} else if sectionState == PARSE_API {
				a.desc = append(a.desc, linestr)
			} else {
				switch dataState {
				case PARSE_STATUS:
					curr.headerLine = linestr
					dataState = PARSE_HEADER
				case PARSE_HEADER:
					if !StartWith(linestr, DEF_DATA) {
						curr.headers = append(curr.headers, linestr)
						break
					}
					dataState = PARSE_DATA
					fallthrough
				case PARSE_DATA:
					if StartWith(linestr, DEF_DATA) {
						curr.datas = append(curr.datas, Trim(linestr[len(DEF_DATA):]))
					}
				}
			}
		}
		return nil, nil
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	wg.Done()
}
