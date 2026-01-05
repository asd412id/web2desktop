package generator

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
	"github.com/user/w2app/internal/config"
)

//go:embed stubs/*
var stubsFS embed.FS

// Options adalah opsi untuk generate aplikasi
type Options struct {
	// Basic
	URL      string
	Name     string
	Output   string
	Platform string
	Icon     string // Path ke icon file (.ico, .png, .jpg) atau URL

	// Window
	Width          int
	Height         int
	Resizable      bool
	Fullscreen     bool
	Frameless      bool
	AlwaysOnTop    bool
	StartMaximized bool
	TitleBarColor  string // Hex color e.g. "#1a1a2e" or "dark"/"light"

	// Behavior
	SingleInstance     bool
	UserAgent          string
	ClearCacheOnExit   bool
	EnableNotification bool // Enable push notifications

	// System Tray
	EnableTray      bool // Enable system tray icon
	MinimizeToTray  bool // Minimize to tray instead of taskbar
	CloseToTray     bool // Close to tray instead of exit
	StartMinimized  bool // Start minimized to tray
	EnableAutoStart bool // Show auto-start toggle in tray menu

	// Injection
	InjectCSS     string
	InjectJS      string
	InjectCSSFile string
	InjectJSFile  string

	// Navigation
	Whitelist        []string
	BlockExternalNav bool

	// Advanced
	DisableContextMenu bool
	DisableDevTools    bool
	AutoIcon           bool // Auto-fetch favicon dari URL target
}

// HTTP client dengan timeout
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// Generate membuat aplikasi webview dari URL
func Generate(opts Options) error {
	// Validasi URL
	if opts.URL == "" {
		return fmt.Errorf("URL tidak boleh kosong")
	}

	// Validasi dan normalisasi URL
	parsedURL, err := url.Parse(opts.URL)
	if err != nil {
		return fmt.Errorf("URL tidak valid: %w", err)
	}

	// Tambahkan https:// jika tidak ada scheme
	if parsedURL.Scheme == "" {
		opts.URL = "https://" + opts.URL
		parsedURL, _ = url.Parse(opts.URL)
	}

	// Validasi name
	if opts.Name == "" {
		return fmt.Errorf("nama aplikasi tidak boleh kosong")
	}

	// Sanitize name untuk filename
	safeName := sanitizeFilename(opts.Name)
	if safeName == "" {
		return fmt.Errorf("nama aplikasi tidak valid")
	}

	// Set defaults
	if opts.Width <= 0 {
		opts.Width = 1024
	}
	if opts.Height <= 0 {
		opts.Height = 768
	}
	if opts.Output == "" {
		opts.Output = "."
	}
	if opts.Platform == "" {
		opts.Platform = "windows"
	}

	// Auto-fetch favicon jika --auto-icon dan tidak ada icon yang di-set
	if opts.AutoIcon && opts.Icon == "" {
		fmt.Print("  Fetching favicon...")
		iconPath, err := fetchFavicon(parsedURL)
		if err != nil {
			fmt.Printf(" gagal: %v\n", err)
		} else {
			opts.Icon = iconPath
			fmt.Printf(" OK\n")
			defer os.Remove(iconPath) // Cleanup temp file
		}
	}

	// Download icon jika berupa URL
	if opts.Icon != "" && isURL(opts.Icon) {
		fmt.Print("  Downloading icon...")
		iconPath, err := downloadIcon(opts.Icon)
		if err != nil {
			fmt.Printf(" gagal: %v\n", err)
			opts.Icon = "" // Reset icon jika gagal download
		} else {
			opts.Icon = iconPath
			fmt.Printf(" OK\n")
			defer os.Remove(iconPath) // Cleanup temp file
		}
	}

	// Load CSS from file if specified
	injectCSS := opts.InjectCSS
	if opts.InjectCSSFile != "" {
		cssData, err := os.ReadFile(opts.InjectCSSFile)
		if err != nil {
			return fmt.Errorf("gagal membaca CSS file: %w", err)
		}
		injectCSS = string(cssData)
	}

	// Load JS from file if specified
	injectJS := opts.InjectJS
	if opts.InjectJSFile != "" {
		jsData, err := os.ReadFile(opts.InjectJSFile)
		if err != nil {
			return fmt.Errorf("gagal membaca JS file: %w", err)
		}
		injectJS = string(jsData)
	}

	// Tentukan stub file
	stubFile := fmt.Sprintf("stubs/stub-%s-amd64.exe", opts.Platform)
	if opts.Platform != "windows" {
		stubFile = fmt.Sprintf("stubs/stub-%s-amd64", opts.Platform)
	}

	// Baca stub dari embedded FS
	stubData, err := stubsFS.ReadFile(stubFile)
	if err != nil {
		return fmt.Errorf("stub untuk platform '%s' tidak tersedia: %w", opts.Platform, err)
	}

	// Buat config
	cfg := config.AppConfig{
		URL:                opts.URL,
		Title:              opts.Name,
		Width:              opts.Width,
		Height:             opts.Height,
		Resizable:          opts.Resizable,
		Fullscreen:         opts.Fullscreen,
		Frameless:          opts.Frameless,
		AlwaysOnTop:        opts.AlwaysOnTop,
		StartMaximized:     opts.StartMaximized,
		TitleBarColor:      opts.TitleBarColor,
		SingleInstance:     opts.SingleInstance,
		UserAgent:          opts.UserAgent,
		ClearCacheOnExit:   opts.ClearCacheOnExit,
		EnableNotification: opts.EnableNotification,
		EnableTray:         opts.EnableTray,
		MinimizeToTray:     opts.MinimizeToTray,
		CloseToTray:        opts.CloseToTray,
		StartMinimized:     opts.StartMinimized,
		EnableAutoStart:    opts.EnableAutoStart,
		InjectCSS:          injectCSS,
		InjectJS:           injectJS,
		Whitelist:          opts.Whitelist,
		BlockExternalNav:   opts.BlockExternalNav,
		DisableContextMenu: opts.DisableContextMenu,
		DisableDevTools:    opts.DisableDevTools,
	}

	// Serialize config ke JSON
	configJSON, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("gagal serialize config: %w", err)
	}

	// Buat output directory jika belum ada
	if err := os.MkdirAll(opts.Output, 0755); err != nil {
		return fmt.Errorf("gagal membuat output directory: %w", err)
	}

	// Tentukan nama file output
	outFileName := safeName
	if opts.Platform == "windows" {
		outFileName += ".exe"
	}
	outPath := filepath.Join(opts.Output, outFileName)

	// Buat file output dengan stub
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("gagal membuat file output: %w", err)
	}

	// Tulis stub binary
	if _, err := outFile.Write(stubData); err != nil {
		outFile.Close()
		return fmt.Errorf("gagal menulis stub: %w", err)
	}
	outFile.Close()

	// PENTING: Embed icon SEBELUM append marker+JSON
	// winres.WriteObject memodifikasi PE headers/resources, jadi harus dilakukan
	// sebelum kita append data tambahan di akhir file
	iconEmbedded := false
	iconSource := ""
	if opts.Icon != "" && opts.Platform == "windows" {
		if err := embedIcon(outPath, opts.Icon, opts.Name); err != nil {
			fmt.Printf("  Warning: Gagal embed icon: %v\n", err)
		} else {
			iconEmbedded = true
			iconSource = opts.Icon
		}
	}

	// Buka file untuk append marker dan config
	outFile, err = os.OpenFile(outPath, os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("gagal membuka file untuk append config: %w", err)
	}

	// Tulis marker
	if _, err := outFile.WriteString(config.ConfigMarker); err != nil {
		outFile.Close()
		return fmt.Errorf("gagal menulis marker: %w", err)
	}

	// Tulis config JSON
	if _, err := outFile.Write(configJSON); err != nil {
		outFile.Close()
		return fmt.Errorf("gagal menulis config: %w", err)
	}

	// Get file size before closing
	fileSize := getFileSize(outFile)
	outFile.Close()

	// Print summary
	fmt.Printf("\n✓ Berhasil membuat aplikasi: %s\n", outPath)
	fmt.Printf("  URL       : %s\n", opts.URL)
	fmt.Printf("  Ukuran    : %s\n", fileSize)
	fmt.Printf("  Window    : %dx%d", opts.Width, opts.Height)
	if opts.Resizable {
		fmt.Print(" (resizable)")
	}
	if opts.Fullscreen {
		fmt.Print(" (fullscreen)")
	}
	fmt.Println()

	if opts.SingleInstance {
		fmt.Println("  Mode      : Single instance")
	}
	if iconEmbedded {
		fmt.Printf("  Icon      : %s (embedded)\n", iconSource)
	}
	if injectCSS != "" {
		fmt.Printf("  CSS       : %d bytes injected\n", len(injectCSS))
	}
	if injectJS != "" {
		fmt.Printf("  JS        : %d bytes injected\n", len(injectJS))
	}
	fmt.Println()

	return nil
}

// isURL mengecek apakah string adalah URL
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// downloadIcon mendownload icon dari URL ke file temporary
func downloadIcon(iconURL string) (string, error) {
	resp, err := httpClient.Get(iconURL)
	if err != nil {
		return "", fmt.Errorf("gagal download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	// Tentukan ekstensi dari Content-Type atau URL
	ext := getIconExtension(resp.Header.Get("Content-Type"), iconURL)

	// Buat temp file
	tmpFile, err := os.CreateTemp("", "w2app-icon-*"+ext)
	if err != nil {
		return "", fmt.Errorf("gagal buat temp file: %w", err)
	}
	defer tmpFile.Close()

	// Copy data
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("gagal write file: %w", err)
	}

	return tmpFile.Name(), nil
}

// fetchFavicon mencoba mendapatkan favicon dari website
func fetchFavicon(siteURL *url.URL) (string, error) {
	baseURL := fmt.Sprintf("%s://%s", siteURL.Scheme, siteURL.Host)

	// Coba berbagai lokasi favicon umum
	faviconURLs := []string{
		baseURL + "/favicon.ico",
		baseURL + "/favicon.png",
		baseURL + "/apple-touch-icon.png",
		baseURL + "/apple-touch-icon-precomposed.png",
		fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=128", siteURL.Host),
		fmt.Sprintf("https://icons.duckduckgo.com/ip3/%s.ico", siteURL.Host),
	}

	for _, faviconURL := range faviconURLs {
		iconPath, err := downloadIcon(faviconURL)
		if err == nil {
			// Verifikasi bahwa file adalah image yang valid
			if isValidImage(iconPath) {
				return iconPath, nil
			}
			os.Remove(iconPath)
		}
	}

	return "", fmt.Errorf("tidak dapat menemukan favicon")
}

// isValidImage mengecek apakah file adalah image yang valid
func isValidImage(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Baca header untuk deteksi format
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil || n < 4 {
		return false
	}

	// Cek magic bytes untuk format umum
	// ICO: 00 00 01 00
	if header[0] == 0x00 && header[1] == 0x00 && header[2] == 0x01 && header[3] == 0x00 {
		return true
	}
	// PNG: 89 50 4E 47
	if header[0] == 0x89 && header[1] == 0x50 && header[2] == 0x4E && header[3] == 0x47 {
		return true
	}
	// JPEG: FF D8 FF
	if header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF {
		return true
	}
	// GIF: 47 49 46 38
	if header[0] == 0x47 && header[1] == 0x49 && header[2] == 0x46 && header[3] == 0x38 {
		return true
	}
	// BMP: 42 4D
	if header[0] == 0x42 && header[1] == 0x4D {
		return true
	}

	return false
}

// getIconExtension menentukan ekstensi file berdasarkan content-type atau URL
func getIconExtension(contentType, iconURL string) string {
	// Dari Content-Type
	switch {
	case strings.Contains(contentType, "image/x-icon"), strings.Contains(contentType, "image/vnd.microsoft.icon"):
		return ".ico"
	case strings.Contains(contentType, "image/png"):
		return ".png"
	case strings.Contains(contentType, "image/jpeg"):
		return ".jpg"
	case strings.Contains(contentType, "image/gif"):
		return ".gif"
	case strings.Contains(contentType, "image/bmp"):
		return ".bmp"
	}

	// Dari URL
	lowerURL := strings.ToLower(iconURL)
	if strings.HasSuffix(lowerURL, ".ico") {
		return ".ico"
	}
	if strings.HasSuffix(lowerURL, ".png") {
		return ".png"
	}
	if strings.HasSuffix(lowerURL, ".jpg") || strings.HasSuffix(lowerURL, ".jpeg") {
		return ".jpg"
	}
	if strings.HasSuffix(lowerURL, ".gif") {
		return ".gif"
	}

	// Default
	return ".ico"
}

// embedIcon menambahkan icon ke executable Windows menggunakan WriteToEXE
func embedIcon(exePath, iconPath, appName string) error {
	// Baca icon file
	iconData, err := os.ReadFile(iconPath)
	if err != nil {
		return fmt.Errorf("gagal membaca icon file: %w", err)
	}

	// Buat winres resource set
	rs := winres.ResourceSet{}

	// Cek format berdasarkan magic bytes
	ext := detectImageFormat(iconData)

	// Use ID 1 for the main application icon (this is the standard convention)
	iconResID := winres.ID(1)

	if ext == ".ico" {
		// Load .ico file langsung
		ico, err := winres.LoadICO(bytes.NewReader(iconData))
		if err != nil {
			return fmt.Errorf("gagal load ICO: %w", err)
		}
		rs.SetIcon(iconResID, ico)
	} else {
		// Load sebagai image (png, jpg, dll)
		img, _, err := image.Decode(bytes.NewReader(iconData))
		if err != nil {
			return fmt.Errorf("gagal decode image: %w", err)
		}

		// Konversi image ke icon dengan multiple sizes untuk kualitas lebih baik
		ico, err := winres.NewIconFromResizedImage(img, nil)
		if err != nil {
			return fmt.Errorf("gagal konversi ke icon: %w", err)
		}
		rs.SetIcon(iconResID, ico)
	}

	// Set version info
	vi := version.Info{
		ProductVersion: [4]uint16{1, 0, 0, 0},
		FileVersion:    [4]uint16{1, 0, 0, 0},
	}
	vi.Set(0x0409, "ProductName", appName)
	vi.Set(0x0409, "FileDescription", appName+" - Web App")
	vi.Set(0x0409, "CompanyName", "W2App")
	vi.Set(0x0409, "LegalCopyright", "Generated by W2App")
	vi.Set(0x0409, "OriginalFilename", filepath.Base(exePath))
	rs.SetVersionInfo(vi)

	// Buka source exe untuk dibaca
	srcFile, err := os.Open(exePath)
	if err != nil {
		return fmt.Errorf("gagal membuka exe source: %w", err)
	}

	// Buat temp file untuk output
	tmpFile, err := os.CreateTemp(filepath.Dir(exePath), "w2app-tmp-*.exe")
	if err != nil {
		srcFile.Close()
		return fmt.Errorf("gagal buat temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// WriteToEXE: tulis exe baru dengan resources yang ditambahkan
	err = rs.WriteToEXE(tmpFile, srcFile)
	srcFile.Close()
	tmpFile.Close()

	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("gagal write resources ke exe: %w", err)
	}

	// Ganti file asli dengan file yang sudah dimodifikasi
	if err := os.Remove(exePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("gagal hapus exe asli: %w", err)
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		return fmt.Errorf("gagal rename temp ke exe: %w", err)
	}

	return nil
}

// detectImageFormat mendeteksi format image dari magic bytes
func detectImageFormat(data []byte) string {
	if len(data) < 4 {
		return ""
	}

	// ICO: 00 00 01 00
	if data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01 && data[3] == 0x00 {
		return ".ico"
	}
	// PNG: 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return ".png"
	}
	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return ".jpg"
	}
	// GIF: 47 49 46 38
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return ".gif"
	}

	return ".png" // default
}

// ListPlatforms mengembalikan daftar platform yang tersedia
func ListPlatforms() []string {
	entries, err := stubsFS.ReadDir("stubs")
	if err != nil {
		return []string{}
	}

	var platforms []string
	for _, entry := range entries {
		name := entry.Name()
		parts := strings.Split(name, "-")
		if len(parts) >= 2 {
			platforms = append(platforms, parts[1])
		}
	}
	return platforms
}

// sanitizeFilename membersihkan nama file dari karakter yang tidak valid
func sanitizeFilename(name string) string {
	reg := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	safe := reg.ReplaceAllString(name, "")
	safe = strings.TrimSpace(safe)
	reg = regexp.MustCompile(`\s+`)
	safe = reg.ReplaceAllString(safe, " ")
	return safe
}

// getFileSize mengembalikan ukuran file dalam format human-readable
func getFileSize(f *os.File) string {
	info, err := f.Stat()
	if err != nil {
		return "unknown"
	}
	return formatBytes(info.Size())
}

// formatBytes mengembalikan ukuran dalam format human-readable
func formatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// formatSize mengembalikan ukuran file dalam format human-readable (deprecated, use getFileSize)
func formatSize(f *os.File) string {
	return getFileSize(f)
}

// CopyFile menyalin file dari src ke dst
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
