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

func getArgInt(node *bbcode.BBCodeNode, key string, min, max int) (int, bool) {
	val, ok := node.GetOpeningTag().Args[key]
	if !ok || val == "" {
		return 0, false
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < min || n > max {
		return 0, false
	}
	return n, true
}

var bbCompiler = GetBBCompiler()

func ParseBBCode(text string) string {
	return bbCompiler.Compile(text)
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

	compiler.SetTag("grid", func(node *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		out := bbcode.NewHTMLTag("")
		out.Name = "div"

		style := "display: grid;"
		if cols, ok := getArgInt(node, "columns", 1, 12); ok {
			style += fmt.Sprintf(" grid-template-columns: repeat(%d, 1fr);", cols)
		}
		if rows, ok := getArgInt(node, "rows", 1, 12); ok {
			style += fmt.Sprintf(" grid-template-rows: repeat(%d, 1fr);", rows)
		}
		if gap, ok := getArg(node, "gap"); ok {
			style += fmt.Sprintf(" gap: %s;", gap)
		}
		if width, ok := getArg(node, "width"); ok {
			style += fmt.Sprintf(" width: %s;", width)
		}

		out.Attrs["style"] = style
		return out, true
	})

	compiler.SetTag("grid-item", func(node *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		out := bbcode.NewHTMLTag("")
		out.Name = "div"

		var style string
		if col, ok := getArgInt(node, "col", 1, 12); ok {
			style += fmt.Sprintf("grid-column-start: %d;", col)
		}
		if row, ok := getArgInt(node, "row", 1, 12); ok {
			style += fmt.Sprintf(" grid-row-start: %d;", row)
		}
		if colSpan, ok := getArgInt(node, "col-span", 1, 12); ok {
			style += fmt.Sprintf(" grid-column-end: span %d;", colSpan)
		}
		if rowSpan, ok := getArgInt(node, "row-span", 1, 12); ok {
			style += fmt.Sprintf(" grid-row-end: span %d;", rowSpan)
		}
		if style != "" {
			out.Attrs["style"] = style
		}

		return out, true
	})

	compiler.SetTag("insert-post", func(node *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		out := bbcode.NewHTMLTag("")
		out.Name = "div"

		value := node.GetOpeningTag().Value
		id, err := strconv.Atoi(value)
		if err != nil || id <= 0 {
			return out, false
		}

		out.Attrs["data-insert"] = strconv.Itoa(id)

		js := fmt.Sprintf(
			`(function(){var el=document.currentScript.parentElement;`+
				`fetch('/post/%d').then(function(r){return r.json();}).then(function(d){`+
				`el.innerHTML=d.content_html;`+
				`});})();`,
			id,
		)
		script := bbcode.NewHTMLTag(js)
		script.Name = "script"
		out.AppendChild(script)

		return out, false
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

	compiler.SetTag("spoiler", func(node *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		out := bbcode.NewHTMLTag("")
		out.Name = "div"
		out.Attrs["class"] = "spoiler-box"

		title := node.GetOpeningTag().Value
		if title == "" {
			title = "Spoiler"
		}

		header := bbcode.NewHTMLTag(html.EscapeString(title))
		header.Name = "div"
		header.Attrs["onclick"] = "var c=this.nextElementSibling;c.style.maxHeight=c.style.maxHeight?'':c.scrollHeight+'px';"
		out.AppendChild(header)

		content := bbcode.NewHTMLTag("")
		content.Name = "div"
		content.Attrs["style"] = "max-height: 0; overflow: hidden; transition: max-height 0.3s ease;"
		out.AppendChild(content)

		return content, true
	})

	compiler.SetTag("img", func(node *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		out := bbcode.NewHTMLTag("")
		out.Name = "img"

		value := node.GetOpeningTag().Value
		if value == "" {
			out.Attrs["src"] = bbcode.ValidURL(bbcode.CompileText(node))
		} else {
			out.Attrs["src"] = bbcode.ValidURL(value)
			text := bbcode.CompileText(node)
			if len(text) > 0 {
				out.Attrs["alt"] = text
				out.Attrs["title"] = text
			}
		}
		out.Attrs["loading"] = "lazy"
		out.Attrs["referrerpolicy"] = "no-referrer"

		return out, false
	})

	return compiler
}
