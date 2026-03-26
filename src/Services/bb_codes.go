package Services

import (
	"fmt"
	"html"
	"strconv"

	"github.com/frustra/bbcode"
)

func getArg(node *bbcode.BBCodeNode, key string) (string, bool) {
	val, ok := node.GetOpeningTag().Args[key]
	if !ok || val == "" {
		return "", false
	}
	return html.EscapeString(val), true
}

func GetBBCompiler() bbcode.Compiler {
	compiler := bbcode.NewCompiler(true, true)

	compiler.SetTag("center", func(node *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		out := bbcode.NewHTMLTag("")
		out.Name = "div"
		out.Attrs["style"] = "text-align: center;"
		return out, true
	})

	compiler.SetTag("left", func(node *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		out := bbcode.NewHTMLTag("")
		out.Name = "div"
		out.Attrs["style"] = "text-align: left;"
		return out, true
	})

	compiler.SetTag("right", func(node *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		out := bbcode.NewHTMLTag("")
		out.Name = "div"
		out.Attrs["style"] = "text-align: right;"
		return out, true
	})

	compiler.SetTag("div", func(node *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		out := bbcode.NewHTMLTag("")
		out.Name = "div"

		style := "display: block; "
		if border, ok := getArg(node, "border"); ok {
			style += fmt.Sprintf("border: %s;", border)
		}
		if width, ok := getArg(node, "width"); ok {
			style += fmt.Sprintf(" width: %s;", width)
		}
		if height, ok := getArg(node, "height"); ok {
			style += fmt.Sprintf(" height: %s;", height)
		}
		if style != "" {
			out.Attrs["style"] = style
		}

		return out, true
	})

	compiler.SetTag("font-size", func(node *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		out := bbcode.NewHTMLTag("")
		out.Name = "span"

		value := node.GetOpeningTag().Value
		if size, err := strconv.Atoi(value); err == nil && size >= 0 && size <= 32 {
			out.Attrs["style"] = fmt.Sprintf("font-size: %dpx;", size)
		}

		return out, true
	})

	compiler.SetTag("font-family", func(node *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		out := bbcode.NewHTMLTag("")
		out.Name = "span"

		validFonts := map[string]bool{
			// Generic families
			"serif": true, "sans-serif": true, "monospace": true,
			"cursive": true, "fantasy": true, "system-ui": true,
			// Web-safe fonts
			"arial": true, "helvetica": true, "verdana": true,
			"tahoma": true, "trebuchet ms": true, "gill sans": true,
			"times new roman": true, "georgia": true, "garamond": true, "palatino": true,
			"courier new": true, "lucida console": true, "monaco": true,
			"comic sans ms": true, "impact": true, "arial black": true,
		}

		value := node.GetOpeningTag().Value
		if validFonts[value] {
			out.Attrs["style"] = fmt.Sprintf("font-family: %s;", value)
		}

		return out, true
	})

	compiler.SetTag("float", func(node *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		out := bbcode.NewHTMLTag("")
		out.Name = "div"

		value := node.GetOpeningTag().Value
		if value == "right" || value == "left" {
			out.Attrs["style"] = fmt.Sprintf("float: %s;", value)
		}

		return out, true
	})

	return compiler
}
