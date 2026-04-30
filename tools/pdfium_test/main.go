package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	defaultDLLPath = "bin/pdfium.dll"
	defaultPDFPath = "var/temp/temp.pdf"
	defaultPNGPath = "var/temp/temp_from_pdfium.png"
	defaultDPI     = 300

	fpdfAnnot   = 0x01
	fpdfLCDText = 0x02
)

type pdfiumAPI struct {
	dll *syscall.LazyDLL

	initLibrary      *syscall.LazyProc
	destroyLibrary   *syscall.LazyProc
	getLastError     *syscall.LazyProc
	loadMemDocument  *syscall.LazyProc
	closeDocument    *syscall.LazyProc
	getPageCount     *syscall.LazyProc
	getPageSizeIndex *syscall.LazyProc
	loadPage         *syscall.LazyProc
	closePage        *syscall.LazyProc

	bitmapCreate    *syscall.LazyProc
	bitmapDestroy   *syscall.LazyProc
	bitmapFillRect  *syscall.LazyProc
	bitmapGetBuffer *syscall.LazyProc
	bitmapGetStride *syscall.LazyProc

	renderPageBitmap *syscall.LazyProc
}

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	procRtlMoveMemory = kernel32.NewProc("RtlMoveMemory")
)

func main() {
	dllPath := flag.String("dll", defaultDLLPath, "path to pdfium.dll")
	pdfPath := flag.String("pdf", defaultPDFPath, "input PDF path")
	pngPath := flag.String("png", defaultPNGPath, "output PNG path")
	dpi := flag.Int("dpi", defaultDPI, "render DPI")
	pageIndex := flag.Int("page", 0, "zero-based page index")
	flag.Parse()

	if err := renderPDFPageToPNG(*dllPath, *pdfPath, *pngPath, *dpi, *pageIndex); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}

	fmt.Println("PDFium render completed")
	fmt.Println("dll :", *dllPath)
	fmt.Println("pdf :", *pdfPath)
	fmt.Println("png :", *pngPath)
	fmt.Println("dpi :", *dpi)
}

func renderPDFPageToPNG(dllPath, pdfPath, pngPath string, dpi int, pageIndex int) error {
	dllPath = strings.TrimSpace(dllPath)
	pdfPath = strings.TrimSpace(pdfPath)
	pngPath = strings.TrimSpace(pngPath)

	if dllPath == "" {
		return fmt.Errorf("pdfium.dll 경로가 비어 있습니다")
	}
	if pdfPath == "" {
		return fmt.Errorf("PDF 경로가 비어 있습니다")
	}
	if pngPath == "" {
		return fmt.Errorf("PNG 경로가 비어 있습니다")
	}
	if dpi <= 0 {
		dpi = defaultDPI
	}
	if pageIndex < 0 {
		return fmt.Errorf("page index는 0 이상이어야 합니다")
	}

	absDLL, err := filepath.Abs(dllPath)
	if err != nil {
		return fmt.Errorf("pdfium.dll 절대경로 변환 실패: %w", err)
	}
	absPDF, err := filepath.Abs(pdfPath)
	if err != nil {
		return fmt.Errorf("PDF 절대경로 변환 실패: %w", err)
	}
	absPNG, err := filepath.Abs(pngPath)
	if err != nil {
		return fmt.Errorf("PNG 절대경로 변환 실패: %w", err)
	}

	if err := verifyFile(absDLL); err != nil {
		return fmt.Errorf("pdfium.dll 확인 실패: %w", err)
	}
	if err := verifyFile(absPDF); err != nil {
		return fmt.Errorf("PDF 확인 실패: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPNG), 0o755); err != nil {
		return fmt.Errorf("PNG 출력 폴더 생성 실패: %w", err)
	}

	api, err := loadPDFium(absDLL)
	if err != nil {
		return err
	}

	api.initLibrary.Call()
	defer api.destroyLibrary.Call()

	pdfBytes, err := os.ReadFile(absPDF)
	if err != nil {
		return fmt.Errorf("PDF 읽기 실패: %w", err)
	}
	if len(pdfBytes) == 0 {
		return fmt.Errorf("PDF 파일 크기가 0입니다")
	}

	doc, err := api.loadDocumentFromMemory(pdfBytes)
	if err != nil {
		return err
	}
	defer api.closeDocument.Call(doc)
	defer runtime.KeepAlive(pdfBytes)

	pageCount := api.pageCount(doc)
	if pageCount <= 0 {
		return fmt.Errorf("PDF 페이지 수가 0입니다")
	}
	if pageIndex >= pageCount {
		return fmt.Errorf("page index 범위 오류: page=%d, pageCount=%d", pageIndex, pageCount)
	}

	widthPt, heightPt, err := api.pageSize(doc, pageIndex)
	if err != nil {
		return err
	}

	widthPx := int(math.Round(widthPt / 72.0 * float64(dpi)))
	heightPx := int(math.Round(heightPt / 72.0 * float64(dpi)))
	if widthPx <= 0 || heightPx <= 0 {
		return fmt.Errorf("렌더링 크기 계산 실패: %.2fpt x %.2fpt, dpi=%d", widthPt, heightPt, dpi)
	}

	page, err := api.loadPDFPage(doc, pageIndex)
	if err != nil {
		return err
	}
	defer api.closePage.Call(page)

	bitmap, err := api.createBitmap(widthPx, heightPx)
	if err != nil {
		return err
	}
	defer api.bitmapDestroy.Call(bitmap)

	// PDFium color format: 0xAARRGGBB
	api.bitmapFillRect.Call(
		bitmap,
		0,
		0,
		uintptr(widthPx),
		uintptr(heightPx),
		0xFFFFFFFF,
	)

	flags := uintptr(fpdfAnnot | fpdfLCDText)

	api.renderPageBitmap.Call(
		bitmap,
		page,
		0,
		0,
		uintptr(widthPx),
		uintptr(heightPx),
		0,
		flags,
	)

	img, err := api.bitmapToRGBA(bitmap, widthPx, heightPx)
	if err != nil {
		return err
	}

	_ = os.Remove(absPNG)

	out, err := os.Create(absPNG)
	if err != nil {
		return fmt.Errorf("PNG 생성 실패: %w", err)
	}
	defer out.Close()

	if err := png.Encode(out, img); err != nil {
		return fmt.Errorf("PNG 인코딩 실패: %w", err)
	}

	fmt.Println("page count :", pageCount)
	fmt.Println("page size  :", fmt.Sprintf("%.2fpt x %.2fpt", widthPt, heightPt))
	fmt.Println("pixel size :", strconv.Itoa(widthPx)+"x"+strconv.Itoa(heightPx))

	return nil
}

func loadPDFium(dllPath string) (*pdfiumAPI, error) {
	dll := syscall.NewLazyDLL(dllPath)

	api := &pdfiumAPI{
		dll: dll,

		initLibrary:      dll.NewProc("FPDF_InitLibrary"),
		destroyLibrary:   dll.NewProc("FPDF_DestroyLibrary"),
		getLastError:     dll.NewProc("FPDF_GetLastError"),
		loadMemDocument:  dll.NewProc("FPDF_LoadMemDocument"),
		closeDocument:    dll.NewProc("FPDF_CloseDocument"),
		getPageCount:     dll.NewProc("FPDF_GetPageCount"),
		getPageSizeIndex: dll.NewProc("FPDF_GetPageSizeByIndex"),
		loadPage:         dll.NewProc("FPDF_LoadPage"),
		closePage:        dll.NewProc("FPDF_ClosePage"),

		bitmapCreate:    dll.NewProc("FPDFBitmap_Create"),
		bitmapDestroy:   dll.NewProc("FPDFBitmap_Destroy"),
		bitmapFillRect:  dll.NewProc("FPDFBitmap_FillRect"),
		bitmapGetBuffer: dll.NewProc("FPDFBitmap_GetBuffer"),
		bitmapGetStride: dll.NewProc("FPDFBitmap_GetStride"),

		renderPageBitmap: dll.NewProc("FPDF_RenderPageBitmap"),
	}

	required := map[string]*syscall.LazyProc{
		"FPDF_InitLibrary":        api.initLibrary,
		"FPDF_DestroyLibrary":     api.destroyLibrary,
		"FPDF_GetLastError":       api.getLastError,
		"FPDF_LoadMemDocument":    api.loadMemDocument,
		"FPDF_CloseDocument":      api.closeDocument,
		"FPDF_GetPageCount":       api.getPageCount,
		"FPDF_GetPageSizeByIndex": api.getPageSizeIndex,
		"FPDF_LoadPage":           api.loadPage,
		"FPDF_ClosePage":          api.closePage,
		"FPDFBitmap_Create":       api.bitmapCreate,
		"FPDFBitmap_Destroy":      api.bitmapDestroy,
		"FPDFBitmap_FillRect":     api.bitmapFillRect,
		"FPDFBitmap_GetBuffer":    api.bitmapGetBuffer,
		"FPDFBitmap_GetStride":    api.bitmapGetStride,
		"FPDF_RenderPageBitmap":   api.renderPageBitmap,
	}

	for name, proc := range required {
		if err := proc.Find(); err != nil {
			return nil, fmt.Errorf("pdfium export 함수 확인 실패: %s: %w", name, err)
		}
	}

	return api, nil
}

func (api *pdfiumAPI) loadDocumentFromMemory(pdfBytes []byte) (uintptr, error) {
	if len(pdfBytes) == 0 {
		return 0, fmt.Errorf("PDF 메모리 버퍼가 비어 있습니다")
	}

	r1, _, _ := api.loadMemDocument.Call(
		uintptr(unsafe.Pointer(&pdfBytes[0])),
		uintptr(len(pdfBytes)),
		0,
	)

	if r1 == 0 {
		return 0, fmt.Errorf("FPDF_LoadMemDocument 실패: pdfium_error=%d", api.lastError())
	}

	return r1, nil
}

func (api *pdfiumAPI) pageCount(doc uintptr) int {
	r1, _, _ := api.getPageCount.Call(doc)
	return int(r1)
}

func (api *pdfiumAPI) pageSize(doc uintptr, pageIndex int) (float64, float64, error) {
	var width float64
	var height float64

	r1, _, _ := api.getPageSizeIndex.Call(
		doc,
		uintptr(pageIndex),
		uintptr(unsafe.Pointer(&width)),
		uintptr(unsafe.Pointer(&height)),
	)

	if r1 == 0 {
		return 0, 0, fmt.Errorf("FPDF_GetPageSizeByIndex 실패: page=%d pdfium_error=%d", pageIndex, api.lastError())
	}

	return width, height, nil
}

func (api *pdfiumAPI) loadPDFPage(doc uintptr, pageIndex int) (uintptr, error) {
	r1, _, _ := api.loadPage.Call(doc, uintptr(pageIndex))
	if r1 == 0 {
		return 0, fmt.Errorf("FPDF_LoadPage 실패: page=%d pdfium_error=%d", pageIndex, api.lastError())
	}
	return r1, nil
}

func (api *pdfiumAPI) createBitmap(width, height int) (uintptr, error) {
	r1, _, _ := api.bitmapCreate.Call(
		uintptr(width),
		uintptr(height),
		1,
	)
	if r1 == 0 {
		return 0, fmt.Errorf("FPDFBitmap_Create 실패: %dx%d", width, height)
	}
	return r1, nil
}

func copyFromNativeBuffer(src uintptr, size int) ([]byte, error) {
	if src == 0 {
		return nil, fmt.Errorf("native buffer pointer is null")
	}
	if size <= 0 {
		return nil, fmt.Errorf("native buffer size is invalid: %d", size)
	}

	if err := procRtlMoveMemory.Find(); err != nil {
		return nil, fmt.Errorf("RtlMoveMemory 확인 실패: %w", err)
	}

	dst := make([]byte, size)

	syscall.SyscallN(
		procRtlMoveMemory.Addr(),
		uintptr(unsafe.Pointer(&dst[0])),
		src,
		uintptr(size),
	)

	runtime.KeepAlive(dst)

	return dst, nil
}

func (api *pdfiumAPI) bitmapToRGBA(bitmap uintptr, width, height int) (*image.RGBA, error) {
	bufferPtr, _, _ := api.bitmapGetBuffer.Call(bitmap)
	if bufferPtr == 0 {
		return nil, fmt.Errorf("FPDFBitmap_GetBuffer 실패")
	}

	strideRaw, _, _ := api.bitmapGetStride.Call(bitmap)
	stride := int(strideRaw)
	if stride <= 0 {
		return nil, fmt.Errorf("FPDFBitmap_GetStride 실패: %d", stride)
	}
	if stride < width*4 {
		return nil, fmt.Errorf("bitmap stride가 너무 작습니다: stride=%d width=%d", stride, width)
	}

	rawSize := stride * height

	// PDFium이 가진 외부 메모리를 Go slice로 직접 감싸지 않고,
	// Windows API로 Go 메모리로 복사한 뒤 처리합니다.
	raw, err := copyFromNativeBuffer(bufferPtr, rawSize)
	if err != nil {
		return nil, err
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		srcRow := y * stride
		dstRow := y * img.Stride

		for x := 0; x < width; x++ {
			src := srcRow + x*4
			dst := dstRow + x*4

			// PDFium bitmap 기본 순서: BGRA
			b := raw[src+0]
			g := raw[src+1]
			r := raw[src+2]
			a := raw[src+3]

			if a == 0 {
				a = 255
			}

			img.Pix[dst+0] = r
			img.Pix[dst+1] = g
			img.Pix[dst+2] = b
			img.Pix[dst+3] = a
		}
	}

	return img, nil
}

func (api *pdfiumAPI) lastError() uintptr {
	if api.getLastError == nil {
		return 0
	}
	r1, _, _ := api.getLastError.Call()
	return r1
}

func verifyFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("디렉토리입니다: %s", path)
	}
	if info.Size() <= 0 {
		return fmt.Errorf("파일 크기가 0입니다: %s", path)
	}
	return nil
}
