package main

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/energye/systray"
	"github.com/go-toast/toast"
	"github.com/jchv/go-webview2"
	"github.com/user/w2app/internal/config"
	"golang.org/x/sys/windows/registry"
)

var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex        = kernel32.NewProc("CreateMutexW")
	procGetModuleHandle    = kernel32.NewProc("GetModuleHandleW")
	procGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")

	user32                       = syscall.NewLazyDLL("user32.dll")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procFindWindowW              = user32.NewProc("FindWindowW")
	procSendMessageW             = user32.NewProc("SendMessageW")
	procLoadImageW               = user32.NewProc("LoadImageW")
	procShowWindow               = user32.NewProc("ShowWindow")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procIsZoomed                 = user32.NewProc("IsZoomed")
	procSetWindowLongPtrW        = user32.NewProc("SetWindowLongPtrW")
	procGetWindowLongPtrW        = user32.NewProc("GetWindowLongPtrW")
	procCallWindowProcW          = user32.NewProc("CallWindowProcW")
	procDefWindowProcW           = user32.NewProc("DefWindowProcW")
	procDestroyIcon              = user32.NewProc("DestroyIcon")
	procGetIconInfo              = user32.NewProc("GetIconInfo")
	procGetDIBits                = user32.NewProc("GetDIBits")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procGetMonitorInfoW          = user32.NewProc("GetMonitorInfoW")
	procMonitorFromWindow        = user32.NewProc("MonitorFromWindow")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procBringWindowToTop         = user32.NewProc("BringWindowToTop")
	procSetFocus                 = user32.NewProc("SetFocus")

	shell32                                     = syscall.NewLazyDLL("shell32.dll")
	procExtractIconEx                           = shell32.NewProc("ExtractIconExW")
	procCoInitializeEx                          = syscall.NewLazyDLL("ole32.dll").NewProc("CoInitializeEx")
	procCoCreateInstance                        = syscall.NewLazyDLL("ole32.dll").NewProc("CoCreateInstance")
	procCoUninitialize                          = syscall.NewLazyDLL("ole32.dll").NewProc("CoUninitialize")
	procSHGetKnownFolderPath                    = shell32.NewProc("SHGetKnownFolderPath")
	procSetCurrentProcessExplicitAppUserModelID = shell32.NewProc("SetCurrentProcessExplicitAppUserModelID")

	gdi32                  = syscall.NewLazyDLL("gdi32.dll")
	procGetObject          = gdi32.NewProc("GetObjectW")
	procGetDIBits2         = gdi32.NewProc("GetDIBits")
	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procDeleteObject       = gdi32.NewProc("DeleteObject")

	dwmapi                    = syscall.NewLazyDLL("dwmapi.dll")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	WM_SETICON     = 0x0080
	WM_CLOSE       = 0x0010
	WM_SYSCOMMAND  = 0x0112
	WM_USER        = 0x0400
	WM_APP_SHOW    = WM_USER + 100 // Custom message to show window
	SC_MINIMIZE    = 0xF020
	SC_CLOSE       = 0xF060
	ICON_SMALL     = 0
	ICON_BIG       = 1
	IMAGE_ICON     = 1
	LR_DEFAULTSIZE = 0x00000040
	LR_SHARED      = 0x00008000
	SW_HIDE        = 0
	SW_SHOW        = 5
	SW_MAXIMIZE    = 3
	SW_RESTORE     = 9
	GWLP_WNDPROC   = -4
	GWL_STYLE      = -16
	GWL_EXSTYLE    = -20

	// Window styles
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_POPUP            = 0x80000000
	WS_VISIBLE          = 0x10000000

	// Extended window styles
	WS_EX_TOPMOST = 0x00000008

	DWMWA_USE_IMMERSIVE_DARK_MODE = 20
	DWMWA_CAPTION_COLOR           = 35
	DWMWA_TEXT_COLOR              = 36

	// Monitor info
	MONITOR_DEFAULTTONEAREST = 0x00000002
)

// Global variables
var (
	appTitle           string
	appConfig          *config.AppConfig
	mainWindow         webview2.WebView
	mainHwnd           uintptr
	isWindowHidden     bool
	originalWndProc    uintptr
	shouldReallyQuit   bool
	windowMutex        sync.Mutex
	wasMaximized       bool   // Track if window was maximized before hiding
	isFullscreenMode   bool   // Track if fullscreen mode is enabled
	startedFromStartup bool   // Track if started from Windows startup
	appUserModelID     string // AppUserModelID for toast notifications
)

func main() {
	// Debug log all startup arguments
	debugLog("=== App started with args: %v ===", os.Args)

	// Check for special arguments
	var showWindow bool
	var notifId string

	for _, arg := range os.Args[1:] {
		switch {
		case arg == "--show" || arg == "/show":
			showWindow = true
			debugLog("Found --show flag")
		case arg == "--startup" || arg == "/startup":
			// Started from Windows startup - should start minimized to tray
			startedFromStartup = true
			debugLog("Found --startup flag")
		case strings.HasPrefix(arg, "--notif-id="):
			notifId = strings.TrimPrefix(arg, "--notif-id=")
			showWindow = true
			debugLog("Found --notif-id=%s", notifId)
		case strings.HasPrefix(arg, "w2app://"):
			// Parse protocol URL: w2app://notification?id=123
			debugLog("Found protocol URL: %s", arg)
			if parsedURL, err := url.Parse(arg); err == nil {
				if parsedURL.Host == "notification" {
					notifId = parsedURL.Query().Get("id")
					showWindow = true
					debugLog("Parsed notification ID from protocol URL: %s", notifId)
				}
			} else {
				debugLog("Failed to parse protocol URL: %v", err)
			}
		}
	}

	// Handle show/notification click - focus existing window and optionally trigger callback
	if showWindow {
		debugLog("showWindow=true, reading config...")
		cfg, err := readEmbeddedConfig()
		if err == nil && cfg.Title != "" {
			debugLog("Focusing existing window: %s", cfg.Title)
			focusExistingWindow(cfg.Title)
			if notifId != "" {
				// Send notification click to existing instance via window message
				debugLog("Sending notification click for id=%s", notifId)
				sendNotificationClick(cfg.Title, notifId)
			}
		} else {
			debugLog("Failed to read config or empty title: err=%v", err)
		}
		return
	}

	// Baca config dari tail binary sendiri
	cfg, err := readEmbeddedConfig()
	if err != nil {
		showError("Gagal membaca konfigurasi: " + err.Error())
		return
	}
	appConfig = cfg

	// Validasi config
	if cfg.URL == "" {
		showError("URL tidak dikonfigurasi")
		return
	}

	// Single instance check
	if cfg.SingleInstance {
		if !acquireLock(cfg.Title) {
			debugLog("Another instance running, focusing existing window")
			focusExistingWindow(cfg.Title)
			// Check if there's a pending notification click (from toast activation)
			if cfg.EnableNotification {
				handlePendingNotificationClick(cfg.Title)
			}
			return
		}
	}

	// Set default values
	if cfg.Width <= 0 {
		cfg.Width = 1024
	}
	if cfg.Height <= 0 {
		cfg.Height = 768
	}
	if cfg.Title == "" {
		cfg.Title = "Web App"
	}

	appTitle = cfg.Title

	// Setup AppUserModelID for toast notifications (must be done early)
	appUserModelID = generateAppUserModelID(cfg.Title)
	setProcessAppUserModelID(appUserModelID)

	// Register custom protocol for toast activation
	if cfg.EnableNotification {
		exePath, _ := os.Executable()
		if err := registerProtocol("w2app", exePath); err != nil {
			debugLog("Warning: failed to register protocol: %v", err)
		}
	}

	// Create Start Menu shortcut with AppUserModelID (required for toast notifications)
	if cfg.EnableNotification {
		if err := ensureStartMenuShortcut(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create Start Menu shortcut: %v\n", err)
		}
	}

	// If tray is enabled, use external loop so webview can run on main thread
	if cfg.EnableTray {
		// Start systray with external loop
		startTray, endTray := systray.RunWithExternalLoop(onTrayReady, onTrayExit)
		startTray()

		// Run webview on main thread (required for Windows)
		runWebView()

		// Cleanup systray after webview closes
		endTray()
	} else {
		// Run without tray
		runWebView()
	}
}

func onTrayReady() {
	// Set tray icon
	iconData := loadTrayIcon()
	systray.SetIcon(iconData)
	systray.SetTitle(appTitle)
	systray.SetTooltip(appTitle)

	// Double-click on tray icon to show window
	systray.SetOnDClick(func(menu systray.IMenu) {
		showMainWindow()
	})

	// Right-click shows the menu (this is default behavior, but we set it explicitly)
	systray.SetOnRClick(func(menu systray.IMenu) {
		menu.ShowMenu()
	})

	// Create menu items
	mShow := systray.AddMenuItem("Show", "Show window")
	mHide := systray.AddMenuItem("Hide", "Hide window")
	systray.AddSeparator()

	// Add auto-startup checkbox if enabled in config
	var mAutoStart *systray.MenuItem
	if appConfig.EnableAutoStart {
		isEnabled := isAutoStartEnabled()
		mAutoStart = systray.AddMenuItemCheckbox("Start on Windows startup", "Start application when Windows starts", isEnabled)
		mAutoStart.Click(func() {
			if mAutoStart.Checked() {
				// Currently checked, so disable it
				if err := setAutoStart(false); err == nil {
					mAutoStart.Uncheck()
				}
			} else {
				// Currently unchecked, so enable it
				if err := setAutoStart(true); err == nil {
					mAutoStart.Check()
				}
			}
		})
		systray.AddSeparator()
	}

	mQuit := systray.AddMenuItem("Exit", "Exit application")

	// Handle menu clicks using Click callback
	mShow.Click(func() {
		showMainWindow()
	})
	mHide.Click(func() {
		hideMainWindow()
	})
	mQuit.Click(func() {
		quitApp()
	})
}

func onTrayExit() {
	// Cleanup
}

func runWebView() {
	cfg := appConfig

	// Buat webview
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     !cfg.DisableDevTools,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  cfg.Title,
			Width:  uint(cfg.Width),
			Height: uint(cfg.Height),
			Center: true,
		},
	})
	if w == nil {
		showError("Gagal membuat webview. Pastikan WebView2 Runtime terinstall.\n\nDownload di: https://developer.microsoft.com/en-us/microsoft-edge/webview2/")
		if cfg.EnableTray {
			systray.Quit()
		}
		return
	}

	mainWindow = w
	mainHwnd = uintptr(w.Window())

	// Set window icon
	setWindowIcon(mainHwnd)

	// Set titlebar color
	if cfg.TitleBarColor != "" {
		setTitleBarColor(mainHwnd, cfg.TitleBarColor)
	}

	// Subclass window if we need to intercept messages:
	// - Close to tray
	// - Minimize to tray
	// - Single instance (to handle WM_APP_SHOW from other instances)
	if (cfg.EnableTray && (cfg.CloseToTray || cfg.MinimizeToTray)) || cfg.SingleInstance {
		subclassWindow(mainHwnd)
	}

	// Start minimized to tray if configured OR if started from Windows startup
	if cfg.EnableTray && (cfg.StartMinimized || startedFromStartup) {
		hideMainWindow()
	}

	// Apply window state: fullscreen or maximized (only if not starting minimized)
	if !cfg.StartMinimized && !startedFromStartup {
		if cfg.Fullscreen {
			isFullscreenMode = true
			setFullscreen(mainHwnd, true)
		} else if cfg.StartMaximized {
			procShowWindow.Call(mainHwnd, SW_MAXIMIZE)
		}
	}

	// Bind functions
	w.Bind("openExternal", func(url string) {
		openBrowser(url)
	})

	w.Bind("toggleFullscreen", func() {})

	// Bind notification functions
	if cfg.EnableNotification {
		// Function to show native toast with app icon
		w.Bind("w2appNotify", func(title, body, icon, notifId, tag string) {
			debugLog("w2appNotify: title=%s, body=%s, id=%s", title, body, notifId)
			go showNativeNotification(title, body, icon, notifId, tag)
		})

		// Function to focus/show window
		w.Bind("w2appFocusWindow", func() {
			debugLog("w2appFocusWindow called")
			showMainWindow()
		})
	}

	// Build init script (notification script is included if enabled)
	initScript := buildInitScript(cfg)
	w.Init(initScript)

	// Start notification click checker
	if cfg.EnableNotification {
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if shouldReallyQuit {
						return
					}
					checkNotificationClick()
				}
			}
		}()
	}

	// Navigate
	w.Navigate(cfg.URL)

	w.Run()

	// After webview closes
	if cfg.ClearCacheOnExit {
		clearWebViewCache()
	}
}

// Window procedure for intercepting close
func customWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	// Handle custom show message from another instance
	if msg == WM_APP_SHOW {
		// Use goroutine to avoid blocking window procedure
		go func() {
			showMainWindow()
		}()
		return 0
	}

	if msg == WM_SYSCOMMAND {
		// SC command is in low-order word of wParam
		cmd := wParam & 0xFFF0

		// Intercept minimize
		if cmd == SC_MINIMIZE && appConfig.MinimizeToTray {
			hideMainWindow()
			return 0
		}
		// Intercept close
		if cmd == SC_CLOSE && appConfig.CloseToTray && !shouldReallyQuit {
			hideMainWindow()
			return 0
		}
	}
	if msg == WM_CLOSE && appConfig.CloseToTray && !shouldReallyQuit {
		hideMainWindow()
		return 0
	}

	// Call original window procedure
	ret, _, _ := procCallWindowProcW.Call(originalWndProc, hwnd, msg, wParam, lParam)
	return ret
}

func subclassWindow(hwnd uintptr) {
	// Get original window procedure
	// GWLP_WNDPROC = -4, but we need to cast it properly for 64-bit
	gwlpWndProc := uintptr(0xFFFFFFFFFFFFFFFC) // -4 as unsigned 64-bit
	originalWndProc, _, _ = procGetWindowLongPtrW.Call(hwnd, gwlpWndProc)

	// Set new window procedure
	newWndProc := syscall.NewCallback(customWndProc)
	procSetWindowLongPtrW.Call(hwnd, gwlpWndProc, newWndProc)
}

func showMainWindow() {
	windowMutex.Lock()
	defer windowMutex.Unlock()

	if mainHwnd == 0 {
		return
	}

	if isWindowHidden {
		// Window is hidden (in tray), show it
		procShowWindow.Call(mainHwnd, SW_SHOW)

		// Restore to correct state: fullscreen, maximized, or normal
		if isFullscreenMode {
			setFullscreen(mainHwnd, true)
		} else if wasMaximized || appConfig.StartMaximized {
			procShowWindow.Call(mainHwnd, SW_MAXIMIZE)
		} else {
			procShowWindow.Call(mainHwnd, SW_RESTORE)
		}
		isWindowHidden = false
	} else {
		// Window is visible, just restore if minimized and bring to front
		// Check if currently maximized to preserve state
		isMaximized, _, _ := procIsZoomed.Call(mainHwnd)
		if isMaximized != 0 || wasMaximized || appConfig.StartMaximized {
			procShowWindow.Call(mainHwnd, SW_MAXIMIZE)
		} else {
			procShowWindow.Call(mainHwnd, SW_RESTORE)
		}
	}

	// Force bring window to foreground
	bringWindowToFront(mainHwnd)
}

// bringWindowToFront forces window to foreground using multiple techniques
func bringWindowToFront(hwnd uintptr) {
	if hwnd == 0 {
		return
	}

	// Get current foreground window's thread
	foregroundHwnd, _, _ := procGetForegroundWindow.Call()
	foregroundThread, _, _ := procGetWindowThreadProcessId.Call(foregroundHwnd, 0)
	currentThread, _, _ := procGetCurrentThreadId.Call()

	// Attach to foreground thread to allow SetForegroundWindow
	if foregroundThread != currentThread {
		procAttachThreadInput.Call(currentThread, foregroundThread, 1)       // Attach
		defer procAttachThreadInput.Call(currentThread, foregroundThread, 0) // Detach
	}

	// Bring window to top
	procBringWindowToTop.Call(hwnd)

	// Set as foreground window
	procSetForegroundWindow.Call(hwnd)

	// Also set focus
	procSetFocus.Call(hwnd)
}

func hideMainWindow() {
	windowMutex.Lock()
	defer windowMutex.Unlock()

	if mainHwnd == 0 || isWindowHidden {
		return
	}

	// Check if window is currently maximized before hiding
	if !isFullscreenMode {
		ret, _, _ := procIsWindowVisible.Call(mainHwnd)
		if ret != 0 {
			// Check if maximized using IsZoomed
			ret, _, _ := procIsZoomed.Call(mainHwnd)
			wasMaximized = ret != 0
		}
	}

	procShowWindow.Call(mainHwnd, SW_HIDE)
	isWindowHidden = true
}

func quitApp() {
	shouldReallyQuit = true
	if mainWindow != nil {
		// Use Dispatch to safely terminate from another goroutine
		mainWindow.Dispatch(func() {
			mainWindow.Terminate()
		})
	} else {
		// If no window, just quit systray
		systray.Quit()
	}
}

func loadTrayIcon() []byte {
	// Try to extract icon from exe using Windows API
	exePath, err := os.Executable()
	if err == nil {
		iconData := extractIconWithAPI(exePath)
		if len(iconData) > 0 {
			return iconData
		}
	}

	// Fallback to simple generated icon
	return createSimpleIcon()
}

// ICONINFO structure
type ICONINFO struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

// BITMAP structure
type BITMAP struct {
	BmType       int32
	BmWidth      int32
	BmHeight     int32
	BmWidthBytes int32
	BmPlanes     uint16
	BmBitsPixel  uint16
	BmBits       uintptr
}

// BITMAPINFOHEADER structure
type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

// RECT structure for fullscreen
type RECT struct {
	Left, Top, Right, Bottom int32
}

// MONITORINFO structure for fullscreen
type MONITORINFO struct {
	CbSize    uint32
	RcMonitor RECT
	RcWork    RECT
	DwFlags   uint32
}

// setFullscreen applies fullscreen mode to the window
func setFullscreen(hwnd uintptr, fullscreen bool) {
	if fullscreen {
		// Get monitor info
		hMonitor, _, _ := procMonitorFromWindow.Call(hwnd, MONITOR_DEFAULTTONEAREST)
		var mi MONITORINFO
		mi.CbSize = uint32(unsafe.Sizeof(mi))
		procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))

		// GWL_STYLE = -16 as unsigned 64-bit
		gwlStyle := uintptr(0xFFFFFFFFFFFFFFF0)

		// Remove window decorations and set fullscreen
		procSetWindowLongPtrW.Call(hwnd, gwlStyle, uintptr(WS_POPUP|WS_VISIBLE))
		procSetWindowPos.Call(hwnd, 0,
			uintptr(mi.RcMonitor.Left),
			uintptr(mi.RcMonitor.Top),
			uintptr(mi.RcMonitor.Right-mi.RcMonitor.Left),
			uintptr(mi.RcMonitor.Bottom-mi.RcMonitor.Top),
			0x0040) // SWP_SHOWWINDOW
	}
}

func extractIconWithAPI(exePath string) []byte {
	exePathPtr, _ := syscall.UTF16PtrFromString(exePath)

	var hIconLarge uintptr
	var hIconSmall uintptr

	// Extract icon from exe
	ret, _, _ := procExtractIconEx.Call(
		uintptr(unsafe.Pointer(exePathPtr)),
		0, // First icon
		uintptr(unsafe.Pointer(&hIconLarge)),
		uintptr(unsafe.Pointer(&hIconSmall)),
		1,
	)

	if ret == 0 {
		return nil
	}

	// Prefer small icon for tray (16x16), fallback to large
	hIcon := hIconSmall
	if hIcon == 0 {
		hIcon = hIconLarge
	}

	if hIcon == 0 {
		return nil
	}

	defer func() {
		if hIconLarge != 0 {
			procDestroyIcon.Call(hIconLarge)
		}
		if hIconSmall != 0 && hIconSmall != hIconLarge {
			procDestroyIcon.Call(hIconSmall)
		}
	}()

	// Get icon info
	var iconInfo ICONINFO
	ret, _, _ = procGetIconInfo.Call(hIcon, uintptr(unsafe.Pointer(&iconInfo)))
	if ret == 0 {
		return nil
	}

	defer func() {
		if iconInfo.HbmColor != 0 {
			procDeleteObject.Call(iconInfo.HbmColor)
		}
		if iconInfo.HbmMask != 0 {
			procDeleteObject.Call(iconInfo.HbmMask)
		}
	}()

	// Get bitmap info
	var bmp BITMAP
	ret, _, _ = procGetObject.Call(iconInfo.HbmColor, unsafe.Sizeof(bmp), uintptr(unsafe.Pointer(&bmp)))
	if ret == 0 {
		return nil
	}

	width := int(bmp.BmWidth)
	height := int(bmp.BmHeight)
	if width == 0 || height == 0 {
		return nil
	}

	// Create DC
	hdc, _, _ := procCreateCompatibleDC.Call(0)
	if hdc == 0 {
		return nil
	}
	defer procDeleteDC.Call(hdc)

	// Prepare BITMAPINFOHEADER for color bitmap
	bih := BITMAPINFOHEADER{
		BiSize:        40,
		BiWidth:       int32(width),
		BiHeight:      int32(height), // Positive = bottom-up
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: 0, // BI_RGB
	}

	// Allocate buffer for color bits
	colorBitsSize := width * height * 4
	colorBits := make([]byte, colorBitsSize)

	// Get color bitmap bits
	ret, _, _ = procGetDIBits2.Call(
		hdc,
		iconInfo.HbmColor,
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&colorBits[0])),
		uintptr(unsafe.Pointer(&bih)),
		0, // DIB_RGB_COLORS
	)
	if ret == 0 {
		return nil
	}

	// Get mask bitmap bits
	maskRowSize := ((width + 31) / 32) * 4
	maskBitsSize := maskRowSize * height
	maskBits := make([]byte, maskBitsSize)

	bihMask := BITMAPINFOHEADER{
		BiSize:     40,
		BiWidth:    int32(width),
		BiHeight:   int32(height),
		BiPlanes:   1,
		BiBitCount: 1,
	}

	procGetDIBits2.Call(
		hdc,
		iconInfo.HbmMask,
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&maskBits[0])),
		uintptr(unsafe.Pointer(&bihMask)),
		0,
	)

	// Build ICO file
	ico := buildICO(width, height, colorBits, maskBits)
	return ico
}

func buildICO(width, height int, colorBits, maskBits []byte) []byte {
	// ICO file structure:
	// - ICONDIR (6 bytes)
	// - ICONDIRENTRY (16 bytes per image)
	// - Image data (BITMAPINFOHEADER + color bits + mask bits)

	colorBitsSize := len(colorBits)
	maskBitsSize := len(maskBits)

	// BITMAPINFOHEADER for ICO (height is doubled)
	bihSize := 40
	imageDataSize := bihSize + colorBitsSize + maskBitsSize

	ico := make([]byte, 0, 22+imageDataSize)

	// ICONDIR
	ico = append(ico, 0, 0) // Reserved
	ico = append(ico, 1, 0) // Type: 1 = ICO
	ico = append(ico, 1, 0) // Count: 1 image

	// ICONDIRENTRY
	w := byte(width)
	h := byte(height)
	if width >= 256 {
		w = 0
	}
	if height >= 256 {
		h = 0
	}
	ico = append(ico, w)     // Width
	ico = append(ico, h)     // Height
	ico = append(ico, 0)     // Color count (0 for 32-bit)
	ico = append(ico, 0)     // Reserved
	ico = append(ico, 1, 0)  // Color planes
	ico = append(ico, 32, 0) // Bits per pixel

	// Image data size (little-endian)
	ico = append(ico,
		byte(imageDataSize),
		byte(imageDataSize>>8),
		byte(imageDataSize>>16),
		byte(imageDataSize>>24),
	)

	// Offset to image data (6 + 16 = 22)
	ico = append(ico, 22, 0, 0, 0)

	// BITMAPINFOHEADER
	bih := make([]byte, 40)
	binary.LittleEndian.PutUint32(bih[0:4], 40)                // biSize
	binary.LittleEndian.PutUint32(bih[4:8], uint32(width))     // biWidth
	binary.LittleEndian.PutUint32(bih[8:12], uint32(height*2)) // biHeight (doubled for XOR+AND)
	binary.LittleEndian.PutUint16(bih[12:14], 1)               // biPlanes
	binary.LittleEndian.PutUint16(bih[14:16], 32)              // biBitCount
	// Rest is zeros (compression, size, etc.)

	ico = append(ico, bih...)
	ico = append(ico, colorBits...)
	ico = append(ico, maskBits...)

	return ico
}

func createSimpleIcon() []byte {
	// Create a 16x16 32-bit ICO file
	width := 16
	height := 16
	bpp := 32

	// Calculate sizes
	pixelDataSize := width * height * (bpp / 8)       // 16*16*4 = 1024 bytes
	andMaskRowSize := ((width + 31) / 32) * 4         // Row size padded to 4 bytes = 4 bytes
	andMaskSize := andMaskRowSize * height            // 4 * 16 = 64 bytes
	imageDataSize := 40 + pixelDataSize + andMaskSize // BITMAPINFOHEADER + pixels + AND mask

	// ICO header (6 bytes)
	ico := []byte{
		0x00, 0x00, // Reserved
		0x01, 0x00, // Type: ICO
		0x01, 0x00, // Number of images: 1
	}

	// Image directory entry (16 bytes)
	dirEntry := []byte{
		byte(width),  // Width (0 means 256)
		byte(height), // Height (0 means 256)
		0x00,         // Color palette size (0 for 32-bit)
		0x00,         // Reserved
		0x01, 0x00,   // Color planes: 1
		byte(bpp), 0x00, // Bits per pixel: 32
	}
	// Image data size (4 bytes, little-endian)
	dirEntry = append(dirEntry,
		byte(imageDataSize),
		byte(imageDataSize>>8),
		byte(imageDataSize>>16),
		byte(imageDataSize>>24),
	)
	// Image data offset (4 bytes, little-endian) = 6 + 16 = 22
	dirEntry = append(dirEntry, 0x16, 0x00, 0x00, 0x00)

	ico = append(ico, dirEntry...)

	// BITMAPINFOHEADER (40 bytes)
	bmpHeader := []byte{
		0x28, 0x00, 0x00, 0x00, // Header size: 40
		byte(width), 0x00, 0x00, 0x00, // Width
		byte(height * 2), 0x00, 0x00, 0x00, // Height (doubled for XOR + AND masks)
		0x01, 0x00, // Planes: 1
		byte(bpp), 0x00, // Bits: 32
		0x00, 0x00, 0x00, 0x00, // Compression: none
		byte(pixelDataSize), byte(pixelDataSize >> 8), 0x00, 0x00, // Image size
		0x00, 0x00, 0x00, 0x00, // X pixels per meter
		0x00, 0x00, 0x00, 0x00, // Y pixels per meter
		0x00, 0x00, 0x00, 0x00, // Colors used
		0x00, 0x00, 0x00, 0x00, // Important colors
	}
	ico = append(ico, bmpHeader...)

	// Pixel data: 16x16 BGRA (bottom-up)
	// Create a nice gradient/solid color icon (WhatsApp green: #25D366)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// BGRA format: Blue, Green, Red, Alpha
			ico = append(ico, 0x66, 0xD3, 0x25, 0xFF) // WhatsApp green
		}
	}

	// AND mask (transparency mask) - all 0 means fully opaque
	for i := 0; i < andMaskSize; i++ {
		ico = append(ico, 0x00)
	}

	return ico
}

// Global variable for notification icon path
var notificationIconPath string

func showNativeNotification(title, body, iconURL, notifId, tag string) {
	if runtime.GOOS != "windows" {
		return
	}

	debugLog("showNativeNotification: title=%s, body=%s, notifId=%s, AUMID=%s", title, body, notifId, appUserModelID)

	if title == "" {
		title = appTitle
	}

	// Ensure AUMID is set
	if appUserModelID == "" {
		appUserModelID = generateAppUserModelID(appTitle)
		debugLog("Generated AUMID: %s", appUserModelID)
	}

	// Build activation URL using custom protocol
	// When user clicks toast, Windows will launch: w2app://notification?id=123
	activationArgs := fmt.Sprintf("w2app://notification?id=%s", notifId)
	debugLog("ActivationArguments: %s", activationArgs)

	notification := toast.Notification{
		AppID:               appUserModelID,
		Title:               title,
		Message:             body,
		Audio:               toast.Default,
		ActivationType:      "protocol", // IMPORTANT: Use "protocol" for URL activation
		ActivationArguments: activationArgs,
	}

	// Try to set icon - extract from exe if not already done
	if notificationIconPath == "" {
		notificationIconPath = extractNotificationIcon()
	}
	if notificationIconPath != "" {
		notification.Icon = notificationIconPath
	}

	debugLog("Pushing notification with AppID=%s, Icon=%s, ActivationType=foreground, ActivationArgs=%s", appUserModelID, notificationIconPath, activationArgs)

	// Save pending notification - this will be read when toast is clicked and app relaunches
	savePendingNotification(appTitle, notifId)

	if err := notification.Push(); err != nil {
		debugLog("Toast error: %v", err)
	} else {
		debugLog("Notification pushed successfully!")
	}
}

// extractNotificationIcon extracts the icon from exe to a temp file for notifications
func extractNotificationIcon() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}

	// Create temp file for icon
	tmpDir := os.TempDir()
	iconPath := filepath.Join(tmpDir, fmt.Sprintf("w2app-%s-icon.png", sanitizeFileName(appTitle)))

	// Check if already exists
	if _, err := os.Stat(iconPath); err == nil {
		return iconPath
	}

	// Extract icon using Windows API and save as PNG
	iconData := extractIconWithAPI(exePath)
	if len(iconData) == 0 {
		return ""
	}

	// Write ICO data to temp file (toast supports ICO)
	icoPath := filepath.Join(tmpDir, fmt.Sprintf("w2app-%s-icon.ico", sanitizeFileName(appTitle)))
	if err := os.WriteFile(icoPath, iconData, 0644); err != nil {
		return ""
	}

	return icoPath
}

// sanitizeFileName removes invalid characters from filename
func sanitizeFileName(name string) string {
	invalid := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

// debugLog writes debug messages to a temp file
func debugLog(format string, args ...interface{}) {
	logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "w2app-debug.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer logFile.Close()
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(logFile, "[%s] %s\n", time.Now().Format("15:04:05"), msg)
}

func buildInitScript(cfg *config.AppConfig) string {
	var scripts []string

	// External link handler
	scripts = append(scripts, `
		document.addEventListener('click', function(e) {
			var target = e.target;
			while (target && target.tagName !== 'A') {
				target = target.parentElement;
			}
			if (target && target.href) {
				try {
					var url = new URL(target.href);
					var currentHost = window.location.hostname;
					if (url.hostname !== currentHost && url.hostname !== '' && url.protocol.startsWith('http')) {
						e.preventDefault();
						openExternal(target.href);
					}
				} catch(e) {}
			}
		}, true);
	`)

	if cfg.DisableContextMenu {
		scripts = append(scripts, `
			document.addEventListener('contextmenu', function(e) {
				e.preventDefault();
				return false;
			});
		`)
	}

	// Add script to intercept notifications and show native toast with app icon
	// Suppress WebView2 native notification, only use go-toast
	if cfg.EnableNotification {
		scripts = append(scripts, `
			(function() {
				var OriginalNotification = window.Notification;
				var pendingNotifications = {};
				var notificationId = 0;
				
				window._w2appHandleNotificationClick = function(id) {
					var data = pendingNotifications[id];
					if (data) {
						if (typeof window.w2appFocusWindow === 'function') {
							window.w2appFocusWindow();
						}
						if (typeof data.onclick === 'function') {
							try {
								data.onclick.call(data.self, { target: data.self });
							} catch(e) {}
						}
						if (data.self && typeof data.self.dispatchEvent === 'function') {
							try {
								data.self.dispatchEvent(new Event('click'));
							} catch(e) {}
						}
					} else {
						if (typeof window.w2appFocusWindow === 'function') {
							window.w2appFocusWindow();
						}
					}
					delete pendingNotifications[id];
				};
				
				function W2AppNotification(title, options) {
					options = options || {};
					var self = this;
					var id = ++notificationId;
					
					this.title = title;
					this.body = options.body || '';
					this.icon = options.icon || '';
					this.tag = options.tag || '';
					this.data = options.data || null;
					this.silent = true;
					this._id = id;
					
					var onclickHandler = null;
					Object.defineProperty(this, 'onclick', {
						get: function() { return onclickHandler; },
						set: function(fn) {
							onclickHandler = fn;
							if (pendingNotifications[id]) {
								pendingNotifications[id].onclick = fn;
							}
						}
					});
					
					pendingNotifications[id] = {
						self: this,
						title: title,
						body: options.body || '',
						tag: options.tag || '',
						onclick: null
					};
					
					if (typeof window.w2appNotify === 'function') {
						window.w2appNotify(title, options.body || '', options.icon || '', id.toString(), options.tag || '');
					}
					
					setTimeout(function() {
						if (typeof self.onshow === 'function') {
							self.onshow();
						}
					}, 100);
					
					setTimeout(function() {
						delete pendingNotifications[id];
					}, 300000);
				}
				
				W2AppNotification.prototype.close = function() {
					delete pendingNotifications[this._id];
					if (typeof this.onclose === 'function') {
						this.onclose();
					}
				};
				W2AppNotification.prototype.addEventListener = function(type, handler) {
					if (type === 'click' && pendingNotifications[this._id]) {
						pendingNotifications[this._id].onclick = handler;
					}
				};
				W2AppNotification.prototype.removeEventListener = function() {};
				W2AppNotification.prototype.dispatchEvent = function(event) {
					if (event.type === 'click' && typeof this.onclick === 'function') {
						this.onclick.call(this, event);
					}
				};
				
				W2AppNotification.permission = 'granted';
				W2AppNotification.maxActions = 2;
				W2AppNotification.requestPermission = function(callback) {
					if (callback) callback('granted');
					return Promise.resolve('granted');
				};
				
				window.Notification = W2AppNotification;
			})();
		`)
	}

	if cfg.InjectCSS != "" {
		escapedCSS := escapeJS(cfg.InjectCSS)
		scripts = append(scripts, fmt.Sprintf(`
			(function() {
				var style = document.createElement('style');
				style.textContent = %q;
				document.head.appendChild(style);
			})();
		`, escapedCSS))
	}

	if cfg.InjectJS != "" {
		scripts = append(scripts, fmt.Sprintf(`
			(function() {
				try { %s } catch(e) {}
			})();
		`, cfg.InjectJS))
	}

	if cfg.UserAgent != "" {
		scripts = append(scripts, fmt.Sprintf(`
			Object.defineProperty(navigator, 'userAgent', {
				get: function() { return %q; }
			});
		`, cfg.UserAgent))
	}

	// Keyboard shortcuts
	scripts = append(scripts, `
		document.addEventListener('keydown', function(e) {
			if (e.key === 'F11') {
				e.preventDefault();
				if (document.fullscreenElement) {
					document.exitFullscreen();
				} else {
					document.documentElement.requestFullscreen();
				}
			}
			if ((e.ctrlKey && e.key === 'r') || e.key === 'F5') {
				e.preventDefault();
				location.reload();
			}
			if (e.ctrlKey && (e.key === '+' || e.key === '=')) {
				e.preventDefault();
				document.body.style.zoom = (parseFloat(document.body.style.zoom) || 1) + 0.1;
			}
			if (e.ctrlKey && e.key === '-') {
				e.preventDefault();
				document.body.style.zoom = Math.max(0.1, (parseFloat(document.body.style.zoom) || 1) - 0.1);
			}
			if (e.ctrlKey && e.key === '0') {
				e.preventDefault();
				document.body.style.zoom = 1;
			}
		});
	`)

	if cfg.Fullscreen {
		scripts = append(scripts, `
			document.addEventListener('DOMContentLoaded', function() {
				setTimeout(function() {
					document.documentElement.requestFullscreen().catch(function(){});
				}, 500);
			});
		`)
	}

	return strings.Join(scripts, "\n")
}

// getNotificationScript returns the notification override script
func getNotificationScript() string {
	return `
		(function() {
			if (window._w2appNotificationInitialized) return;
			window._w2appNotificationInitialized = true;
			
			var pendingNotifications = window._w2appPendingNotifications || {};
			window._w2appPendingNotifications = pendingNotifications;
			var notificationId = window._w2appNotificationId || 0;

			function callNativeNotification(title, body, icon, id, tag) {
				try {
					if (typeof window.w2appNotify === 'function') {
						window.w2appNotify(title || '', body || '', icon || '', id || '', tag || '').catch(function(){});
					}
				} catch(e) {}
			}

			window._w2appHandleNotificationClick = function(id) {
				var n = pendingNotifications[id];
				if (n) {
					var handled = false;
					if (typeof n.onclick === 'function') {
						try {
							var event = {
								target: n,
								currentTarget: n,
								type: 'click',
								preventDefault: function() {},
								stopPropagation: function() {}
							};
							n.onclick.call(n, event);
							handled = true;
						} catch(e) {}
					}
					if (!handled && n.data && n.data.url) {
						window.location.href = n.data.url;
						handled = true;
					}
					if (!handled && n._fromServiceWorker) {
						var customEvent = new CustomEvent('w2app-notification-click', {
							detail: { title: n.title, body: n.body, tag: n.tag, data: n.data }
						});
						window.dispatchEvent(customEvent);
						if (typeof n.close === 'function') n.close();
					}
				}
				delete pendingNotifications[id];
			};

			window._w2appTestNotification = function(title, body) {
				title = title || 'Test Notification';
				body = body || 'This is a test notification from W2App';
				callNativeNotification(title, body, '', '0', '');
			};

			function W2AppNotification(title, options) {
				options = options || {};
				var self = this;
				var id = ++notificationId;
				window._w2appNotificationId = notificationId;
				
				this.title = title;
				this.body = options.body || '';
				this.icon = options.icon || '';
				this.tag = options.tag || '';
				this.data = options.data || null;
				this.onclick = null;
				this.onclose = null;
				this.onerror = null;
				this.onshow = null;
				this._id = id;
				this._fromServiceWorker = false;

				pendingNotifications[id] = this;
				
				var onclickValue = null;
				Object.defineProperty(this, 'onclick', {
					get: function() { return onclickValue; },
					set: function(fn) { onclickValue = fn; }
				});
				
				callNativeNotification(title, this.body, this.icon, id.toString(), this.tag);
				setTimeout(function() { if (typeof self.onshow === 'function') self.onshow(); }, 100);
				setTimeout(function() { delete pendingNotifications[id]; }, 30000);
			}
			W2AppNotification.permission = 'granted';
			W2AppNotification.maxActions = 2;
			W2AppNotification.requestPermission = function(cb) {
				if (cb) cb('granted');
				return Promise.resolve('granted');
			};
			W2AppNotification.prototype.close = function() {
				delete pendingNotifications[this._id];
				if (typeof this.onclose === 'function') this.onclose();
			};
			
			window.Notification = W2AppNotification;

			if (navigator.serviceWorker) {
				ServiceWorkerRegistration.prototype.showNotification = function(title, options) {
					options = options || {};
					var id = ++notificationId;
					window._w2appNotificationId = notificationId;
					
					var notif = { 
						title: title, 
						body: options.body || '', 
						icon: options.icon || '',
						tag: options.tag || '',
						data: options.data || null,
						_id: id,
						_fromServiceWorker: true,
						close: function() {}
					};
					pendingNotifications[id] = notif;
					callNativeNotification(title, options.body || '', options.icon || '', id.toString(), options.tag || '');
					return Promise.resolve();
				};
			}
		})();
	`
}

func escapeJS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

func readEmbeddedConfig() (*config.AppConfig, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("gagal mendapatkan path executable: %w", err)
	}

	data, err := os.ReadFile(exePath)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca executable: %w", err)
	}

	marker := []byte(config.ConfigMarker)
	idx := bytes.LastIndex(data, marker)
	if idx == -1 {
		return nil, fmt.Errorf("config marker tidak ditemukan")
	}

	configData := data[idx+len(marker):]
	configData = bytes.TrimRight(configData, "\x00 \n\r\t")

	var cfg config.AppConfig
	if err := json.Unmarshal(configData, &cfg); err != nil {
		return nil, fmt.Errorf("gagal parse config JSON: %w", err)
	}

	return &cfg, nil
}

func acquireLock(name string) bool {
	if runtime.GOOS != "windows" {
		return true
	}

	mutexName := fmt.Sprintf("Global\\w2app_%x", md5.Sum([]byte(name)))
	mutexNamePtr, _ := syscall.UTF16PtrFromString(mutexName)

	ret, _, err := procCreateMutex.Call(0, 1, uintptr(unsafe.Pointer(mutexNamePtr)))
	if ret == 0 {
		return false
	}

	if err.(syscall.Errno) == 183 {
		return false
	}

	return true
}

func focusExistingWindow(title string) {
	if runtime.GOOS != "windows" {
		return
	}

	titlePtr, _ := syscall.UTF16PtrFromString(title)
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	if hwnd != 0 {
		// Send custom message to the existing window to show itself
		// This allows the main instance to handle showing with correct state (maximized/fullscreen)
		procSendMessageW.Call(hwnd, WM_APP_SHOW, 0, 0)

		// Also set foreground as backup
		procSetForegroundWindow.Call(hwnd)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	cmd.Start()
}

func clearWebViewCache() {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return
	}

	cachePaths := []string{
		filepath.Join(localAppData, "Microsoft", "Edge", "User Data", "Default", "Cache"),
	}

	for _, p := range cachePaths {
		os.RemoveAll(p)
	}
}

func showError(msg string) {
	if runtime.GOOS == "windows" {
		script := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.MessageBox]::Show('%s', 'Error', 'OK', 'Error')`, strings.ReplaceAll(msg, "'", "''"))
		exec.Command("powershell", "-Command", script).Run()
	} else {
		fmt.Fprintln(os.Stderr, "Error:", msg)
	}
}

func setWindowIcon(hwnd uintptr) {
	if runtime.GOOS != "windows" || hwnd == 0 {
		return
	}

	hModule, _, _ := procGetModuleHandle.Call(0)
	if hModule == 0 {
		return
	}

	var hIconBig, hIconSmall uintptr
	iconIDs := []uintptr{1, 3, 100, 101}

	for _, iconID := range iconIDs {
		hIconBig, _, _ = procLoadImageW.Call(hModule, iconID, IMAGE_ICON, 32, 32, LR_SHARED)
		if hIconBig != 0 {
			break
		}
	}

	for _, iconID := range iconIDs {
		hIconSmall, _, _ = procLoadImageW.Call(hModule, iconID, IMAGE_ICON, 16, 16, LR_SHARED)
		if hIconSmall != 0 {
			break
		}
	}

	if hIconBig != 0 {
		procSendMessageW.Call(hwnd, WM_SETICON, ICON_BIG, hIconBig)
	}
	if hIconSmall != 0 {
		procSendMessageW.Call(hwnd, WM_SETICON, ICON_SMALL, hIconSmall)
	}
}

func setTitleBarColor(hwnd uintptr, colorStr string) {
	if runtime.GOOS != "windows" || hwnd == 0 || colorStr == "" {
		return
	}

	switch strings.ToLower(colorStr) {
	case "dark":
		var darkMode int32 = 1
		procDwmSetWindowAttribute.Call(hwnd, DWMWA_USE_IMMERSIVE_DARK_MODE, uintptr(unsafe.Pointer(&darkMode)), unsafe.Sizeof(darkMode))
		return
	case "light":
		var darkMode int32 = 0
		procDwmSetWindowAttribute.Call(hwnd, DWMWA_USE_IMMERSIVE_DARK_MODE, uintptr(unsafe.Pointer(&darkMode)), unsafe.Sizeof(darkMode))
		return
	}

	colorStr = strings.TrimPrefix(colorStr, "#")
	if len(colorStr) != 6 {
		return
	}

	r, err := strconv.ParseUint(colorStr[0:2], 16, 8)
	if err != nil {
		return
	}
	g, err := strconv.ParseUint(colorStr[2:4], 16, 8)
	if err != nil {
		return
	}
	b, err := strconv.ParseUint(colorStr[4:6], 16, 8)
	if err != nil {
		return
	}

	colorRef := uint32(r) | (uint32(g) << 8) | (uint32(b) << 16)
	procDwmSetWindowAttribute.Call(hwnd, DWMWA_CAPTION_COLOR, uintptr(unsafe.Pointer(&colorRef)), unsafe.Sizeof(colorRef))

	luminance := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255.0
	if luminance < 0.5 {
		textColor := uint32(0xFFFFFF)
		procDwmSetWindowAttribute.Call(hwnd, DWMWA_TEXT_COLOR, uintptr(unsafe.Pointer(&textColor)), unsafe.Sizeof(textColor))
	} else {
		textColor := uint32(0x000000)
		procDwmSetWindowAttribute.Call(hwnd, DWMWA_TEXT_COLOR, uintptr(unsafe.Pointer(&textColor)), unsafe.Sizeof(textColor))
	}
}

// sendNotificationClick writes notification ID to a temp file for the main instance to read
func sendNotificationClick(appName, notifId string) {
	notifFile := filepath.Join(os.TempDir(), fmt.Sprintf("w2app-%s-notif.txt", sanitizeFileName(appName)))
	debugLog("sendNotificationClick: writing notifId=%s to %s", notifId, notifFile)
	os.WriteFile(notifFile, []byte(notifId), 0644)
}

// savePendingNotification saves the notification ID when a toast is shown
// This will be read when the toast is clicked and app is relaunched
func savePendingNotification(appName, notifId string) {
	pendingFile := filepath.Join(os.TempDir(), fmt.Sprintf("w2app-%s-pending.txt", sanitizeFileName(appName)))
	debugLog("savePendingNotification: saving notifId=%s to %s", notifId, pendingFile)
	os.WriteFile(pendingFile, []byte(notifId), 0644)
}

// handlePendingNotificationClick checks for pending notification and sends click to main instance
func handlePendingNotificationClick(appName string) {
	pendingFile := filepath.Join(os.TempDir(), fmt.Sprintf("w2app-%s-pending.txt", sanitizeFileName(appName)))
	data, err := os.ReadFile(pendingFile)
	if err != nil {
		debugLog("handlePendingNotificationClick: no pending file found")
		return
	}

	// Remove the pending file
	os.Remove(pendingFile)

	notifId := strings.TrimSpace(string(data))
	if notifId == "" {
		debugLog("handlePendingNotificationClick: pending file empty")
		return
	}

	debugLog("handlePendingNotificationClick: found pending notifId=%s, sending click", notifId)
	sendNotificationClick(appName, notifId)
}

// checkNotificationClick checks if there's a notification click signal from a relaunched instance
// The flow is:
// 1. User clicks toast -> Windows relaunches app (or tries to)
// 2. Second instance detects main instance running, reads pending file, writes to notif file
// 3. Main instance polls notif file here and handles the click
// IMPORTANT: We ONLY process the notif file (written by 2nd instance), NOT the pending file directly
// The pending file is just storage for the notifId - we don't auto-show when it's created
func checkNotificationClick() {
	if mainWindow == nil {
		return
	}

	// Only check the notif file (signal from relaunched instance that toast was clicked)
	notifFile := filepath.Join(os.TempDir(), fmt.Sprintf("w2app-%s-notif.txt", sanitizeFileName(appTitle)))
	data, err := os.ReadFile(notifFile)
	if err != nil {
		// No notif file means no toast was clicked
		return
	}

	// Remove the file immediately to prevent re-processing
	os.Remove(notifFile)

	notifId := strings.TrimSpace(string(data))
	if notifId == "" {
		debugLog("checkNotificationClick: notif file was empty")
		return
	}

	debugLog("checkNotificationClick: found notif file, notifId=%s", notifId)

	// Show window first (toast was clicked, so user wants to see the app)
	showMainWindow()

	// Execute JavaScript to handle notification click
	mainWindow.Dispatch(func() {
		js := fmt.Sprintf("if(window._w2appHandleNotificationClick) window._w2appHandleNotificationClick(%s);", notifId)
		debugLog("checkNotificationClick: executing JS: %s", js)
		mainWindow.Eval(js)
	})
}

// isWindowVisible checks if the main window is currently visible
func isWindowVisible() bool {
	if mainHwnd == 0 {
		return false
	}
	ret, _, _ := procIsWindowVisible.Call(mainHwnd)
	return ret != 0
}

// Registry key for Windows startup
const startupRegistryKey = `Software\Microsoft\Windows\CurrentVersion\Run`

// isAutoStartEnabled checks if auto-start is enabled in Windows registry
func isAutoStartEnabled() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, startupRegistryKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(appTitle)
	return err == nil
}

// setAutoStart enables or disables auto-start in Windows registry
func setAutoStart(enable bool) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, startupRegistryKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if enable {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}
		// Add quotes around path and --startup flag to start minimized to tray
		value := fmt.Sprintf(`"%s" --startup`, exePath)
		return key.SetStringValue(appTitle, value)
	} else {
		return key.DeleteValue(appTitle)
	}
}

// COM GUIDs for shortcut creation
var (
	CLSID_ShellLink      = syscall.GUID{0x00021401, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	IID_IShellLinkW      = syscall.GUID{0x000214F9, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	IID_IPersistFile     = syscall.GUID{0x0000010B, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	IID_IPropertyStore   = syscall.GUID{0x886D8EEB, 0x8CF2, 0x4446, [8]byte{0x8D, 0x02, 0xCD, 0xBA, 0x1D, 0xBD, 0xCF, 0x99}}
	PKEY_AppUserModel_ID = propertyKey{fmtid: syscall.GUID{0x9F4C2855, 0x9F79, 0x4B39, [8]byte{0xA8, 0xD0, 0xE1, 0xD4, 0x2D, 0xE1, 0xD5, 0xF3}}, pid: 5}
	FOLDERID_Programs    = syscall.GUID{0xA77F5D77, 0x2E2B, 0x44C3, [8]byte{0xA6, 0xA2, 0xAB, 0xA6, 0x01, 0x05, 0x4A, 0x51}}
)

type propertyKey struct {
	fmtid syscall.GUID
	pid   uint32
}

// generateAppUserModelID creates a unique AUMID for the app
func generateAppUserModelID(title string) string {
	// Create a valid AUMID: CompanyName.AppName
	// Use sanitized title with "W2App" prefix
	sanitized := sanitizeFileName(title)
	sanitized = strings.ReplaceAll(sanitized, " ", "")
	return fmt.Sprintf("W2App.%s", sanitized)
}

// setProcessAppUserModelID sets the AUMID for the current process
func setProcessAppUserModelID(aumid string) {
	if runtime.GOOS != "windows" {
		return
	}
	aumidPtr, _ := syscall.UTF16PtrFromString(aumid)
	procSetCurrentProcessExplicitAppUserModelID.Call(uintptr(unsafe.Pointer(aumidPtr)))
}

// registerProtocol registers custom URL protocol for toast activation
func registerProtocol(protocol, exePath string) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	debugLog("registerProtocol: registering %s:// protocol", protocol)

	// Open/create protocol key
	key, _, err := registry.CreateKey(registry.CURRENT_USER,
		fmt.Sprintf(`Software\Classes\%s`, protocol),
		registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("failed to create protocol key: %w", err)
	}
	defer key.Close()

	// Set default value and URL Protocol marker
	if err := key.SetStringValue("", fmt.Sprintf("URL:%s Protocol", protocol)); err != nil {
		return err
	}
	if err := key.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}

	// Create command key
	cmdKey, _, err := registry.CreateKey(registry.CURRENT_USER,
		fmt.Sprintf(`Software\Classes\%s\shell\open\command`, protocol),
		registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("failed to create command key: %w", err)
	}
	defer cmdKey.Close()

	// Set command with quoted path and %1 for URL argument
	command := fmt.Sprintf(`"%s" "%%1"`, exePath)
	if err := cmdKey.SetStringValue("", command); err != nil {
		return err
	}

	debugLog("registerProtocol: successfully registered %s:// -> %s", protocol, command)
	return nil
}

// ensureStartMenuShortcut creates Start Menu shortcut with AppUserModelID for toast notifications
func ensureStartMenuShortcut() error {
	if runtime.GOOS != "windows" {
		return nil
	}

	debugLog("ensureStartMenuShortcut: starting...")

	exePath, err := os.Executable()
	if err != nil {
		debugLog("ensureStartMenuShortcut: failed to get exe path: %v", err)
		return err
	}
	debugLog("ensureStartMenuShortcut: exePath=%s", exePath)

	// Get Start Menu Programs folder path
	var pathPtr *uint16
	ret, _, lastErr := procSHGetKnownFolderPath.Call(
		uintptr(unsafe.Pointer(&FOLDERID_Programs)),
		0,
		0,
		uintptr(unsafe.Pointer(&pathPtr)),
	)
	if ret != 0 {
		debugLog("ensureStartMenuShortcut: SHGetKnownFolderPath failed: ret=%x, err=%v", ret, lastErr)
		return fmt.Errorf("failed to get Start Menu path: %x", ret)
	}
	programsPath := syscall.UTF16ToString((*[260]uint16)(unsafe.Pointer(pathPtr))[:])
	debugLog("ensureStartMenuShortcut: programsPath=%s", programsPath)

	// Free the path memory
	syscall.NewLazyDLL("ole32.dll").NewProc("CoTaskMemFree").Call(uintptr(unsafe.Pointer(pathPtr)))

	shortcutPath := filepath.Join(programsPath, sanitizeFileName(appTitle)+".lnk")
	debugLog("ensureStartMenuShortcut: shortcutPath=%s", shortcutPath)

	// Check if shortcut exists and is recent (within 7 days)
	// If old, delete and recreate to ensure protocol support
	if info, err := os.Stat(shortcutPath); err == nil {
		if time.Since(info.ModTime()) < 7*24*time.Hour {
			debugLog("ensureStartMenuShortcut: shortcut already exists and is recent")
			return nil
		}
		// Old shortcut, delete and recreate
		os.Remove(shortcutPath)
		debugLog("ensureStartMenuShortcut: removed old shortcut, will recreate")
	}

	// Initialize COM
	ret, _, lastErr = procCoInitializeEx.Call(0, 0)
	debugLog("ensureStartMenuShortcut: CoInitializeEx ret=%x, err=%v", ret, lastErr)
	defer procCoUninitialize.Call()

	// Create ShellLink object
	var shellLink uintptr
	ret, _, lastErr = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&CLSID_ShellLink)),
		0,
		1, // CLSCTX_INPROC_SERVER
		uintptr(unsafe.Pointer(&IID_IShellLinkW)),
		uintptr(unsafe.Pointer(&shellLink)),
	)
	if ret != 0 {
		debugLog("ensureStartMenuShortcut: CoCreateInstance failed: ret=%x, err=%v", ret, lastErr)
		return fmt.Errorf("failed to create ShellLink: %x", ret)
	}
	debugLog("ensureStartMenuShortcut: ShellLink created: %x", shellLink)
	defer releaseComObject(shellLink)

	// Set shortcut target path
	setPath := getVTableProc(shellLink, 20) // IShellLinkW::SetPath
	exePathPtr, _ := syscall.UTF16PtrFromString(exePath)
	ret, _, _ = syscall.SyscallN(setPath, shellLink, uintptr(unsafe.Pointer(exePathPtr)))
	debugLog("ensureStartMenuShortcut: SetPath ret=%x", ret)

	// Set description
	setDescription := getVTableProc(shellLink, 7) // IShellLinkW::SetDescription
	descPtr, _ := syscall.UTF16PtrFromString(appTitle)
	syscall.SyscallN(setDescription, shellLink, uintptr(unsafe.Pointer(descPtr)))

	// Set working directory
	setWorkingDir := getVTableProc(shellLink, 9) // IShellLinkW::SetWorkingDirectory
	workingDir := filepath.Dir(exePath)
	workingDirPtr, _ := syscall.UTF16PtrFromString(workingDir)
	syscall.SyscallN(setWorkingDir, shellLink, uintptr(unsafe.Pointer(workingDirPtr)))

	// Set icon location (use exe itself)
	setIconLocation := getVTableProc(shellLink, 17) // IShellLinkW::SetIconLocation
	syscall.SyscallN(setIconLocation, shellLink, uintptr(unsafe.Pointer(exePathPtr)), 0)

	// Get IPropertyStore interface to set AppUserModelID
	var propStore uintptr
	queryInterface := getVTableProc(shellLink, 0) // QueryInterface
	ret, _, lastErr = syscall.SyscallN(queryInterface, shellLink, uintptr(unsafe.Pointer(&IID_IPropertyStore)), uintptr(unsafe.Pointer(&propStore)))
	debugLog("ensureStartMenuShortcut: QueryInterface(IPropertyStore) ret=%x, propStore=%x, err=%v", ret, propStore, lastErr)
	if ret == 0 && propStore != 0 {
		defer releaseComObject(propStore)

		// Set AppUserModelID property
		var propVar propVariant
		propVar.vt = 31 // VT_LPWSTR
		aumidPtr, _ := syscall.UTF16PtrFromString(appUserModelID)
		propVar.ptr = uintptr(unsafe.Pointer(aumidPtr))

		setValue := getVTableProc(propStore, 6) // IPropertyStore::SetValue
		ret, _, _ = syscall.SyscallN(setValue, propStore, uintptr(unsafe.Pointer(&PKEY_AppUserModel_ID)), uintptr(unsafe.Pointer(&propVar)))
		debugLog("ensureStartMenuShortcut: SetValue(AUMID=%s) ret=%x", appUserModelID, ret)

		// Commit changes
		commit := getVTableProc(propStore, 7) // IPropertyStore::Commit
		ret, _, _ = syscall.SyscallN(commit, propStore)
		debugLog("ensureStartMenuShortcut: Commit ret=%x", ret)
	}

	// Get IPersistFile interface and save
	var persistFile uintptr
	ret, _, lastErr = syscall.SyscallN(queryInterface, shellLink, uintptr(unsafe.Pointer(&IID_IPersistFile)), uintptr(unsafe.Pointer(&persistFile)))
	if ret != 0 {
		debugLog("ensureStartMenuShortcut: QueryInterface(IPersistFile) failed: ret=%x, err=%v", ret, lastErr)
		return fmt.Errorf("failed to get IPersistFile: %x", ret)
	}
	debugLog("ensureStartMenuShortcut: IPersistFile obtained: %x", persistFile)
	defer releaseComObject(persistFile)

	// Save the shortcut
	save := getVTableProc(persistFile, 6) // IPersistFile::Save
	shortcutPathPtr, _ := syscall.UTF16PtrFromString(shortcutPath)
	ret, _, lastErr = syscall.SyscallN(save, persistFile, uintptr(unsafe.Pointer(shortcutPathPtr)), 1) // TRUE = remember
	if ret != 0 {
		debugLog("ensureStartMenuShortcut: Save failed: ret=%x, err=%v", ret, lastErr)
		return fmt.Errorf("failed to save shortcut: %x", ret)
	}

	debugLog("ensureStartMenuShortcut: shortcut created successfully at %s", shortcutPath)
	return nil
}

type propVariant struct {
	vt       uint16
	reserved [6]byte
	ptr      uintptr
}

func getVTableProc(obj uintptr, index int) uintptr {
	vtable := *(*uintptr)(unsafe.Pointer(obj))
	return *(*uintptr)(unsafe.Pointer(vtable + uintptr(index)*unsafe.Sizeof(uintptr(0))))
}

func releaseComObject(obj uintptr) {
	if obj != 0 {
		release := getVTableProc(obj, 2) // IUnknown::Release
		syscall.SyscallN(release, obj)
	}
}
