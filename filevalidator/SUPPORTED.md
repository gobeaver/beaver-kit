# Supported File Formats

The `filevalidator` package supports **60+ file formats** across various categories.

## 🖼️ Images

| Format | MIME Type | Extensions | Validation Features |
|:-------|:----------|:-----------|:--------------------|
| **JPEG** | `image/jpeg` | `.jpg`, `.jpeg` | Dimensions, Pixel limit |
| **PNG** | `image/png` | `.png` | Dimensions, Pixel limit |
| **GIF** | `image/gif` | `.gif` | Dimensions, Pixel limit |
| **WebP** | `image/webp` | `.webp` | Dimensions, Pixel limit |
| **BMP** | `image/bmp` | `.bmp` | Dimensions, Pixel limit |
| **TIFF** | `image/tiff` | `.tiff`, `.tif` | Dimensions, Pixel limit |
| **SVG** | `image/svg+xml` | `.svg` | File size limit (XML-based) |
| **Icon** | `image/x-icon` | `.ico` | Dimensions, Pixel limit |
| **HEIC** | `image/heic` | `.heic` | Magic bytes detection only |
| **AVIF** | `image/avif` | `.avif` | Magic bytes detection only |

## 📄 Documents

| Format | MIME Type | Extensions | Validation Features |
|:-------|:----------|:-----------|:--------------------|
| **PDF** | `application/pdf` | `.pdf` | Header/Trailer structure check |
| **Word** | `application/vnd.openxmlformats...` | `.docx` | ZIP structure, Macro detection |
| **Excel** | `application/vnd.openxmlformats...` | `.xlsx` | ZIP structure, Macro detection |
| **PowerPoint** | `application/vnd.openxmlformats...` | `.pptx` | ZIP structure, Macro detection |
| **Word (Macro)** | `application/vnd.ms-word...` | `.docm` | Blocked by default |
| **Excel (Macro)** | `application/vnd.ms-excel...` | `.xlsm` | Blocked by default |
| **PowerPoint (Macro)** | `application/vnd.ms-powerpoint...` | `.pptm` | Blocked by default |
| **RTF** | `application/rtf` | `.rtf` | Magic bytes detection only |

## 📦 Archives

All archive formats include **Zip Bomb Protection** (compression ratio, nested archives, file count limits).

| Format | MIME Type | Extensions | Validation Features |
|:-------|:----------|:-----------|:--------------------|
| **ZIP** | `application/zip` | `.zip` | Full content validation |
| **GZIP** | `application/gzip` | `.gz` | Header validation |
| **TAR** | `application/x-tar` | `.tar` | Header validation |
| **TAR.GZ** | `application/x-gtar` | `.tar.gz`, `.tgz` | Header validation |
| **RAR** | `application/x-rar-compressed` | `.rar` | Magic bytes detection only |
| **7-Zip** | `application/x-7z-compressed` | `.7z` | Magic bytes detection only |
| **BZIP2** | `application/x-bzip2` | `.bz2` | Magic bytes detection only |
| **XZ** | `application/x-xz` | `.xz` | Magic bytes detection only |
| **JAR** | `application/java-archive` | `.jar` | Treated as ZIP |

## 🎵 Audio

| Format | MIME Type | Extensions | Validation Features |
|:-------|:----------|:-----------|:--------------------|
| **MP3** | `audio/mpeg` | `.mp3` | ID3/Frame sync check |
| **WAV** | `audio/wav` | `.wav` | RIFF/WAVE header check |
| **OGG** | `audio/ogg` | `.ogg` | OggS header check |
| **FLAC** | `audio/flac` | `.flac` | fLaC header check |
| **AAC** | `audio/aac` | `.aac` | ADTS/ADIF header check |
| **MIDI** | `audio/midi` | `.mid`, `.midi` | Magic bytes detection only |
| **M4A** | `audio/mp4` | `.m4a` | FTYP brand check |

## 🎬 Video

| Format | MIME Type | Extensions | Validation Features |
|:-------|:----------|:-----------|:--------------------|
| **MP4** | `video/mp4` | `.mp4` | FTYP brand check |
| **WebM** | `video/webm` | `.webm` | EBML header check |
| **MKV** | `video/x-matroska` | `.mkv` | EBML header check |
| **AVI** | `video/x-msvideo` | `.avi` | RIFF/AVI header check |
| **MOV** | `video/quicktime` | `.mov` | MOOV atom check |
| **FLV** | `video/x-flv` | `.flv` | FLV header check |
| **3GP** | `video/3gpp` | `.3gp` | FTYP brand check |
| **M4V** | `video/x-m4v` | `.m4v` | FTYP brand check |

## 📝 Text & Data

| Format | MIME Type | Extensions | Validation Features |
|:-------|:----------|:-----------|:--------------------|
| **JSON** | `application/json` | `.json` | Syntax, Depth limit |
| **XML** | `application/xml` | `.xml` | Syntax, XXE protection |
| **CSV** | `text/csv` | `.csv` | Row/Col limits, UTF-8 check |
| **HTML** | `text/html` | `.html` | Magic bytes detection only |
| **Plain Text** | `text/plain` | `.txt` | UTF-8/ASCII check |

## 🚫 Executables (Detected & Blocked)

These formats are detected specifically to be **blocked** or flagged.

| Format | MIME Type | Extensions |
|:-------|:----------|:-----------|
| **EXE/DLL** | `application/x-msdownload` | `.exe`, `.dll` |
| **ELF** | `application/x-executable` | (No extension) |
| **Mach-O** | `application/x-mach-binary` | (No extension) |
| **Scripts** | `application/x-sh`, etc. | `.sh`, `.bat` |

## 🔤 Fonts

| Format | MIME Type | Extensions |
|:-------|:----------|:-----------|
| **WOFF** | `font/woff` | `.woff` |
| **WOFF2** | `font/woff2` | `.woff2` |
| **TTF** | `font/ttf` | `.ttf` |
| **OTF** | `font/otf` | `.otf` |
