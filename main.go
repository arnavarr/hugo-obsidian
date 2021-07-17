package main

import (
	"flag"
	"fmt"
	md "github.com/nikitavoloboev/markdown-parser"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
)

type Link struct {
	Source string
	Target string
	Text   string
}

type LinkTable = map[string][]Link
type Index struct {
	Links LinkTable
	Backlinks LinkTable
}

func trim(source, prefix, suffix string) string {
	return strings.TrimPrefix(strings.TrimSuffix(source, suffix), prefix)
}

// parse single file for links
func parse(dir, pathPrefix string) []Link {
	// read file
	bytes, err := ioutil.ReadFile(dir)
	if err != nil {
		panic(err)
	}

	// parse md
	var links []Link
	fmt.Printf("%s\n", trim(dir, pathPrefix, ".md"))
	for text, target := range md.GetAllLinks(string(bytes)) {
		fmt.Printf("  %s\n", trim(target, pathPrefix, ".md"))
		links = append(links, Link{
			Source: trim(dir, pathPrefix, ".md"),
			Target: target,
			Text: text,
		})
	}
	return links
}

// recursively walk directory and return all files with given extension
func walk(root, ext string) (res []Link) {
	println(root)
	err := filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if filepath.Ext(d.Name()) == ext {
			res = append(res, parse(s, root)...)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("parsed %d total links \n", len(res))
	return res
}

// filter out certain links (e.g. to media)
func filter(links []Link) (res []Link) {
	for _, l := range links {
		// filter external and non-md
		isInternal := strings.HasPrefix(l.Target, "/")
		isMarkdown := filepath.Ext(l.Target) == "" || filepath.Ext(l.Target) == ".md"
		if isInternal && isMarkdown {
			res = append(res, l)
		}
	}
	fmt.Printf("removed %d external and non-markdown links\n", len(links) - len(res))
	return res
}

// constructs index from links
func index(links []Link) (index Index) {
	linkMap := make(map[string][]Link)
	backlinkMap := make(map[string][]Link)
	for _, l := range links {
		bl := Link{
			Source: l.Target,
			Target: l.Source,
			Text: l.Text,
		}

		// backlink (only if internal)
		if val, ok := backlinkMap[l.Target]; ok {
			val = append(val, bl)
		} else {
			backlinkMap[l.Target] = []Link{bl}
		}

		// regular link
		if val, ok := linkMap[l.Source]; ok {
			val = append(val, l)
		} else {
			linkMap[l.Target] = []Link{l}
		}
	}
	index.Links = linkMap
	index.Backlinks = backlinkMap
	return index
}

func write(links []Link, out string) error {
	index := index(links)
	resStruct := struct{
		Index Index
		Links []Link
	}{
		Index: index,
		Links: links,
	}
	marshalledIndex, mErr := yaml.Marshal(&resStruct)
	if mErr != nil {
		return mErr
	}

	writeErr := ioutil.WriteFile(path.Join(out, "linkIndex.yaml"), marshalledIndex, 0644)
	if writeErr != nil {
		return writeErr
	}
	return nil
}

func main() {
	in := flag.String("input", ".", "Input Directory")
	out := flag.String("output", ".", "Output Directory")
	flag.Parse()
	l := walk(*in, ".md")
	f := filter(l)
	err := write(f, *out)
	if err != nil {
		panic(err)
	}
}
