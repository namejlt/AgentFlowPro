package export

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"time"
)

// BuildDOCX creates a minimal OOXML document containing paragraphs from plain text / markdown lines.
func BuildDOCX(title, body string) ([]byte, error) {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	add := func(name, data string) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(data))
		return err
	}
	if err := add("[Content_Types].xml", contentTypes()); err != nil {
		return nil, err
	}
	if err := add("_rels/.rels", relsRoot()); err != nil {
		return nil, err
	}
	if err := add("word/_rels/document.xml.rels", relsDoc()); err != nil {
		return nil, err
	}
	if err := add("word/document.xml", wordDocument(title, body)); err != nil {
		return nil, err
	}
	if err := add("docProps/core.xml", corePropsSimple(title)); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func esc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func wordDocument(title, body string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	b.WriteString(`<w:body>`)
	b.WriteString(`<w:p><w:r><w:t>` + esc(title) + `</w:t></w:r></w:p>`)
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		b.WriteString(`<w:p><w:r><w:t>` + esc(line) + `</w:t></w:r></w:p>`)
	}
	b.WriteString(`</w:body></w:document>`)
	return b.String()
}

func corePropsSimple(title string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" ` +
		`xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
		`<dc:title>` + esc(title) + `</dc:title><dcterms:created xsi:type="dcterms:W3CDTF">` +
		timeNowRFC3339() + `</dcterms:created></cp:coreProperties>`
}

func timeNowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func contentTypes() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
</Types>`
}

func relsRoot() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
</Relationships>`
}

func relsDoc() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`
}

func DOCXFilename(title string) string {
	return fmt.Sprintf("%s.docx", strings.ReplaceAll(strings.TrimSpace(title), "/", "-"))
}
