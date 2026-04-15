// Package workspace — chunk.go implements paragraph-aware text chunking for
// the specd knowledge base. It supports markdown, plain text, HTML (via text
// extraction), and PDF (via page-aware text extraction).
package workspace

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
)

const (
	chunkTargetMin = 500
	chunkTargetMax = 1000
	chunkHardCap   = 2000
	maxChunksPerDoc = 10000
)

// Chunk represents a single chunk of text extracted from a KB document.
type Chunk struct {
	Position  int    // 0-based index within the document
	Text      string // chunk text content
	CharStart int    // character offset from start of extracted text
	CharEnd   int    // character offset end (exclusive)
	Page      *int   // page number (PDF only, 1-based)
}

// chunkParagraphs splits text into chunks respecting paragraph boundaries.
// Targets chunkTargetMin–chunkTargetMax characters, hard cap at chunkHardCap.
func chunkParagraphs(text string) []Chunk {
	paragraphs := splitParagraphs(text)
	if len(paragraphs) == 0 {
		return nil
	}

	var chunks []Chunk
	var buf strings.Builder
	charStart := 0
	bufStart := 0

	for _, para := range paragraphs {
		paraLen := utf8.RuneCountInString(para)

		// If a single paragraph exceeds hard cap, split it mid-sentence.
		if paraLen > chunkHardCap {
			// Flush current buffer first.
			if buf.Len() > 0 {
				chunks = append(chunks, Chunk{
					Position:  len(chunks),
					Text:      buf.String(),
					CharStart: bufStart,
					CharEnd:   charStart,
				})
				buf.Reset()
			}

			subChunks := splitLongParagraph(para, charStart, len(chunks))
			chunks = append(chunks, subChunks...)
			charStart += paraLen
			bufStart = charStart
			continue
		}

		bufRuneLen := utf8.RuneCountInString(buf.String())

		// If adding this paragraph would exceed target max, flush first.
		if bufRuneLen > 0 && bufRuneLen+paraLen > chunkTargetMax {
			chunks = append(chunks, Chunk{
				Position:  len(chunks),
				Text:      buf.String(),
				CharStart: bufStart,
				CharEnd:   charStart,
			})
			buf.Reset()
			bufStart = charStart
		}

		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(para)
		charStart += paraLen

		// If buffer is in the target range, flush.
		bufRuneLen = utf8.RuneCountInString(buf.String())
		if bufRuneLen >= chunkTargetMin {
			chunks = append(chunks, Chunk{
				Position:  len(chunks),
				Text:      buf.String(),
				CharStart: bufStart,
				CharEnd:   charStart,
			})
			buf.Reset()
			bufStart = charStart
		}
	}

	// Flush remaining.
	if buf.Len() > 0 {
		chunks = append(chunks, Chunk{
			Position:  len(chunks),
			Text:      buf.String(),
			CharStart: bufStart,
			CharEnd:   charStart,
		})
	}

	if len(chunks) > maxChunksPerDoc {
		return chunks[:maxChunksPerDoc]
	}
	return chunks
}

// splitParagraphs splits text on double newlines, trimming whitespace.
func splitParagraphs(text string) []string {
	raw := strings.Split(text, "\n\n")
	var result []string
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// splitLongParagraph breaks a paragraph that exceeds the hard cap into
// sentence-boundary chunks. Falls back to word boundaries if needed.
func splitLongParagraph(para string, globalOffset int, startPos int) []Chunk {
	var chunks []Chunk
	remaining := para
	localOffset := 0

	for len(remaining) > 0 {
		runeLen := utf8.RuneCountInString(remaining)
		if runeLen <= chunkTargetMax {
			chunks = append(chunks, Chunk{
				Position:  startPos + len(chunks),
				Text:      remaining,
				CharStart: globalOffset + localOffset,
				CharEnd:   globalOffset + localOffset + runeLen,
			})
			break
		}

		// Try to find a sentence boundary within target range.
		cutPoint := findSentenceBoundary(remaining, chunkTargetMax)
		piece := remaining[:cutPoint]
		pieceRuneLen := utf8.RuneCountInString(piece)

		chunks = append(chunks, Chunk{
			Position:  startPos + len(chunks),
			Text:      strings.TrimSpace(piece),
			CharStart: globalOffset + localOffset,
			CharEnd:   globalOffset + localOffset + pieceRuneLen,
		})

		localOffset += cutPoint
		remaining = strings.TrimSpace(remaining[cutPoint:])
	}

	return chunks
}

// findSentenceBoundary finds the best cut point within maxRunes characters,
// preferring sentence endings (. ! ?), falling back to word boundaries.
func findSentenceBoundary(text string, maxRunes int) int {
	// Convert rune limit to byte offset.
	byteLimit := 0
	runeCount := 0
	for i := range text {
		if runeCount >= maxRunes {
			byteLimit = i
			break
		}
		runeCount++
	}
	if byteLimit == 0 {
		byteLimit = len(text)
	}

	window := text[:byteLimit]

	// Look for sentence-ending punctuation followed by space.
	lastSentence := -1
	for i := len(window) - 1; i > 0; i-- {
		if (window[i] == '.' || window[i] == '!' || window[i] == '?') &&
			i+1 < len(window) && window[i+1] == ' ' {
			lastSentence = i + 1
			break
		}
	}
	if lastSentence > len(window)/4 {
		return lastSentence
	}

	// Fall back to last space.
	lastSpace := strings.LastIndex(window, " ")
	if lastSpace > len(window)/4 {
		return lastSpace + 1
	}

	return byteLimit
}

// ChunkMarkdown chunks a markdown document's text content.
func ChunkMarkdown(content string) []Chunk {
	return chunkParagraphs(content)
}

// ChunkPlainText chunks a plain text document.
func ChunkPlainText(content string) []Chunk {
	return chunkParagraphs(content)
}

// ExtractHTMLText parses HTML and extracts visible text content.
func ExtractHTMLText(rawHTML string) (string, error) {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return "", fmt.Errorf("parse HTML: %w", err)
	}

	var buf strings.Builder
	extractText(doc, &buf)
	return strings.TrimSpace(buf.String()), nil
}

// extractText recursively extracts text from HTML nodes, skipping
// script, style, and other non-visible elements.
func extractText(n *html.Node, buf *strings.Builder) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "script", "style", "noscript", "template":
			return
		}
	}

	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			if buf.Len() > 0 {
				buf.WriteString(" ")
			}
			buf.WriteString(text)
		}
	}

	// Add paragraph breaks after block elements.
	isBlock := false
	if n.Type == html.ElementNode {
		switch n.Data {
		case "p", "div", "section", "article", "h1", "h2", "h3", "h4", "h5", "h6",
			"li", "blockquote", "pre", "tr", "br", "hr":
			isBlock = true
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractText(c, buf)
	}

	if isBlock && buf.Len() > 0 {
		buf.WriteString("\n\n")
	}
}

// SanitizeHTML runs bluemonday UGC policy on raw HTML, producing a safe
// sidecar suitable for browser rendering.
func SanitizeHTML(rawHTML string) string {
	p := bluemonday.UGCPolicy()
	return p.Sanitize(rawHTML)
}

// ChunkHTML extracts text from HTML and chunks it.
func ChunkHTML(rawHTML string) ([]Chunk, error) {
	text, err := ExtractHTMLText(rawHTML)
	if err != nil {
		return nil, err
	}
	return chunkParagraphs(text), nil
}

// PDFPage holds extracted text for a single page.
type PDFPage struct {
	Number int    // 1-based page number
	Text   string // extracted text content
}

// ExtractPDFPages extracts text from a PDF file, returning per-page text.
func ExtractPDFPages(pdfPath string) ([]PDFPage, error) {
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("open PDF: %w", err)
	}
	defer f.Close()

	totalPages := r.NumPage()
	if totalPages == 0 {
		return nil, fmt.Errorf("PDF has no pages")
	}

	var pages []PDFPage
	for i := 1; i <= totalPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := extractPageText(page)
		if err != nil {
			// Skip pages that fail to extract; log and continue.
			continue
		}

		text = strings.TrimSpace(text)
		if text != "" {
			pages = append(pages, PDFPage{Number: i, Text: text})
		}
	}

	return pages, nil
}

// extractPageText extracts text content from a single PDF page.
func extractPageText(page pdf.Page) (string, error) {
	rows, err := page.GetTextByRow()
	if err != nil {
		// Fallback: try plain text content.
		content := page.Content()
		var buf strings.Builder
		for _, t := range content.Text {
			buf.WriteString(t.S)
		}
		return buf.String(), nil
	}

	var buf strings.Builder
	for _, row := range rows {
		for _, word := range row.Content {
			if buf.Len() > 0 {
				buf.WriteString(" ")
			}
			buf.WriteString(word.S)
		}
		buf.WriteString("\n")
	}
	return buf.String(), err
}

// ChunkPDF extracts text from a PDF and chunks it with page awareness.
// Chunk boundaries align to paragraph breaks within a page; a page boundary
// is never broken mid-chunk.
func ChunkPDF(pdfPath string) ([]Chunk, int, error) {
	pages, err := ExtractPDFPages(pdfPath)
	if err != nil {
		return nil, 0, err
	}

	pageCount := 0
	if len(pages) > 0 {
		pageCount = pages[len(pages)-1].Number
	}

	var chunks []Chunk
	globalOffset := 0

	for _, page := range pages {
		pageNum := page.Number
		paragraphs := splitParagraphs(page.Text)

		var buf strings.Builder
		bufStart := globalOffset

		for _, para := range paragraphs {
			paraLen := utf8.RuneCountInString(para)

			bufRuneLen := utf8.RuneCountInString(buf.String())
			if bufRuneLen > 0 && bufRuneLen+paraLen > chunkTargetMax {
				pg := pageNum
				chunks = append(chunks, Chunk{
					Position:  len(chunks),
					Text:      buf.String(),
					CharStart: bufStart,
					CharEnd:   globalOffset,
					Page:      &pg,
				})
				buf.Reset()
				bufStart = globalOffset
			}

			if buf.Len() > 0 {
				buf.WriteString("\n\n")
			}
			buf.WriteString(para)
			globalOffset += paraLen

			bufRuneLen = utf8.RuneCountInString(buf.String())
			if bufRuneLen >= chunkTargetMin {
				pg := pageNum
				chunks = append(chunks, Chunk{
					Position:  len(chunks),
					Text:      buf.String(),
					CharStart: bufStart,
					CharEnd:   globalOffset,
					Page:      &pg,
				})
				buf.Reset()
				bufStart = globalOffset
			}
		}

		// Flush remaining text for this page (never carry across pages).
		if buf.Len() > 0 {
			pg := pageNum
			chunks = append(chunks, Chunk{
				Position:  len(chunks),
				Text:      buf.String(),
				CharStart: bufStart,
				CharEnd:   globalOffset,
				Page:      &pg,
			})
			buf.Reset()
		}
	}

	if len(chunks) > maxChunksPerDoc {
		return nil, 0, fmt.Errorf("PDF exceeds maximum chunk count (%d)", maxChunksPerDoc)
	}

	return chunks, pageCount, nil
}

// Ensure pdf.Open's file handle is properly typed.
var _ io.Closer = (*os.File)(nil)
var _ = bytes.NewReader // suppress unused import if needed
