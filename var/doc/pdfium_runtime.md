# PDFium Runtime

## Purpose

S2QT uses PDFium as the preferred runtime for converting generated PDF files into PNG images.

## Installed File

- bin/pdfium.dll

Current installed path:

D:\gotest\s2qt\bin\pdfium.dll

## Source File

The DLL was extracted from:

D:\gotest\s2qt\var\data\pdfium_extract\runtimes\win-x64\native\pdfium.dll

## Downloaded Package

- Package: bblanchon.PDFium.Win32
- URL: https://www.nuget.org/api/v2/package/bblanchon.PDFium.Win32/149.0.7811

## S2QT PNG Policy

S2QT output policy:

- HTML: review/edit/preview
- PDF: official document
- PNG: shared image generated from PDF

The PDFium path should be used first for PDF-to-PNG conversion.
The existing HTML screenshot-based PNG generation remains as fallback.

## Packaging Policy

Current runtime placement:

- bin/pdfium.dll

Future candidate:

- s2qt.exe internal PDFium rendering using bin/pdfium.dll

## License Notice

Keep the relevant PDFium license and third-party notices with the application package before redistribution.
