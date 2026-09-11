package ezsign

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"math"
)

var palette = [4][3]float64{{0, 0, 0}, {255, 255, 255}, {255, 192, 0}, {255, 0, 0}}

func renderNative(ctx context.Context, img image.Image, options WriteOptions) (preview []byte, frame []byte, err error) {
	bounds := img.Bounds()
	ratio := math.Min(400/float64(bounds.Dx()), 300/float64(bounds.Dy()))
	w := max(1, int(math.Round(float64(bounds.Dx())*ratio)))
	h := max(1, int(math.Round(float64(bounds.Dy())*ratio)))
	left, top := (400-w)/2, (300-h)/2
	rgb := make([][3]float64, 400*300)
	for i := range rgb {
		rgb[i] = palette[1]
	}
	sample := func(x, y int) [3]float64 {
		r, g, b, a := img.At(bounds.Min.X+max(0, min(x, bounds.Dx()-1)), bounds.Min.Y+max(0, min(y, bounds.Dy()-1))).RGBA()
		return [3]float64{float64(r+65535-a) / 257, float64(g+65535-a) / 257, float64(b+65535-a) / 257}
	}
	for y := 0; y < h; y++ {
		if err = ctx.Err(); err != nil {
			return nil, nil, err
		}
		sy := (float64(y)+0.5)*float64(bounds.Dy())/float64(h) - 0.5
		y0 := int(math.Floor(sy))
		fy := sy - float64(y0)
		for x := 0; x < w; x++ {
			sx := (float64(x)+0.5)*float64(bounds.Dx())/float64(w) - 0.5
			x0 := int(math.Floor(sx))
			fx := sx - float64(x0)
			a, b, c, d := sample(x0, y0), sample(x0+1, y0), sample(x0, y0+1), sample(x0+1, y0+1)
			px, py := left+x, top+y
			// The installed display orientation is 180 degrees from wire coordinates.
			if options.Rotation == 0 {
				px, py = 399-px, 299-py
			}
			for k := 0; k < 3; k++ {
				rgb[py*400+px][k] = (a[k]*(1-fx)+b[k]*fx)*(1-fy) + (c[k]*(1-fx)+d[k]*fx)*fy
			}
		}
	}
	out := image.NewNRGBA(image.Rect(0, 0, 400, 300))
	frame = make([]byte, 30000)
	for y := 0; y < 300; y++ {
		if err = ctx.Err(); err != nil {
			return nil, nil, err
		}
		for x := 0; x < 400; x++ {
			i := y*400 + x
			p := rgb[i]
			for k := range p {
				p[k] = math.Max(0, math.Min(255, p[k]))
			}
			best := 0
			distance := math.Inf(1)
			for c, q := range palette {
				dist := 0.0
				for k := 0; k < 3; k++ {
					dist += (p[k] - q[k]) * (p[k] - q[k])
				}
				if dist < distance {
					distance = dist
					best = c
				}
			}
			q := palette[best]
			out.SetNRGBA(x, y, color.NRGBA{byte(q[0]), byte(q[1]), byte(q[2]), 255})
			frame[(299-y)*100+x/4] |= byte(best) << uint(6-2*(x%4))
			if !options.NoDither {
				for _, neighbor := range []struct {
					dx, dy int
					weight float64
				}{{1, 0, 7.0 / 16}, {-1, 1, 3.0 / 16}, {0, 1, 5.0 / 16}, {1, 1, 1.0 / 16}} {
					nx, ny := x+neighbor.dx, y+neighbor.dy
					if nx >= 0 && nx < 400 && ny < 300 {
						for k := 0; k < 3; k++ {
							rgb[ny*400+nx][k] += (p[k] - q[k]) * neighbor.weight
						}
					}
				}
			}
		}
	}
	var buf bytes.Buffer
	err = png.Encode(&buf, out)
	return buf.Bytes(), frame, err
}
