package media

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	cardWidth  = 1280
	cardHeight = 720
)

func RenderTitleCard(title string) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, cardWidth, cardHeight))
	bg := image.NewUniform(color.RGBA{R: 18, G: 24, B: 38, A: 255})
	draw.Draw(img, img.Bounds(), bg, image.Point{}, draw.Src)

	// Accent strip to make the card more recognizable.
	draw.Draw(img, image.Rect(0, cardHeight-24, cardWidth, cardHeight), image.NewUniform(color.RGBA{R: 42, G: 130, B: 228, A: 255}), image.Point{}, draw.Src)

	addLabel(img, title, 72, 360)

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func addLabel(img *image.RGBA, text string, x, y int) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.White),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}
