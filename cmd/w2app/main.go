package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/user/w2app/internal/generator"
)

var (
	version = "1.2.0"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "create":
		createCmd(os.Args[2:])
	case "platforms":
		listPlatforms()
	case "version", "-v", "--version":
		fmt.Println("w2app version", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		createCmd(os.Args[1:])
	}
}

func createCmd(args []string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)

	// Basic
	url := fs.String("url", "", "URL target (wajib)")
	urlShort := fs.String("u", "", "URL target - shorthand")
	name := fs.String("name", "", "Nama aplikasi (wajib)")
	nameShort := fs.String("n", "", "Nama aplikasi - shorthand")
	output := fs.String("out", ".", "Direktori output")
	outputShort := fs.String("o", "", "Direktori output - shorthand")
	platform := fs.String("platform", "windows", "Platform target")
	platformShort := fs.String("p", "", "Platform target - shorthand")
	icon := fs.String("icon", "", "Path/URL ke icon file (.ico, .png, .jpg)")
	iconShort := fs.String("i", "", "Path/URL ke icon file - shorthand")
	autoIcon := fs.Bool("auto-icon", false, "Auto-fetch favicon dari URL target")

	// Window
	width := fs.Int("width", 1024, "Lebar window")
	height := fs.Int("height", 768, "Tinggi window")
	resizable := fs.Bool("resizable", true, "Window bisa di-resize")
	fullscreen := fs.Bool("fullscreen", false, "Start dalam mode fullscreen")
	frameless := fs.Bool("frameless", false, "Window tanpa frame/border")
	alwaysOnTop := fs.Bool("always-on-top", false, "Window selalu di atas")
	maximized := fs.Bool("maximized", false, "Start dalam kondisi maximized")
	titleBarColor := fs.String("titlebar-color", "", "Warna titlebar (hex: #1a1a2e atau dark/light)")

	// Behavior
	singleInstance := fs.Bool("single-instance", false, "Hanya boleh 1 instance berjalan")
	userAgent := fs.String("user-agent", "", "Custom User-Agent string")
	clearCache := fs.Bool("clear-cache", false, "Hapus cache saat exit")
	enableNotification := fs.Bool("enable-notification", false, "Enable push notifications")

	// System Tray
	enableTray := fs.Bool("tray", false, "Enable system tray icon")
	minimizeToTray := fs.Bool("minimize-to-tray", false, "Minimize to tray instead of taskbar")
	closeToTray := fs.Bool("close-to-tray", false, "Close to tray instead of exit")
	startMinimized := fs.Bool("start-minimized", false, "Start minimized to tray")
	enableAutoStart := fs.Bool("auto-startup", false, "Show auto-startup toggle in tray menu")

	// Injection
	injectCSS := fs.String("inject-css", "", "CSS string untuk di-inject")
	injectJS := fs.String("inject-js", "", "JavaScript string untuk di-inject")
	injectCSSFile := fs.String("css-file", "", "Path ke file CSS untuk di-inject")
	injectJSFile := fs.String("js-file", "", "Path ke file JS untuk di-inject")

	// Navigation
	whitelist := fs.String("whitelist", "", "Domain whitelist (comma-separated)")
	blockExternal := fs.Bool("block-external", false, "Block navigasi ke external URL")

	// Advanced
	disableContextMenu := fs.Bool("no-context-menu", false, "Disable klik kanan")
	disableDevTools := fs.Bool("no-devtools", false, "Disable DevTools (F12)")

	fs.Usage = func() {
		fmt.Println("Usage: w2app create [options]")
		fmt.Println("\nOptions:")
		fmt.Println("\n  BASIC:")
		fmt.Println("    --url, -u          URL target (wajib)")
		fmt.Println("    --name, -n         Nama aplikasi (wajib)")
		fmt.Println("    --icon, -i         Path/URL ke icon file (.ico, .png, .jpg)")
		fmt.Println("    --auto-icon        Auto-fetch favicon dari URL target")
		fmt.Println("    --out, -o          Direktori output (default: .)")
		fmt.Println("    --platform, -p     Platform target (default: windows)")
		fmt.Println("\n  WINDOW:")
		fmt.Println("    --width            Lebar window (default: 1024)")
		fmt.Println("    --height           Tinggi window (default: 768)")
		fmt.Println("    --resizable        Window bisa di-resize (default: true)")
		fmt.Println("    --fullscreen       Start dalam mode fullscreen")
		fmt.Println("    --frameless        Window tanpa frame/border")
		fmt.Println("    --always-on-top    Window selalu di atas")
		fmt.Println("    --maximized        Start dalam kondisi maximized")
		fmt.Println("    --titlebar-color   Warna titlebar (hex: #1a1a2e atau dark/light)")
		fmt.Println("\n  BEHAVIOR:")
		fmt.Println("    --single-instance    Hanya boleh 1 instance berjalan")
		fmt.Println("    --user-agent         Custom User-Agent string")
		fmt.Println("    --clear-cache        Hapus cache saat exit")
		fmt.Println("    --enable-notification Enable push notifications (Windows toast)")
		fmt.Println("\n  SYSTEM TRAY:")
		fmt.Println("    --tray               Enable system tray icon")
		fmt.Println("    --minimize-to-tray   Minimize to tray instead of taskbar")
		fmt.Println("    --close-to-tray      Close to tray instead of exit")
		fmt.Println("    --start-minimized    Start minimized to tray")
		fmt.Println("    --auto-startup       Show auto-startup toggle in tray menu")
		fmt.Println("\n  INJECTION:")
		fmt.Println("    --inject-css       CSS string untuk di-inject")
		fmt.Println("    --inject-js        JavaScript string untuk di-inject")
		fmt.Println("    --css-file         Path ke file CSS untuk di-inject")
		fmt.Println("    --js-file          Path ke file JS untuk di-inject")
		fmt.Println("\n  NAVIGATION:")
		fmt.Println("    --whitelist        Domain whitelist (comma-separated)")
		fmt.Println("    --block-external   Block navigasi ke external URL")
		fmt.Println("\n  ADVANCED:")
		fmt.Println("    --no-context-menu  Disable klik kanan")
		fmt.Println("    --no-devtools      Disable DevTools (F12)")
		fmt.Println("\nKeyboard Shortcuts (dalam app):")
		fmt.Println("    F11                Toggle fullscreen")
		fmt.Println("    F5 / Ctrl+R        Refresh")
		fmt.Println("    Ctrl+Plus          Zoom in")
		fmt.Println("    Ctrl+Minus         Zoom out")
		fmt.Println("    Ctrl+0             Reset zoom")
		fmt.Println("\nExamples:")
		fmt.Println("  w2app create --url https://google.com --name GoogleApp --auto-icon")
		fmt.Println("  w2app -u https://github.com -n GitHub --icon https://github.com/favicon.ico")
		fmt.Println("  w2app --url https://app.slack.com --name Slack --icon slack.png")
		fmt.Println("  w2app -u https://web.whatsapp.com -n WhatsApp --single-instance --auto-icon")
		fmt.Println("  w2app -u https://web.whatsapp.com -n WhatsApp --tray --close-to-tray --auto-icon")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Merge shorthand flags
	finalURL := *url
	if *urlShort != "" {
		finalURL = *urlShort
	}

	finalName := *name
	if *nameShort != "" {
		finalName = *nameShort
	}

	finalOutput := *output
	if *outputShort != "" {
		finalOutput = *outputShort
	}

	finalPlatform := *platform
	if *platformShort != "" {
		finalPlatform = *platformShort
	}

	finalIcon := *icon
	if *iconShort != "" {
		finalIcon = *iconShort
	}

	// Validasi
	if finalURL == "" {
		fmt.Println("Error: URL wajib diisi (--url atau -u)")
		fmt.Println()
		fs.Usage()
		os.Exit(1)
	}

	if finalName == "" {
		fmt.Println("Error: Nama aplikasi wajib diisi (--name atau -n)")
		fmt.Println()
		fs.Usage()
		os.Exit(1)
	}

	// Parse whitelist
	var whitelistDomains []string
	if *whitelist != "" {
		for _, domain := range strings.Split(*whitelist, ",") {
			domain = strings.TrimSpace(domain)
			if domain != "" {
				whitelistDomains = append(whitelistDomains, domain)
			}
		}
	}

	// Generate aplikasi
	opts := generator.Options{
		URL:                finalURL,
		Name:               finalName,
		Output:             finalOutput,
		Platform:           strings.ToLower(finalPlatform),
		Icon:               finalIcon,
		AutoIcon:           *autoIcon,
		Width:              *width,
		Height:             *height,
		Resizable:          *resizable,
		Fullscreen:         *fullscreen,
		Frameless:          *frameless,
		AlwaysOnTop:        *alwaysOnTop,
		StartMaximized:     *maximized,
		TitleBarColor:      *titleBarColor,
		SingleInstance:     *singleInstance,
		UserAgent:          *userAgent,
		ClearCacheOnExit:   *clearCache,
		EnableNotification: *enableNotification,
		EnableTray:         *enableTray,
		MinimizeToTray:     *minimizeToTray,
		CloseToTray:        *closeToTray,
		StartMinimized:     *startMinimized,
		EnableAutoStart:    *enableAutoStart,
		InjectCSS:          *injectCSS,
		InjectJS:           *injectJS,
		InjectCSSFile:      *injectCSSFile,
		InjectJSFile:       *injectJSFile,
		Whitelist:          whitelistDomains,
		BlockExternalNav:   *blockExternal,
		DisableContextMenu: *disableContextMenu,
		DisableDevTools:    *disableDevTools,
	}

	if err := generator.Generate(opts); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func listPlatforms() {
	platforms := generator.ListPlatforms()
	if len(platforms) == 0 {
		fmt.Println("Tidak ada platform yang tersedia.")
		return
	}

	fmt.Println("Platform yang tersedia:")
	for _, p := range platforms {
		fmt.Printf("  - %s\n", p)
	}
}

func printUsage() {
	fmt.Println("w2app - Web to Desktop App Generator v" + version)
	fmt.Println()
	fmt.Println("Convert any website/URL into a standalone desktop application.")
	fmt.Println()
	fmt.Println("Usage: w2app <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  create     Buat aplikasi desktop dari URL")
	fmt.Println("  platforms  Tampilkan daftar platform yang tersedia")
	fmt.Println("  version    Tampilkan versi aplikasi")
	fmt.Println("  help       Tampilkan bantuan")
	fmt.Println()
	fmt.Println("Quick Usage:")
	fmt.Println("  w2app --url <URL> --name <AppName>")
	fmt.Println("  w2app --url <URL> --name <AppName> --auto-icon")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  w2app --url https://google.com --name GoogleApp --auto-icon")
	fmt.Println("  w2app -u https://github.com -n GitHub --icon https://github.com/favicon.ico")
	fmt.Println("  w2app --url https://web.whatsapp.com --name WhatsApp --single-instance")
	fmt.Println()
	fmt.Println("For more options:")
	fmt.Println("  w2app create --help")
}
