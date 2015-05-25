package main

import (
	"fmt"
	"io"
)

func (a *API) WriteMarkdown(w io.Writer, index int, as *APIs, writeSectionName bool) {
	if index != -1 {
		fmt.Fprintf(w, "##### %d. %s\n", index+1, a.name)
	}

	for _, d := range a.desc {
		fmt.Fprintf(w, "%s  \n", d)
	}
	if a.req != nil && writeSectionName {
		writeSection(w, true, "Request", as, a.req)
	}

	writeSection(w, writeSectionName, "Response", as, a.resps...)

	for _, s := range a.subresps {
		if sa := as.subresp[s]; sa != nil {
			writeSection(w, writeSectionName && len(a.resps) == 0, "Response", as, sa)
		}
	}

	for _, s := range a.subapis {
		if sa := as.subapi[s]; sa != nil {
			sa.WriteMarkdown(w, -1, as, false)
		}
	}
}

func writeSection(w io.Writer, writeSectionName bool, name string, as *APIs, secs ...*section) {
	if len(secs) == 0 {
		return
	}
	if writeSectionName {
		fmt.Fprintf(w, "* **%s**\n", name)
	}

	for _, sec := range secs {
		fmt.Fprintf(w, "    * %s  \n", sec.headerLine)
		for _, h := range sec.subheaders {
			if sec := as.subheaders[h]; sec != nil {
				writeBodyLines(w, sec.headers)
			}
		}

		writeBodyLines(w, sec.headers)
		writeBodyLines(w, sec.datas)
	}
}

func writeBodyLines(w io.Writer, headers []string) {
	for _, h := range headers {
		fmt.Fprintf(w, "      %s  \n", h)
	}
}

func (as *APIs) WriteMarkDown(w io.Writer) {
	index := 1
	for category, apis := range as.categories {
		fmt.Fprintf(w, "#### %d. %s\n", index, category)
		index++

		for i, a := range apis {
			a.WriteMarkdown(w, i, as, true)
			fmt.Fprintln(w)
		}
	}
}
