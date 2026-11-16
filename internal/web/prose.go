package web

import "strings"

// sheet is the vim document. A blank line keeps the title apart from the body
// so :Wrap can fold each paragraph to the text width without joining them.
func sheet(title, body string) string {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if body == "" {
		return title + "\n"
	}
	return title + "\n\n" + body + "\n"
}

// readSheet joins wrapped lines back into paragraphs. The first paragraph is
// the title. Later paragraphs stay separated by a blank line.
func readSheet(text string) (string, string) {
	parts := paragraphs(text)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], "\n\n")
}

// oneParagraph joins every wrapped line into the single paragraph a short
// landing field stores.
func oneParagraph(text string) string {
	return strings.Join(paragraphs(text), " ")
}

func paragraphs(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.Trim(text, "\n")
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var out []string
	var buf []string
	flush := func() {
		if len(buf) == 0 {
			return
		}
		out = append(out, joinLines(buf))
		buf = nil
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		buf = append(buf, line)
	}
	flush()
	return out
}

func joinLines(lines []string) string {
	var words []string
	for _, line := range lines {
		words = append(words, strings.Fields(line)...)
	}
	return strings.Join(words, " ")
}
