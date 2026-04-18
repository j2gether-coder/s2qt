package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"

	_ "image/jpeg"
	_ "image/png"

	"s2qt/util"
)

type SkinTestInput struct {
	Enabled       bool   `json:"enabled"`
	SkinImagePath string `json:"skin_image_path"`

	OutputPNGPath string `json:"output_png_path"`
	OutputPDFPath string `json:"output_pdf_path"`

	// 전경(temp.png)이 들어갈 영역
	ForegroundLeftPX   int `json:"foreground_left_px"`
	ForegroundTopPX    int `json:"foreground_top_px"`
	ForegroundWidthPX  int `json:"foreground_width_px"`
	ForegroundHeightPX int `json:"foreground_height_px"`

	// contain / cover
	FitMode string `json:"fit_mode"`

	// center / left / right
	AlignX string `json:"align_x"`
	// center / top / bottom
	AlignY string `json:"align_y"`

	// 디버그용 rect 표시
	DebugRect bool `json:"debug_rect"`
}

func loadSkinTestJSON(path string) (*SkinTestInput, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read test_skin.json: %w", err)
	}

	var in SkinTestInput
	if err := json.Unmarshal(b, &in); err != nil {
		return nil, fmt.Errorf("failed to parse test_skin.json: %w", err)
	}

	if !in.Enabled {
		return &in, nil
	}

	in.SkinImagePath = strings.TrimSpace(in.SkinImagePath)
	in.OutputPNGPath = strings.TrimSpace(in.OutputPNGPath)
	in.OutputPDFPath = strings.TrimSpace(in.OutputPDFPath)
	in.FitMode = strings.ToLower(strings.TrimSpace(in.FitMode))
	in.AlignX = strings.ToLower(strings.TrimSpace(in.AlignX))
	in.AlignY = strings.ToLower(strings.TrimSpace(in.AlignY))

	if in.SkinImagePath == "" {
		return nil, fmt.Errorf("skin_image_path is empty")
	}
	if in.OutputPNGPath == "" {
		in.OutputPNGPath = filepath.Join("var", "temp", "temp_skin.png")
	}
	if in.OutputPDFPath == "" {
		in.OutputPDFPath = filepath.Join("var", "temp", "temp_skin.pdf")
	}

	if in.ForegroundLeftPX < 0 {
		in.ForegroundLeftPX = 0
	}
	if in.ForegroundTopPX < 0 {
		in.ForegroundTopPX = 0
	}
	if in.ForegroundWidthPX <= 0 {
		in.ForegroundWidthPX = 2240
	}
	if in.ForegroundHeightPX <= 0 {
		in.ForegroundHeightPX = 3180
	}

	if in.FitMode == "" {
		in.FitMode = "contain"
	}
	if in.AlignX == "" {
		in.AlignX = "center"
	}
	if in.AlignY == "" {
		in.AlignY = "top"
	}

	return &in, nil
}

func runSkinTest(paths *util.AppPaths, db *sql.DB, skin *SkinTestInput) []string {
	var outputs []string

	_ = db // 현재 PNG 합성안에서는 사용하지 않음

	if paths == nil {
		fmt.Println("[WARN] skin test skipped: paths is nil")
		return outputs
	}
	if skin == nil || !skin.Enabled {
		fmt.Println("[INFO] skin test skipped: disabled")
		return outputs
	}

	if _, err := os.Stat(paths.TempPng); err != nil {
		fmt.Printf("[WARN] skin test skipped: temp.png not found: %v\n", err)
		return outputs
	}

	if err := os.MkdirAll(filepath.Dir(skin.OutputPNGPath), 0o755); err != nil {
		fmt.Printf("[WARN] skin png dir create failed: %v\n", err)
		return outputs
	}
	if err := os.MkdirAll(filepath.Dir(skin.OutputPDFPath), 0o755); err != nil {
		fmt.Printf("[WARN] skin pdf dir create failed: %v\n", err)
	}

	fmt.Println("=== Skin Composite Start ===")
	fmt.Printf("template png : %s\n", skin.SkinImagePath)
	fmt.Printf("foreground   : %s\n", paths.TempPng)
	fmt.Printf("output png   : %s\n", skin.OutputPNGPath)
	fmt.Printf("fit mode     : %s\n", skin.FitMode)
	fmt.Printf("rect         : left=%d, top=%d, width=%d, height=%d\n",
		skin.ForegroundLeftPX,
		skin.ForegroundTopPX,
		skin.ForegroundWidthPX,
		skin.ForegroundHeightPX,
	)
	fmt.Println("[INFO] template is resized to foreground canvas before compositing")

	if err := composeTemplateAndForeground(
		skin.SkinImagePath,
		paths.TempPng,
		skin.OutputPNGPath,
		skin,
	); err != nil {
		fmt.Printf("[WARN] skin png compose failed: %v\n", err)
	} else {
		fmt.Printf("[OK] skin png : %s\n", skin.OutputPNGPath)
		outputs = append(outputs, skin.OutputPNGPath)
	}

	// 비교용으로 안정적인 temp.pdf를 temp_skin.pdf로 복사
	if _, err := os.Stat(paths.TempPdf); err == nil {
		if err := copyFile(paths.TempPdf, skin.OutputPDFPath); err != nil {
			fmt.Printf("[WARN] skin pdf copy failed: %v\n", err)
		} else {
			fmt.Printf("[OK] skin pdf : %s\n", skin.OutputPDFPath)
			outputs = append(outputs, skin.OutputPDFPath)
		}
	} else {
		fmt.Printf("[WARN] temp.pdf not found, skip pdf copy: %v\n", err)
	}

	fmt.Println("=== Skin Composite Done ===")
	return outputs
}

func composeTemplateAndForeground(templatePath, foregroundPath, outputPath string, skin *SkinTestInput) error {
	bgImg, err := decodeImageFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to load template image: %w", err)
	}

	fgImg, err := decodeImageFile(foregroundPath)
	if err != nil {
		return fmt.Errorf("failed to load foreground image: %w", err)
	}

	bgBounds := bgImg.Bounds()
	bgW := bgBounds.Dx()
	bgH := bgBounds.Dy()

	fgBounds := fgImg.Bounds()
	fgW := fgBounds.Dx()
	fgH := fgBounds.Dy()

	if fgW <= 0 || fgH <= 0 {
		return fmt.Errorf("invalid foreground image size: %dx%d", fgW, fgH)
	}

	// 핵심:
	// 최종 캔버스는 foreground(temp.png) 기준으로 고정
	canvas := image.NewRGBA(image.Rect(0, 0, fgW, fgH))

	// template.png를 foreground 캔버스 크기로 먼저 정규화해서 배경으로 깐다
	xdraw.CatmullRom.Scale(
		canvas,
		canvas.Bounds(),
		bgImg,
		bgBounds,
		xdraw.Src,
		nil,
	)

	targetRect := image.Rect(
		skin.ForegroundLeftPX,
		skin.ForegroundTopPX,
		skin.ForegroundLeftPX+skin.ForegroundWidthPX,
		skin.ForegroundTopPX+skin.ForegroundHeightPX,
	).Intersect(canvas.Bounds())

	if targetRect.Empty() {
		return fmt.Errorf("foreground rect is empty or out of bounds: %v", targetRect)
	}

	dstRect, srcRect := calcPlacement(
		fgW,
		fgH,
		targetRect,
		skin.FitMode,
		skin.AlignX,
		skin.AlignY,
	)

	fmt.Printf("[DEBUG] template size   : %dx%d\n", bgW, bgH)
	fmt.Printf("[DEBUG] foreground size : %dx%d\n", fgW, fgH)
	fmt.Printf("[DEBUG] canvas size     : %dx%d\n", fgW, fgH)
	fmt.Printf("[DEBUG] target rect     : x=%d, y=%d, w=%d, h=%d\n",
		targetRect.Min.X,
		targetRect.Min.Y,
		targetRect.Dx(),
		targetRect.Dy(),
	)
	fmt.Printf("[DEBUG] dst rect        : x=%d, y=%d, w=%d, h=%d\n",
		dstRect.Min.X,
		dstRect.Min.Y,
		dstRect.Dx(),
		dstRect.Dy(),
	)

	// temp.png를 지정 rect 안에 contain/cover로 배치
	xdraw.CatmullRom.Scale(
		canvas,
		dstRect,
		fgImg,
		srcRect,
		xdraw.Over,
		nil,
	)

	if skin.DebugRect {
		drawRectBorder(canvas, targetRect, color.RGBA{R: 220, G: 38, B: 38, A: 255}, 3)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output png: %w", err)
	}
	defer out.Close()

	if err := png.Encode(out, canvas); err != nil {
		return fmt.Errorf("failed to encode output png: %w", err)
	}

	return nil
}

func calcPlacement(
	fgW, fgH int,
	target image.Rectangle,
	fitMode, alignX, alignY string,
) (image.Rectangle, image.Rectangle) {
	tw := target.Dx()
	th := target.Dy()

	srcRect := image.Rect(0, 0, fgW, fgH)

	if fitMode == "cover" {
		scale := math.Max(float64(tw)/float64(fgW), float64(th)/float64(fgH))
		scaledW := int(math.Round(float64(fgW) * scale))
		scaledH := int(math.Round(float64(fgH) * scale))

		dstX := calcAlignedOffset(target.Min.X, tw, scaledW, alignX)
		dstY := calcAlignedOffset(target.Min.Y, th, scaledH, alignY)

		return image.Rect(dstX, dstY, dstX+scaledW, dstY+scaledH), srcRect
	}

	// 기본: contain
	scale := math.Min(float64(tw)/float64(fgW), float64(th)/float64(fgH))
	scaledW := int(math.Round(float64(fgW) * scale))
	scaledH := int(math.Round(float64(fgH) * scale))

	dstX := calcAlignedOffset(target.Min.X, tw, scaledW, alignX)
	dstY := calcAlignedOffset(target.Min.Y, th, scaledH, alignY)

	return image.Rect(dstX, dstY, dstX+scaledW, dstY+scaledH), srcRect
}

func calcAlignedOffset(start, outer, inner int, align string) int {
	switch align {
	case "left", "top":
		return start
	case "right", "bottom":
		return start + (outer - inner)
	default:
		return start + (outer-inner)/2
	}
}

func decodeImageFile(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

func drawImage(dst *image.RGBA, src image.Image, dstRect image.Rectangle) {
	xdraw.Draw(dst, dstRect, src, src.Bounds().Min, xdraw.Src)
}

func drawRectBorder(img *image.RGBA, rect image.Rectangle, c color.Color, thickness int) {
	if thickness <= 0 {
		thickness = 1
	}

	// top
	fillRect(img, image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+thickness), c)
	// bottom
	fillRect(img, image.Rect(rect.Min.X, rect.Max.Y-thickness, rect.Max.X, rect.Max.Y), c)
	// left
	fillRect(img, image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+thickness, rect.Max.Y), c)
	// right
	fillRect(img, image.Rect(rect.Max.X-thickness, rect.Min.Y, rect.Max.X, rect.Max.Y), c)
}

func fillRect(img *image.RGBA, rect image.Rectangle, c color.Color) {
	r := rect.Intersect(img.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.Set(x, y, c)
		}
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Close()
}
