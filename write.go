package main

import (
	"fmt"
	"io"

	"github.com/cosiner/gohper/terminal/color"
)

func (a *API) WriteMarkdown(w io.Writer, index int, as *APIs, writeSectionName bool) {
	if index != -1 {
		printfln(w, "##### %d. %s", index+1, a.name)
	}

	for _, d := range a.desc {
		printfln(w, "%s  ", d)
	}
	if a.req != nil && writeSectionName {
		as.writeSection(w, a.name, "Request", true, a.req)
	}

	as.writeSection(w, a.name, "Response", writeSectionName, a.resps...)

	for _, s := range a.subresps {
		if resp := as.subresp[s]; resp != nil {
			as.writeSection(w, a.name, "Response", writeSectionName && len(a.resps) == 0, resp)
		} else {
			reportErrorln(`sub-response "%s" for api "%s" not found.`, s, a.name)
		}
	}

	for _, s := range a.subapis {
		if api := as.subapi[s]; api != nil {
			api.WriteMarkdown(w, -1, as, false)
		} else {
			reportErrorln(`sub-api "%s" for api "%s" not found.`, s, a.name)
		}
	}
}

func (as *APIs) writeSection(w io.Writer, apiname, secname string, writeSectionName bool, secs ...*section) {
	if len(secs) == 0 {
		return
	}
	if writeSectionName {
		printfln(w, "* **%s**", secname)
	}

	for _, sec := range secs {
		printfln(w, "    * %s  ", sec.headerLine)
		for _, h := range sec.subheaders {
			if sec := as.subheaders[h]; sec != nil {
				writeBodyLines(w, sec.headers)
			} else {
				reportErrorln(`sub-header "%s" for api "%s" not found.`, h, apiname)
			}
		}

		writeBodyLines(w, sec.headers)
		writeBodyLines(w, sec.datas)
	}
}

func writeBodyLines(w io.Writer, headers []string) {
	for _, h := range headers {
		printfln(w, "      %s  ", h)
	}
}

func (as *APIs) WriteMarkDown(w io.Writer, orders []string) {
	index := 0
	for _, category := range orders {
		if apis := as.categories[category]; apis != nil {
			delete(as.categories, category)
			index++
			as.writeCategory(w, index, category, apis)
		}
	}

	for category, apis := range as.categories {
		index++
		as.writeCategory(w, index, category, apis)
	}
}

func (as *APIs) writeCategory(w io.Writer, index int, category string, apis []*API) {
	printfln(w, "#### %d. %s", index, category)

	for i, a := range apis {
		a.WriteMarkdown(w, i, as, true)
		fmt.Fprintln(w)
	}
}

func reportErrorln(format string, args ...interface{}) {
	color.Red.Errorf(format+"\n", args...)
}

func printfln(w io.Writer, format string, args ...interface{}) {
	fmt.Fprintf(w, format+"\n", args...)
}
