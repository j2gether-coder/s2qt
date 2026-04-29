# PDFium Runtime Test

## Purpose

This document records a temporary PDFium DLL download test for S2QT.

## Installed File

- bin\pdfium.dll

## Downloaded Package

- var\data\pdfium_test\bblanchon.PDFium.Win32.149.0.7811.nupkg

## Selected Entry

- runtimes/win-x64/native/pdfium.dll

## Source Package

- Package: bblanchon.PDFium
- URL: https://www.nuget.org/api/v2/package/bblanchon.PDFium.Win32/149.0.7811

## Notes

This is a temporary test flow.

If PDFium-based PDF to PNG conversion is verified successfully, the final runtime installation flow should be moved into util_service.go.

Target final placement:

- bin/pdfium.dll
- bin/pdfium_to_png.exe

The existing HTML screenshot-based PNG generation should remain as fallback until PDFium conversion is stable.
