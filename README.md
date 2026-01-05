# w2app - Web to Desktop App Generator

CLI tool untuk mengubah website/URL menjadi aplikasi desktop standalone (~3MB).

## Features

- **Single executable output** (~3MB per app)
- **Icon dari URL** - Download icon langsung dari URL
- **Auto-fetch favicon** - Otomatis ambil favicon dari website target
- **Custom icon embedding** - Support .ico, .png, .jpg (file atau URL)
- **Single instance mode** - Cegah multiple window
- **CSS/JS injection** - Kustomisasi tampilan dan behavior
- **Keyboard shortcuts** - F11 fullscreen, zoom, refresh
- **Fullscreen/Kiosk mode** - Untuk digital signage, kiosk
- **Always-on-top** - Window selalu di atas
- **Frameless window** - Tanpa title bar
- **External link handling** - Buka di browser sistem
- **Tidak perlu Go di PC target** - Portable
- Menggunakan WebView2 (Edge) di Windows

## Requirements

- **Untuk build w2app**: Go 1.21+
- **Untuk menjalankan app hasil generate**: WebView2 Runtime (biasanya sudah terinstall di Windows 10/11)

## Installation

### Download Release

Download `w2app.exe` dari [Releases](https://github.com/user/w2app/releases).

### Build dari source

```bash
# Clone repository
git clone https://github.com/user/w2app.git
cd w2app

# Build
build.bat
```

## Quick Start

```bash
# Basic - buat app dari URL dengan auto-fetch favicon
w2app --url https://google.com --name GoogleApp --auto-icon

# Icon dari URL
w2app -u https://github.com -n GitHub --icon https://github.com/favicon.ico

# WhatsApp desktop dengan single instance
w2app -u https://web.whatsapp.com -n WhatsApp --single-instance --auto-icon
```

## Usage

```bash
w2app [command] [options]

Commands:
  create     Buat aplikasi desktop dari URL (default)
  platforms  Tampilkan platform yang tersedia
  version    Tampilkan versi
  help       Tampilkan bantuan
```

### Options

#### Basic
| Option | Short | Description |
|--------|-------|-------------|
| `--url` | `-u` | URL target (wajib) |
| `--name` | `-n` | Nama aplikasi (wajib) |
| `--icon` | `-i` | Path/URL ke icon file (.ico, .png, .jpg) |
| `--auto-icon` | | Auto-fetch favicon dari URL target |
| `--out` | `-o` | Direktori output (default: .) |
| `--platform` | `-p` | Platform target (default: windows) |

#### Window
| Option | Description |
|--------|-------------|
| `--width` | Lebar window (default: 1024) |
| `--height` | Tinggi window (default: 768) |
| `--resizable` | Window bisa di-resize (default: true) |
| `--fullscreen` | Start dalam mode fullscreen |
| `--frameless` | Window tanpa frame/border |
| `--always-on-top` | Window selalu di atas |
| `--maximized` | Start dalam kondisi maximized |

#### Behavior
| Option | Description |
|--------|-------------|
| `--single-instance` | Hanya boleh 1 instance berjalan |
| `--user-agent` | Custom User-Agent string |
| `--clear-cache` | Hapus cache saat exit |

#### Injection
| Option | Description |
|--------|-------------|
| `--inject-css` | CSS string untuk di-inject |
| `--inject-js` | JavaScript string untuk di-inject |
| `--css-file` | Path ke file CSS untuk di-inject |
| `--js-file` | Path ke file JS untuk di-inject |

#### Navigation
| Option | Description |
|--------|-------------|
| `--whitelist` | Domain whitelist (comma-separated) |
| `--block-external` | Block navigasi ke external URL |

#### Advanced
| Option | Description |
|--------|-------------|
| `--no-context-menu` | Disable klik kanan |
| `--no-devtools` | Disable DevTools (F12) |

## Examples

### Basic App dengan Auto Icon
```bash
w2app --url https://google.com --name GoogleApp --auto-icon
```

### App dengan Icon dari URL
```bash
w2app -u https://github.com -n GitHub --icon https://github.com/favicon.ico
```

### Chat App (Single Instance)
```bash
w2app -u https://web.whatsapp.com -n WhatsApp --single-instance --auto-icon --maximized
```

### Kiosk/Signage Mode
```bash
w2app --url https://dashboard.example.com --name Dashboard \
  --fullscreen --no-context-menu --no-devtools --auto-icon
```

### Custom Styling (Dark Mode)
```bash
# style.css
body { filter: invert(1) hue-rotate(180deg); }
img, video { filter: invert(1) hue-rotate(180deg); }

# Generate
w2app -u https://example.com -n DarkApp --css-file style.css --auto-icon
```

### Corporate App dengan Whitelist
```bash
w2app --url https://app.company.com --name CompanyApp \
  --whitelist "app.company.com,api.company.com,auth.company.com" \
  --single-instance --auto-icon
```

### Floating Widget
```bash
w2app -u https://clock.example.com -n FloatingClock \
  --width 200 --height 100 --frameless --always-on-top --resizable=false
```

## Icon Sources

w2app mendukung berbagai sumber icon:

1. **File lokal**: `--icon ./myicon.ico`
2. **URL langsung**: `--icon https://example.com/icon.png`
3. **Auto-fetch**: `--auto-icon` (otomatis cari favicon)

Auto-fetch akan mencoba:
- `/favicon.ico`
- `/favicon.png`
- `/apple-touch-icon.png`
- Google Favicon Service
- DuckDuckGo Icon Service

## Keyboard Shortcuts

Shortcut yang tersedia di dalam aplikasi:

| Shortcut | Action |
|----------|--------|
| `F11` | Toggle fullscreen |
| `F5` / `Ctrl+R` | Refresh page |
| `Ctrl+Plus` | Zoom in |
| `Ctrl+Minus` | Zoom out |
| `Ctrl+0` | Reset zoom |

## Output

Aplikasi yang dihasilkan:
- Single executable file (.exe untuk Windows)
- Ukuran ~3MB
- Portable - bisa dipindahkan/disalin ke PC lain
- Tidak perlu Go terinstall di PC target

## How It Works

w2app menggunakan arsitektur "stub-based packaging":

1. **Stub**: Pre-compiled webview launcher yang membaca config dari tail binary
2. **Generator**: CLI yang meng-append config JSON ke stub untuk membuat app baru

```
[stub binary] + [marker] + [config JSON] = [final app.exe]
```

## Troubleshooting

### "Gagal membuat webview" error
WebView2 Runtime tidak terinstall. Download dari:
https://developer.microsoft.com/en-us/microsoft-edge/webview2/

### App tidak berjalan di PC lain
Pastikan target PC memiliki:
- Windows 10/11
- WebView2 Runtime terinstall

### Icon tidak muncul
- Pastikan file/URL icon valid (.ico, .png, .jpg)
- Untuk hasil terbaik, gunakan file .ico dengan multiple sizes
- Coba `--auto-icon` untuk otomatis fetch favicon

### Auto-icon gagal
- Pastikan koneksi internet aktif
- Website mungkin tidak memiliki favicon standard
- Coba gunakan `--icon <url>` dengan URL icon langsung

## File Structure

```
web2desktop/
├── cmd/
│   ├── stub/          # Stub launcher source
│   │   └── main.go
│   └── w2app/         # Generator CLI source
│       └── main.go
├── internal/
│   ├── config/        # Shared config struct
│   │   └── config.go
│   └── generator/     # Generator logic
│       ├── generator.go
│       └── stubs/     # Pre-compiled stubs
│           └── stub-windows-amd64.exe
├── build.bat          # Build script
├── go.mod
└── README.md
```

## Roadmap

- [ ] Support Linux (GTK WebKit)
- [ ] Support macOS (WebKit)
- [ ] System tray support
- [ ] Auto-update mechanism
- [ ] Offline mode (embed webpage snapshot)
- [ ] Notification support

## License

MIT
