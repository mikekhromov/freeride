package media

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	cardWidth  = 1280
	cardHeight = 720
)

func RenderTitleCard(title string) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, cardWidth, cardHeight))
	if err := drawBackground(img); err != nil {
		drawFallbackBackground(img)
	}

	// Dark overlay for text readability.
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 0, G: 0, B: 0, A: 95}), image.Point{}, draw.Over)

	addCenteredLabel(img, title)

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func drawFallbackBackground(dst *image.RGBA) {
	top := color.RGBA{R: 9, G: 18, B: 34, A: 255}
	bottom := color.RGBA{R: 22, G: 38, B: 68, A: 255}
	height := dst.Bounds().Dy()
	width := dst.Bounds().Dx()
	if height <= 1 {
		draw.Draw(dst, dst.Bounds(), image.NewUniform(top), image.Point{}, draw.Src)
		return
	}
	for y := 0; y < height; y++ {
		t := float64(y) / float64(height-1)
		r := uint8(float64(top.R)*(1-t) + float64(bottom.R)*t)
		g := uint8(float64(top.G)*(1-t) + float64(bottom.G)*t)
		b := uint8(float64(top.B)*(1-t) + float64(bottom.B)*t)
		line := image.NewUniform(color.RGBA{R: r, G: g, B: b, A: 255})
		draw.Draw(dst, image.Rect(0, y, width, y+1), line, image.Point{}, draw.Src)
	}
}

func drawBackground(dst *image.RGBA) error {
	src, err := loadBackgroundImage()
	if err != nil {
		return err
	}
	srcBounds := src.Bounds()
	if srcBounds.Dx() == 0 || srcBounds.Dy() == 0 {
		return fmt.Errorf("background image has invalid dimensions")
	}
	// Cover resize: fill 1280x720 without empty borders.
	scale := maxFloat(
		float64(cardWidth)/float64(srcBounds.Dx()),
		float64(cardHeight)/float64(srcBounds.Dy()),
	)
	scaledW := int(float64(srcBounds.Dx()) * scale)
	scaledH := int(float64(srcBounds.Dy()) * scale)
	if scaledW < cardWidth {
		scaledW = cardWidth
	}
	if scaledH < cardHeight {
		scaledH = cardHeight
	}
	tmp := image.NewRGBA(image.Rect(0, 0, scaledW, scaledH))
	xdraw.CatmullRom.Scale(tmp, tmp.Bounds(), src, srcBounds, draw.Over, nil)
	offX := (scaledW - cardWidth) / 2
	offY := (scaledH - cardHeight) / 2
	draw.Draw(dst, dst.Bounds(), tmp, image.Point{X: offX, Y: offY}, draw.Src)
	return nil
}

func loadBackgroundImage() (image.Image, error) {
	for _, p := range backgroundPathCandidates() {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		img, _, decErr := image.Decode(f)
		_ = f.Close()
		if decErr == nil {
			return img, nil
		}
	}
	return nil, fmt.Errorf("background image not found")
}

func backgroundPathCandidates() []string {
	var out []string
	if p := os.Getenv("BOT_CARD_BG_PATH"); p != "" {
		out = append(out, p)
	}
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		out = append(out, filepath.Join(exeDir, "bg.jpg"))
	}
	if wdPath, err := filepath.Abs("bg.jpg"); err == nil {
		out = append(out, wdPath)
	}
	out = append(out,
		"bg.jpg",
		filepath.Join(".", "bg.jpg"),
		filepath.Join("arengate-landing", "public", "bg.jpg"),
		filepath.Join("assets", "bg.jpg"),
	)
	return out
}

func addCenteredLabel(img *image.RGBA, text string) {
	face, err := buildTitleFace(text)
	if err != nil {
		return
	}
	defer face.Close()

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.White),
		Face: face,
	}
	advance := d.MeasureString(text)
	textW := advance.Ceil()
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	descent := metrics.Descent.Ceil()
	textH := ascent + descent

	x := (cardWidth - textW) / 2
	y := (cardHeight-textH)/2 + ascent
	d.Dot = fixed.P(x, y)
	d.DrawString(text)
}

func buildTitleFace(text string) (font.Face, error) {
	ft, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return nil, err
	}
	// Target text width ~= 40% of card width.
	target := int(float64(cardWidth) * 0.40)
	best := 72.0
	for size := 220.0; size >= 48.0; size -= 2 {
		face, e := opentype.NewFace(ft, &opentype.FaceOptions{
			Size:    size,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if e != nil {
			continue
		}
		w := font.MeasureString(face, text).Ceil()
		_ = face.Close()
		if w <= target {
			best = size
			break
		}
	}
	return opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    best,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func LoadStaticCard(name string) ([]byte, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, fmt.Errorf("empty card name")
	}
	for _, p := range staticCardPathCandidates(trimmed) {
		b, err := os.ReadFile(p)
		if err == nil && len(b) > 0 {
			return b, nil
		}
	}
	return nil, fmt.Errorf("static card not found: %s", trimmed)
}

func staticCardPathCandidates(name string) []string {
	fileNames := []string{name + ".png", name + ".jpg", name + ".jpeg"}
	var out []string
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		for _, fn := range fileNames {
			out = append(out, filepath.Join(exeDir, "img", fn))
		}
	}
	if wd, err := os.Getwd(); err == nil {
		for _, fn := range fileNames {
			out = append(out, filepath.Join(wd, "img", fn))
		}
	}
	for _, fn := range fileNames {
		out = append(out, filepath.Join("img", fn))
		out = append(out, filepath.Join("vpn-bot", "img", fn))
	}
	return out
}
