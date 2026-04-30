import sys
from pathlib import Path

import fitz  # PyMuPDF


def pdf_to_png(pdf_path: str, png_path: str, dpi: int = 300, page_no: int = 0) -> None:
    pdf_file = Path(pdf_path)
    png_file = Path(png_path)

    if not pdf_file.exists():
        raise FileNotFoundError(f"PDF 파일을 찾을 수 없습니다: {pdf_file}")

    png_file.parent.mkdir(parents=True, exist_ok=True)

    doc = fitz.open(str(pdf_file))
    try:
        if page_no < 0 or page_no >= len(doc):
            raise ValueError(f"page_no 범위 오류: {page_no}, 전체 페이지 수: {len(doc)}")

        page = doc[page_no]

        zoom = dpi / 72.0
        matrix = fitz.Matrix(zoom, zoom)

        pix = page.get_pixmap(
            matrix=matrix,
            alpha=False
        )

        pix.save(str(png_file))
    finally:
        doc.close()


if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("사용법: python pdf_to_png.py <input.pdf> <output.png> [dpi]")
        sys.exit(1)

    input_pdf = sys.argv[1]
    output_png = sys.argv[2]
    dpi = int(sys.argv[3]) if len(sys.argv) >= 4 else 300

    pdf_to_png(input_pdf, output_png, dpi)
    print(f"PNG 생성 완료: {output_png}")
    