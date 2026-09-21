// Package sharecard renders the 1200x630 OG image for a market entirely
// offline — no network font fetch, no WASM — using Go's own embedded
// gofont TTFs. This is what PRD "Share card and deep link" needs for
// WhatsApp/X/Telegram link previews.
package sharecard

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	regularFont *truetype.Font
	boldFont    *truetype.Font
)

func init() {
	var err error
	regularFont, err = truetype.Parse(goregular.TTF)
	if err != nil {
		panic(fmt.Sprintf("sharecard: parse goregular: %v", err))
	}
	boldFont, err = truetype.Parse(gobold.TTF)
	if err != nil {
		panic(fmt.Sprintf("sharecard: parse gobold: %v", err))
	}
}

func face(f *truetype.Font, size float64) font.Face {
	return truetype.NewFace(f, &truetype.Options{Size: size})
}

var (
	colorInk       = hex("#111426")
	colorPaper     = hex("#FBF8F1")
	colorHighlight = hex("#FFD23F")
	colorYes       = hex("#0B7F76")
	colorNo        = hex("#D63645")
	colorSlate     = hex("#5B6478")
	colorMist      = hex("#E7E4DA")
)

func hex(s string) color.RGBA {
	var r, g, b int
	fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

type CardInput struct {
	Question string
	Category string
	YesPrice float64 // 0..1
	NoPrice  float64 // 0..1
}

// wrapText is a character-count approximation, not real font metrics —
// good enough for a share card, revisit if headlines start clipping.
func wrapText(text string, maxCharsPerLine, maxLines int) []string {
	words := strings.Fields(text)
	var lines []string
	current := ""
	for _, w := range words {
		candidate := w
		if current != "" {
			candidate = current + " " + w
		}
		if len(candidate) > maxCharsPerLine && current != "" {
			lines = append(lines, current)
			current = w
		} else {
			current = candidate
		}
		if len(lines) == maxLines {
			break
		}
	}
	if current != "" && len(lines) < maxLines {
		lines = append(lines, current)
	}
	if len(lines) == maxLines && len(lines[maxLines-1]) > maxCharsPerLine {
		lines[maxLines-1] = lines[maxLines-1][:maxCharsPerLine-1] + "…"
	}
	return lines
}

// Render draws the card and returns PNG bytes.
func Render(in CardInput) ([]byte, error) {
	const w, h = 1200, 630
	dc := gg.NewContext(w, h)

	dc.SetColor(colorPaper)
	dc.Clear()

	dc.SetColor(colorHighlight)
	dc.DrawRectangle(0, 0, w, 10)
	dc.Fill()

	category := strings.ToUpper(in.Category)
	dc.SetFontFace(face(boldFont, 20))
	pillWidth := float64(len(category)*14 + 48)
	if pillWidth < 120 {
		pillWidth = 120
	}
	dc.SetColor(colorMist)
	dc.DrawRoundedRectangle(64, 56, pillWidth, 44, 20)
	dc.Fill()
	dc.SetColor(colorInk)
	dc.DrawStringAnchored(category, 64+pillWidth/2, 56+22, 0.5, 0.35)

	dc.SetFontFace(face(boldFont, 22))
	dc.SetColor(colorSlate)
	dc.DrawStringAnchored("Callit", w-64, 56+22, 1, 0.35)

	lines := wrapText(in.Question, 34, 4)
	lineHeight := 64.0
	dc.SetFontFace(face(boldFont, 52))
	dc.SetColor(colorInk)
	startY := h/2.0 - (float64(len(lines)-1)*lineHeight)/2 - 20
	for i, line := range lines {
		dc.DrawString(line, 64, startY+float64(i)*lineHeight)
	}

	yesPct := int(in.YesPrice*100 + 0.5)
	noPct := int(in.NoPrice*100 + 0.5)

	dc.SetColor(colorYes)
	dc.DrawRoundedRectangle(64, 486, 504, 88, 16)
	dc.Fill()
	dc.SetFontFace(face(boldFont, 34))
	dc.SetColor(colorPaper)
	dc.DrawString(fmt.Sprintf("YES %d¢", yesPct), 96, 486+54)

	dc.SetColor(colorNo)
	dc.DrawRoundedRectangle(632, 486, 504, 88, 16)
	dc.Fill()
	dc.SetColor(colorPaper)
	dc.DrawString(fmt.Sprintf("NO %d¢", noPct), 664, 486+54)

	dc.SetFontFace(face(regularFont, 22))
	dc.SetColor(colorSlate)
	dc.DrawString("Powered by Panta", 64, 600)

	return encodePNG(dc)
}
