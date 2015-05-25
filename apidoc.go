package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cosiner/gohper/errors"
	"github.com/cosiner/gohper/os2/file"
	"github.com/cosiner/gohper/strings2"
	"github.com/cosiner/gohper/unsafe2"
)

type section struct {
	headerLine string
	headers    []string
	datas      []string

	subheaders []string
}

type API struct {
	name  string
	desc  []string
	req   *section
	resps []*section

	subresps []string
	subapis  []string
}

type APIs struct {
	categories map[string][]*API
	subresp    map[string]*section
	subapi     map[string]*API
	subheaders map[string]*section
	sync.Mutex
}

func (as *APIs) AddAPI(category string, a *API) {
	as.Lock()
	as.categories[category] = append(as.categories[category], a)
	as.Unlock()
}

func (as *APIs) AddSubResp(name string, resp *section) {
	as.Lock()
	as.subresp[name] = resp
	as.Unlock()
}

func (as *APIs) AddSubAPI(name string, a *API) {
	as.Lock()
	as.subapi[name] = a
	as.Unlock()
}

func (as *APIs) AddSubHeader(name string, sec *section) {
	as.Lock()
	as.subheaders[name] = sec
	as.Unlock()
}

var as = APIs{
	subresp:    make(map[string]*section),
	subapi:     make(map[string]*API),
	categories: make(map[string][]*API),
	subheaders: make(map[string]*section),
}

var fname string
var comment string
var ext string
var outputType string
var overwrite bool

func init() {
	flag.Usage = func() {
		fmt.Println(os.Args[0] + " [OPTIONS] [FILE/DIR]")
		flag.PrintDefaults()
	}
	flag.StringVar(&fname, "f", "", "save result to file")
	flag.BoolVar(&overwrite, "o", false, "overwrite exist file content")
	flag.StringVar(&comment, "c", "//", "comment start")
	flag.StringVar(&ext, "e", "go", "file extension name")
	flag.StringVar(&outputType, "t", "md", "output format, currently only support markdown")
	flag.Parse()
}

func main() {
	args := flag.Args()

	path := "."
	if len(args) != 0 {
		path = args[0]
	}
	ext = "." + ext

	wg := sync.WaitGroup{}
	errors.Fatal(
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && filepath.Ext(info.Name()) == ext {
				wg.Add(1)
				go process(path, &wg)
			}

			return nil
		}),
	)

	wg.Wait()

	errors.CondDo(len(as.categories) == 0,
		errors.Err("No files contains api in this dir or file"),
		errors.FatalAnyln)
	errors.CondDo(outputType != "md",
		errors.Err("Sorry, currently only support markdown format"),
		errors.FatalAnyln)

	if fname != "" {
		errors.Fatalln(file.OpenOrCreate(fname, overwrite, func(fd *os.File) error {
			as.WriteMarkDown(fd)

			return nil
		}))
	} else {
		as.WriteMarkDown(os.Stdout)
	}
}

const (
	PARSE_INIT = iota
	PARSE_API
	PARSE_BODY

	PARSE_STATUS
	PARSE_HEADER
	PARSE_DATA

	TAG_CATEGORY    = "@Category"
	TAG_AT_CATEGORY = "@C"

	TAG_API    = "@API"
	TAG_ENDAPI = "@EndAPI"

	TAG_SUBAPI  = "@SubAPI"
	TAG_APIINCL = "@APIIncl"

	TAG_HEADER     = "@Header"
	TAG_HEADERINCL = "@HeaderIncl"

	TAG_SUBRESP  = "@SubResp"
	TAG_RESPINCL = "@RespIncl"

	TAG_RESP = "@Resp"
	TAG_REQ  = "@Req"
	TAG_DATA = "->"
)

func process(path string, wg *sync.WaitGroup) {
	sectionState := PARSE_INIT
	dataState := PARSE_INIT
	var a *API
	var sec *section
	var category string = "global"

	err := file.Filter(path, func(linum int, line []byte) ([]byte, error) {
		line = bytes.TrimSpace(line)
		if !strings.HasPrefix(unsafe2.String(line), comment) {
			sectionState = PARSE_INIT
			return nil, nil
		}

		line = bytes.TrimSpace(line[len(comment):])
		index := bytes.Index(line, unsafe2.Bytes(comment))
		if index > 0 {
			line = bytes.TrimSpace(line[:index])
		}
		if len(line) == 0 {
			return nil, nil
		}

		linestr := string(line)
		switch {
		case strings.HasPrefix(linestr, TAG_RESPINCL):
			if a != nil {
				a.subresps = append(a.subresps, strings2.TrimSplit(linestr[len(TAG_RESPINCL):], ",")...)
			}

		case strings.HasPrefix(linestr, TAG_APIINCL):
			if a != nil {
				a.subapis = append(a.subapis, strings2.TrimSplit(linestr[len(TAG_APIINCL):], ",")...)
			}

		case strings.HasPrefix(linestr, TAG_HEADERINCL):
			if sec != nil {
				sec.subheaders = append(sec.subheaders, strings2.TrimSplit(linestr[len(TAG_HEADERINCL):], ",")...)
			}

		case strings.HasPrefix(linestr, TAG_SUBRESP):
			name := strings.TrimSpace(linestr[len(TAG_SUBRESP):])
			sectionState = PARSE_BODY
			dataState = PARSE_STATUS
			sec = &section{}
			as.AddSubResp(name, sec)

		case strings.HasPrefix(linestr, TAG_SUBAPI):
			a = &API{}
			sectionState = PARSE_API
			a.name = strings.TrimSpace(linestr[len(TAG_SUBAPI):])

			as.AddSubAPI(a.name, a)
		case strings.HasPrefix(linestr, TAG_HEADER):
			sec = &section{}
			sectionState = PARSE_BODY
			dataState = PARSE_HEADER

			as.AddSubHeader(strings.TrimSpace(linestr[len(TAG_HEADER):]), sec)
		case strings.HasPrefix(linestr, TAG_API):
			a = &API{}
			sectionState = PARSE_API
			name := strings.TrimSpace(linestr[len(TAG_API):])
			names := strings2.TrimSplit(name, TAG_AT_CATEGORY)
			a.name = names[0]

			if len(names) > 1 {
				as.AddAPI(names[1], a)
			} else {
				as.AddAPI(category, a)
			}

		case strings.HasPrefix(linestr, TAG_CATEGORY):
			category = strings.TrimSpace(linestr[len(TAG_CATEGORY):])

		case strings.HasPrefix(linestr, TAG_REQ):
			if sectionState != PARSE_INIT {
				sectionState = PARSE_BODY
				sec = &section{}
				a.req = sec
				dataState = PARSE_STATUS
			}

		case strings.HasPrefix(linestr, TAG_RESP):
			if sectionState != PARSE_INIT {
				sectionState = PARSE_BODY
				dataState = PARSE_STATUS
				sec = &section{}
				a.resps = append(a.resps, sec)
			}

		case strings.HasPrefix(linestr, TAG_ENDAPI):
			sectionState = PARSE_INIT

		default:
			if sectionState == PARSE_INIT {
				break
			} else if sectionState == PARSE_API {
				a.desc = append(a.desc, linestr)

				break
			}

			switch dataState {
			case PARSE_STATUS:
				sec.headerLine = linestr
				dataState = PARSE_HEADER

			case PARSE_HEADER:
				if !strings.HasPrefix(linestr, TAG_DATA) {
					sec.headers = append(sec.headers, linestr)

					break
				}
				dataState = PARSE_DATA
				fallthrough

			case PARSE_DATA:
				if strings.HasPrefix(linestr, TAG_DATA) {
					sec.datas = append(sec.datas, strings.TrimSpace(linestr[len(TAG_DATA):]))
				}
			}
		}

		return nil, nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	wg.Done()
}
