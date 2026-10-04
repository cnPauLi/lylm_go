package captcha

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"math/rand"
)

// 图片尺寸
const (
	width  = 100
	height = 38
)

// digitFont 5x7 点阵字体（0-9），每行取低 5 位，最高位为最左侧像素
var digitFont = [10][7]uint8{
	{0b01110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110}, // 0
	{0b00100, 0b01100, 0b00100, 0b00100, 0b00100, 0b00100, 0b01110}, // 1
	{0b01110, 0b10001, 0b00001, 0b00010, 0b00100, 0b01000, 0b11111}, // 2
	{0b11111, 0b00010, 0b00100, 0b00010, 0b00001, 0b10001, 0b01110}, // 3
	{0b00010, 0b00110, 0b01010, 0b10010, 0b11111, 0b00010, 0b00010}, // 4
	{0b11111, 0b10000, 0b11110, 0b00001, 0b00001, 0b10001, 0b01110}, // 5
	{0b00110, 0b01000, 0b10000, 0b11110, 0b10001, 0b10001, 0b01110}, // 6
	{0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b01000, 0b01000}, // 7
	{0b01110, 0b10001, 0b10001, 0b01110, 0b10001, 0b10001, 0b01110}, // 8
	{0b01110, 0b10001, 0b10001, 0b01111, 0b00001, 0b00010, 0b01100}, // 9
}

// NewCode 生成 4 位随机数字验证码
func NewCode() string {
	b := make([]byte, 4)
	for i := range b {
		b[i] = byte('0' + rand.Intn(10))
	}
	return string(b)
}

// Generate 将验证码绘制为 100x38 的 PNG 写入 w
func Generate(w io.Writer, code string) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, white)
		}
	}

	// 干扰横线
	for i := 0; i < 10; i++ {
		line := color.RGBA{
			R: uint8(70 + rand.Intn(21)),
			G: uint8(70 + rand.Intn(21)),
			B: uint8(70 + rand.Intn(21)),
			A: 255,
		}
		drawLine(img, rand.Intn(99)+1, rand.Intn(99)+1, rand.Intn(99)+1, rand.Intn(37)+1, line)
	}

	// 绘制数字
	const scale = 4
	digitW := 5 * scale
	gap := (width - 4*digitW) / 5
	if gap < 1 {
		gap = 1
	}
	for i := 0; i < 4 && i < len(code); i++ {
		d := int(code[i] - '0')
		if d < 0 || d > 9 {
			continue
		}
		col := color.RGBA{
			R: uint8(30 + rand.Intn(90)),
			G: uint8(30 + rand.Intn(90)),
			B: uint8(30 + rand.Intn(90)),
			A: 255,
		}
		ox := gap + i*(digitW+gap) + rand.Intn(5) - 2
		oy := (height-7*scale)/2 + rand.Intn(5) - 2
		drawDigit(img, digitFont[d], ox, oy, scale, col)
	}

	// 干扰点
	for i := 0; i < 30; i++ {
		dot := color.RGBA{
			R: uint8(80 + rand.Intn(11)),
			G: uint8(80 + rand.Intn(11)),
			B: uint8(80 + rand.Intn(11)),
			A: 255,
		}
		img.Set(rand.Intn(width), rand.Intn(height), dot)
	}

	return png.Encode(w, img)
}

// drawDigit 按比例绘制单个点阵数字
func drawDigit(img *image.RGBA, font [7]uint8, ox, oy, scale int, col color.RGBA) {
	for row := 0; row < 7; row++ {
		bits := font[row]
		for c := 0; c < 5; c++ {
			if bits&(1<<uint(4-c)) == 0 {
				continue
			}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					img.Set(ox+c*scale+dx, oy+row*scale+dy, col)
				}
			}
		}
	}
}

// drawLine 绘制任意方向直线（Bresenham 算法）
func drawLine(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		if image.Pt(x0, y0).In(img.Bounds()) {
			img.Set(x0, y0, col)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
