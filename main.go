package main

import (
	"bytes"
	"code.google.com/p/go.net/html"
	"fmt"
	"github.com/voxelbrain/goptions"
	"io"
	"os"
	"path"
	"strings"
)

var verbose bool = false

func Logf(format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

func main() {
	options := struct {
		Verbose    bool   `goptions:"-v, --verbose, description='Log what bundlescript is currently doing'"`
		Htdocs     string `goptions:"--htdocs, description='htdocs directory', obligatory"`
		HtmlOutput string `goptions:"--htmlout, description='HTML output file (relative to htdocs; if \"-\" then stdout will be used)', obligatory"`
		JsOutput   string `goptions:"--jsout, description='JavaScript output file (relative to htdocs)', obligatory"`
		HtmlInput  string `goptions:"--htmlin, description='HTML input file (relative to htdocs; if \"-\" then stdin will be used)', obligatory"`
	}{}

	goptions.ParseAndFail(&options)

	verbose = options.Verbose

	outbuf := &bytes.Buffer{}

	inputStream := io.Reader(os.Stdin)

	if options.HtmlInput != "-" {
		if inf, err := os.Open(path.Join(options.Htdocs, options.HtmlInput)); err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't open input file: %v\n", err)
			os.Exit(1)
		} else {
			Logf("Reading HTML file %s.", options.HtmlInput)
			inputStream = inf
		}
	} else {
		Logf("Reading HTML from stdin.")
	}

	tokenizer := html.NewTokenizer(inputStream)

	jsSources := []string{}

	preserveClosingScriptTag := false
	insideScript := false

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}

		token := tokenizer.Token()

		if token.Type == html.StartTagToken && token.Data == "script" {
			insideScript = true
			if ignoreScriptTag(token.Attr) {
				preserveClosingScriptTag = true
				outbuf.WriteString(token.String())
			} else if src, foundSrcAttribute := extractSource(token.Attr); foundSrcAttribute {
				if src != "" {
					Logf("Found JS source file to merge: %s", src)
					jsSources = append(jsSources, src)
				}
			} else {
				outbuf.WriteString(token.String())
				preserveClosingScriptTag = true
			}
			continue
		}
		if token.Type == html.EndTagToken && token.Data == "script" {
			insideScript = false
			if preserveClosingScriptTag {
				outbuf.WriteString(token.String())
				preserveClosingScriptTag = false
			}
			continue
		}
		if token.Type == html.SelfClosingTagToken && token.Data == "script" {
			if ignoreScriptTag(token.Attr) {
				outbuf.WriteString(token.String())
			} else if src, foundSrcAttribute := extractSource(token.Attr); foundSrcAttribute {
				if src != "" {
					Logf("Found JS source file to merge: %s", src)
					jsSources = append(jsSources, src)
				}
			}
			continue
		}
		if token.Type == html.TextToken && insideScript {
			if preserveClosingScriptTag {
				outbuf.WriteString(token.String())
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
		Logf("Writing HTML output to %s.", options.HtmlOutput)
		io.Copy(f, outbuf)
	}

	Logf("Done.")
}

func ignoreScriptTag(attr []html.Attribute) bool {
	for _, a := range attr {
		if a.Namespace == "" && a.Key == "data-bundlescript" && a.Val == "ignore" {
			return true
		}
		if a.Namespace == "" && a.Key == "src" && (strings.HasPrefix(a.Val, "//") || strings.HasPrefix(a.Val, "http://") || strings.HasPrefix(a.Val, "https://")) {
			return true
		}
	}
	return false
}

func extractSource(attr []html.Attribute) (src string, foundSrc bool) {
	for _, a := range attr {
		if a.Namespace == "" && a.Key == "src" {
			return a.Val, true
		}
	}
	return "", false
}

func mergeJsSources(htdocs, jsOutput string, jsSources []string) error {
	Logf("Merging JS sources...")
	outf, err := os.OpenFile(path.Join(htdocs, jsOutput), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	Logf("Writing JS output to %s", jsOutput)
	defer outf.Close()

	for _, input := range jsSources {
		inf, err := os.Open(path.Join(htdocs, input))
		if err != nil {
			return err
		}
		defer inf.Close()
		Logf("Merging %s", input)
		io.Copy(outf, inf)
	}

	Logf("Finished with merging.")

	return nil
}
