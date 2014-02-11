package main

import (
	"bytes"
	"code.google.com/p/go.net/html"
	"fmt"
	"github.com/voxelbrain/goptions"
	"io"
	"os"
	"path"
)

func main() {
	options := struct {
		Htdocs     string `goptions:"--htdocs, description='htdocs directory', obligatory"`
		HtmlOutput string `goptions:"--htmlout, description='HTML output file (relative to htdocs; if \"-\" then stdout will be used)', obligatory"`
		JsOutput   string `goptions:"--jsout, description='JavaScript output file (relative to htdocs)', obligatory"`
		HtmlInput  string `goptions:"--htmlin, description='HTML input file (relative to htdocs; if \"-\" then stdin will be used)', obligatory"`
	}{}

	goptions.ParseAndFail(&options)

	outbuf := &bytes.Buffer{}

	inputStream := io.Reader(os.Stdin)

	if options.HtmlInput != "-" {
		if inf, err := os.Open(path.Join(options.Htdocs, options.HtmlInput)); err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't open input file: %v\n", err)
			os.Exit(1)
		} else {
			inputStream = inf
		}
	}

	tokenizer := html.NewTokenizer(inputStream)

	jsSources := []string{}

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}

		token := tokenizer.Token()

		if token.Type == html.StartTagToken && token.Data == "script" {
			if src := extractSource(token.Attr); src != "" {
				jsSources = append(jsSources, src)
			}
			continue
		}
		if token.Type == html.EndTagToken && token.Data == "script" {
			// skip closing tag
			continue
		}
		if token.Type == html.SelfClosingTagToken && token.Data == "script" {
			if src := extractSource(token.Attr); src != "" {
				jsSources = append(jsSources, src)
			}
			continue
		}
		if token.Type == html.EndTagToken && token.Data == "html" {
			err := mergeJsSources(options.Htdocs, options.JsOutput, jsSources)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't merge JavaScript sources: %v", err)
				os.Exit(1)
			}

			scriptStartToken := &html.Token{
				Type: html.StartTagToken,
				Data: "script",
				Attr: []html.Attribute{html.Attribute{Key: "src", Val: options.JsOutput}},
			}
			outbuf.WriteString(scriptStartToken.String())

			scriptEndToken := &html.Token{
				Type: html.EndTagToken,
				Data: "script",
			}
			outbuf.WriteString(scriptEndToken.String())

			linebreakToken := &html.Token{
				Type: html.TextToken,
				Data: "\n",
			}
			outbuf.WriteString(linebreakToken.String())
		}
		outbuf.WriteString(token.String())
	}

	if options.HtmlOutput == "-" {
		os.Stdout.WriteString(outbuf.String())
	} else {
		f, err := os.OpenFile(path.Join(options.Htdocs, options.HtmlOutput), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't open HTML output file: %v", err)
			os.Exit(1)
		}
		defer f.Close()
		io.Copy(f, outbuf)
	}
}

func extractSource(attr []html.Attribute) string {
	for _, a := range attr {
		if a.Namespace == "" && a.Key == "src" {
			return a.Val
		}
	}
	return ""
}

func mergeJsSources(htdocs, jsOutput string, jsSources []string) error {
	outf, err := os.OpenFile(path.Join(htdocs, jsOutput), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer outf.Close()

	for _, input := range jsSources {
		inf, err := os.Open(path.Join(htdocs, input))
		if err != nil {
			return err
		}
		defer inf.Close()

		io.Copy(outf, inf)
	}

	return nil
}