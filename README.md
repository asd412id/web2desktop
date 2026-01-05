# W2App - Web to Desktop App Generator

CLI tool untuk mengubah website/URL menjadi aplikasi desktop Windows standalone (~4MB).

## Features

### Core
- **Single executable output** (~4MB per app)
- **WebView2-based** - Menggunakan Microsoft Edge WebView2 (modern, fast)
- **Portable** - Tidak perlu instalasi, bisa dipindahkan ke PC lain
- **Icon embedding** - Support .ico, .png, .jpg (file atau URL)
- **Auto-fetch favicon** - Otomatis ambil favicon dari website target

### System Tray
- **Tray icon** - App bisa minimize ke system tray
- **Tray menu** - Show, Hide, Exit via right-click menu
- **Double-click to show** - Klik dua kali tray icon untuk show window
- **Close to tray** - Tombol close minimize ke tray instead of exit
- **Minimize to tray** - Tombol minimize langsung ke tray

### Notifications
- **Native Windows Toast** - Notifikasi native Windows 10/11
- **Auto AppUserModelID** - Otomatis register AUMID untuk toast
- **Auto Start Menu shortcut** - Otomatis buat shortcut untuk notifikasi
- **Click handling** - Klik notifikasi untuk focus app

### Window
- **Single instance mode** - Cegah multiple window, focus existing
- **Fullscreen/Kiosk mode** - Untuk digital signage, kiosk
- **Start maximized** - Mulai dalam kondisi maximized
- **Frameless window** - Tanpa title bar
- **Custom titlebar color** - Warna titlebar kustom (hex atau dark/light)
- **Always-on-top** - Window selalu di atas

### Auto-start
- **Windows startup** - Option untuk start saat Windows boot
- **Start minimized** - Mulai dalam kondisi minimized ke tray
- **Tray checkbox** - Toggle auto-start dari tray menu

### Injection
- **CSS injection** - Inject custom CSS ke halaman
- **JS injection** - Inject custom JavaScript ke halaman
- **External link handler** - Buka link eksternal di browser default

### Keyboard Shortcuts
| Shortcut | Action |
|----------|--------|
| `F11` | Toggle fullscreen |
| `F5` / `Ctrl+R` | Refresh page |
| `Ctrl+Plus` | Zoom in |
| `Ctrl+Minus` | Zoom out |
| `Ctrl+0` | Reset zoom |

## Requirements

- **Untuk build w2app**: Go 1.21+
- **Untuk menjalankan app hasil generate**: WebView2 Runtime (biasanya sudah terinstall di Windows 10/11)

## Installation

### Download Release

Download `w2app.exe` dari [Releases](https://github.com/user/w2app/releases).

### Build dari Source

```bash
# Clone repository
git clone https://github.com/user/w2app.git
cd w2app

# Build (Windows)
build.bat

# Atau manual:
go build -ldflags="-s -w -H windowsgui" -o internal/generator/stubs/stub-windows-amd64.exe ./cmd/stub
go build -ldflags="-s -w" -o w2app.exe ./cmd/w2app
```

## Quick Start

```bash
# Basic - buat app dari URL dengan auto-fetch favicon
w2app create --url https://google.com --name GoogleApp --auto-icon

# WhatsApp desktop dengan semua fitur
w2app create -u https://web.whatsapp.com -n WhatsApp \
  --auto-icon --tray --close-to-tray --single-instance \
  --enable-notification --auto-startup

# Kiosk mode untuk digital signage
w2app create -u https://dashboard.example.com -n Dashboard \
  --fullscreen --no-context-menu --no-devtools --auto-icon
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
| `--titlebar-color` | Warna titlebar (hex: #RRGGBB, atau "dark"/"light") |

#### System Tray
| Option | Description |
|--------|-------------|
| `--tray` | Enable system tray icon |
| `--close-to-tray` | Close button minimize ke tray |
| `--minimize-to-tray` | Minimize button ke tray |
| `--start-minimized` | Start app minimized ke tray |

#### Notifications
| Option | Description |
|--------|-------------|
| `--enable-notification` | Enable native Windows toast notifications |

#### Auto-start
| Option | Description |
|--------|-------------|
| `--auto-startup` | Enable auto-start option (adds tray menu checkbox) |

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

#### Advanced
| Option | Description |
|--------|-------------|
| `--no-context-menu` | Disable klik kanan |
| `--no-devtools` | Disable DevTools (F12) |

## Examples

### WhatsApp Desktop (Full Featured)
```bash
w2app create -u https://web.whatsapp.com -n WhatsApp \
  --auto-icon \
  --tray \
  --close-to-tray \
  --single-instance \
  --enable-notification \
  --auto-startup \
  --titlebar-color "#1F2C34"
```

### Telegram Web
```bash
w2app create -u https://web.telegram.org -n Telegram \
  --auto-icon \
  --tray \
  --close-to-tray \
  --single-instance \
  --enable-notification \
  --maximized
```

### Kiosk/Digital Signage
```bash
w2app create -u https://dashboard.example.com -n Dashboard \
  --fullscreen \
  --no-context-menu \
  --no-devtools \
  --auto-icon \
  --single-instance
```

### Custom Dark Mode App
```bash
# dark.css
body { filter: invert(1) hue-rotate(180deg); }
img, video { filter: invert(1) hue-rotate(180deg); }

# Generate
w2app create -u https://example.com -n DarkApp \
  --css-file dark.css \
  --auto-icon \
  --titlebar-color dark
```

### Floating Widget
```bash
w2app create -u https://clock.example.com -n FloatingClock \
  --width 200 \
  --height 100 \
  --frameless \
  --always-on-top \
  --resizable=false
```

### Corporate App dengan Auto-start
```bash
w2app create -u https://app.company.com -n CompanyApp \
  --auto-icon \
  --single-instance \
  --tray \
  --close-to-tray \
  --auto-startup \
  --start-minimized
```

## Icon Sources

W2App mendukung berbagai sumber icon:

1. **File lokal**: `--icon ./myicon.ico`
2. **URL langsung**: `--icon https://example.com/icon.png`
3. **Auto-fetch**: `--auto-icon` (otomatis cari favicon)

Auto-fetch akan mencoba:
- `/favicon.ico`
- `/favicon.png`
- `/apple-touch-icon.png`
- Google Favicon Service
- DuckDuckGo Icon Service

## How It Works

W2App menggunakan arsitektur "stub-based packaging":

1. **Stub**: Pre-compiled webview launcher yang membaca config dari tail binary
2. **Generator**: CLI yang meng-append config JSON ke stub untuk membuat app baru

```
[stub binary] + [marker] + [config JSON] = [final app.exe]
```

### Notification Flow
1. App generates unique AppUserModelID (AUMID): `W2App.{AppName}`
2. App creates Start Menu shortcut with AUMID on first run
3. JS injection overrides `Notification` API in webpage
4. When page calls `new Notification()`, native Windows toast is shown
5. Toast click brings app to foreground

## Output

Aplikasi yang dihasilkan:
- Single executable file (.exe)
- Ukuran ~4MB
- Portable - bisa dipindahkan/disalin ke PC lain
- Tidak perlu Go terinstall di PC target
- Auto-creates Start Menu shortcut (for notifications)

## File Structure

```
web2desktop/
├── cmd/
│   ├── stub/              # Stub launcher source
│   │   └── main.go
│   └── w2app/             # Generator CLI source
│       └── main.go
├── internal/
│   ├── config/            # Shared config struct
│   │   └── config.go
│   └── generator/         # Generator logic
│       ├── generator.go
│       └── stubs/         # Pre-compiled stubs
│           └── stub-windows-amd64.exe
├── build.bat              # Build script
├── go.mod
├── go.sum
├── .gitignore
└── README.md
```

## Troubleshooting

### "Gagal membuat webview" error
WebView2 Runtime tidak terinstall. Download dari:
https://developer.microsoft.com/en-us/microsoft-edge/webview2/

### App tidak berjalan di PC lain
Pastikan target PC memiliki:
- Windows 10/11
- WebView2 Runtime terinstall

### Notifikasi tidak muncul
- Pastikan menggunakan `--enable-notification` flag
- App akan otomatis membuat Start Menu shortcut
- Check Windows notification settings untuk app tersebut
- Restart app setelah shortcut dibuat

### Icon tidak muncul
- Pastikan file/URL icon valid (.ico, .png, .jpg)
- Untuk hasil terbaik, gunakan file .ico dengan multiple sizes
- Coba `--auto-icon` untuk otomatis fetch favicon

### Auto-icon gagal
- Pastikan koneksi internet aktif
- Website mungkin tidak memiliki favicon standard
- Coba gunakan `--icon <url>` dengan URL icon langsung

### Tray icon tidak muncul
- Pastikan menggunakan `--tray` flag
- Check Windows taskbar settings untuk hidden icons

## Dependencies

- [go-webview2](https://github.com/jchv/go-webview2) - WebView2 bindings for Go
- [energye/systray](https://github.com/energye/systray) - System tray with double-click support
- [go-toast](https://github.com/go-toast/toast) - Windows toast notifications
- [winres](https://github.com/tc-hib/winres) - Windows resource embedding

## Roadmap

- [x] System tray support
- [x] Native Windows notifications
- [x] Auto-start on Windows startup
- [x] Custom titlebar color
- [ ] Support Linux (GTK WebKit)
- [ ] Support macOS (WebKit)
- [ ] Auto-update mechanism
- [ ] Offline mode (embed webpage snapshot)

## License

MIT
