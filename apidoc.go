package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cosiner/gohper/ds/tree"
	"github.com/cosiner/gohper/errors"
	"github.com/cosiner/gohper/os2/file"
	"github.com/cosiner/gohper/slices"
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
var order string
var orderSep string
var tabsize int

var tabreplace = []byte("    ")
var tab = []byte("\t")

func init() {
	flag.Usage = func() {
		fmt.Println(os.Args[0] + " [OPTIONS] [FILE/DIR]")
		flag.PrintDefaults()
	}
	flag.StringVar(&fname, "f", "", "save result to file")
	flag.BoolVar(&overwrite, "o", false, "overwrite exist file content")
	flag.StringVar(&order, "order", "", "output order of category")
	flag.StringVar(&orderSep, "orderSep", ",", "seperator of multiple cateogry")
	flag.StringVar(&comment, "c", "//", "comment start")
	flag.StringVar(&ext, "e", "go", "file extension name")
	flag.StringVar(&outputType, "t", "md", "output format, currently only support markdown")
	flag.IntVar(&tabsize, "tab", 4, "tab size, tab will replaced with n space, if 0, don't replace")
	flag.Parse()

	if tabsize == 0 {
		tabreplace = tab
	} else {
		tabreplace = bytes.Repeat([]byte(" "), tabsize)
	}
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

	orders := slices.Strings(strings2.SplitAndTrim(order, orderSep)).Clear("")
	if fname != "" {
		errors.Fatalln(file.OpenOrCreate(fname, overwrite, func(fd *os.File) error {
			as.WriteMarkDown(fd, orders)

			return nil
		}))
	} else {
		as.WriteMarkDown(os.Stdout, orders)
	}
}

type Tag int

func (t Tag) String() string {
	switch t {
	case TAG_CATEGORY:
		return "@Category"
	case TAG_API:
		return "@API"
	case TAG_ENDAPI:
		return "@EndAPI"
	case TAG_SUBAPI:
		return "@SubAPI"
	case TAG_APIINCL:
		return "@APIIncl"
	case TAG_HEADER:
		return "@Header"
	case TAG_HEADERINCL:
		return "@HeaderIncl"
	case TAG_SUBRESP:
		return "@SubResp"
	case TAG_RESPINCL:
		return "@RespIncl"
	case TAG_REQ:
		return "@Req"
	case TAG_RESP:
		return "@Resp"
	case TAG_DATA:
		return "->"
	}

	panic("unexpected tag")
}

func (t Tag) Strlen() int {
	return len(t.String())
}

const (
	TAG_AT_CATEGORY = "@C"

	TAG_CATEGORY Tag = iota + 1
	TAG_API
	TAG_ENDAPI
	TAG_SUBAPI
	TAG_APIINCL
	TAG_HEADER
	TAG_HEADERINCL
	TAG_SUBRESP
	TAG_RESPINCL
	TAG_RESP
	TAG_REQ

	TAG_DATA
)

var tagTree tree.Trie

func init() {
	for t := TAG_CATEGORY; t < TAG_DATA; t++ {
		tagTree.AddPath(t.String(), t)
	}
}

func matchTag(line string) Tag {
	val := tagTree.PrefixMatchValue(line)
	if val == nil {
		return 0
	}

	return val.(Tag)
}

const (
	PARSE_INIT = iota
	PARSE_API
	PARSE_BODY

	PARSE_STATUS
	PARSE_HEADER
	PARSE_DATA
)

func process(path string, wg *sync.WaitGroup) {
	sectionState := PARSE_INIT
	dataState := PARSE_INIT
	var a *API
	var sec *section
	var category string = "global"

	dataStart := 0

	err := file.Filter(path, func(linum int, line []byte) error {
		line = bytes.Replace(line, tab, tabreplace, -1)
		originLine := string(line)

		if !bytes.HasPrefix(line, unsafe2.Bytes(comment)) {
			sectionState = PARSE_INIT
			return nil
		}

		line = bytes.TrimSpace(line[len(comment):])
		line = bytes.Replace(line, []byte{'\t'}, []byte("    "), -1)
		if len(line) == 0 {
			return nil
		}

		linestr := string(line)
		switch tag := matchTag(linestr); tag {
		case TAG_CATEGORY:
			category = strings.TrimSpace(linestr[tag.Strlen():])

		case TAG_API:
			a = &API{}
			sectionState = PARSE_API
			name := strings.TrimSpace(linestr[tag.Strlen():])
			names := strings2.SplitAndTrim(name, TAG_AT_CATEGORY)
			a.name = names[0]

			if len(names) > 1 {
				as.AddAPI(names[1], a)
			} else {
				as.AddAPI(category, a)
			}
		case TAG_SUBAPI:
			a = &API{}
			sectionState = PARSE_API
			a.name = strings.TrimSpace(linestr[tag.Strlen():])

			as.AddSubAPI(a.name, a)
		case TAG_APIINCL:
			if a != nil {
				a.subapis = append(a.subapis, strings2.SplitAndTrim(linestr[tag.Strlen():], ",")...)
			}
		case TAG_ENDAPI:
			sectionState = PARSE_INIT

		case TAG_HEADER:
			sec = &section{}
			sectionState = PARSE_BODY
			dataState = PARSE_HEADER

			as.AddSubHeader(strings.TrimSpace(linestr[tag.Strlen():]), sec)
		case TAG_HEADERINCL:
			if sec != nil {
				sec.subheaders = append(sec.subheaders, strings2.SplitAndTrim(linestr[tag.Strlen():], ",")...)
			}

		case TAG_RESP:
			if sectionState != PARSE_INIT {
				sectionState = PARSE_BODY
				dataState = PARSE_STATUS
				sec = &section{}
				a.resps = append(a.resps, sec)
			}
		case TAG_SUBRESP:
			name := strings.TrimSpace(linestr[tag.Strlen():])
			sectionState = PARSE_BODY
			dataState = PARSE_STATUS
			sec = &section{}
			as.AddSubResp(name, sec)
		case TAG_RESPINCL:
			if a != nil {
				a.subresps = append(a.subresps, strings2.SplitAndTrim(linestr[tag.Strlen():], ",")...)
			}

		case TAG_REQ:
			if sectionState != PARSE_INIT {
				sectionState = PARSE_BODY
				sec = &section{}
				a.req = sec
				dataState = PARSE_STATUS
			}

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
				if !strings.HasPrefix(linestr, TAG_DATA.String()) {
					sec.headers = append(sec.headers, linestr)

					break
				}

				dataStart = strings.Index(originLine, TAG_DATA.String()) + TAG_DATA.Strlen()
				dataState = PARSE_DATA
				fallthrough

			case PARSE_DATA:
				if len(originLine) >= dataStart {
					sec.datas = append(sec.datas, originLine[dataStart:])
				} else {
					reportErrorln("skipped data line: %s:%d:%s", path, linum, originLine)
				}
			}
		}

		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	wg.Done()
}
