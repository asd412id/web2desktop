package config

// AppConfig adalah konfigurasi yang di-embed ke dalam binary hasil generate
type AppConfig struct {
	// Basic
	URL   string `json:"url"`
	Title string `json:"title"`

	// Window
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	Resizable      bool   `json:"resizable"`
	Fullscreen     bool   `json:"fullscreen,omitempty"`
	Frameless      bool   `json:"frameless,omitempty"`
	AlwaysOnTop    bool   `json:"always_on_top,omitempty"`
	StartMaximized bool   `json:"start_maximized,omitempty"`
	TitleBarColor  string `json:"titlebar_color,omitempty"` // Hex color e.g. "#1a1a2e" or "dark"

	// Behavior
	SingleInstance     bool   `json:"single_instance,omitempty"`
	UserAgent          string `json:"user_agent,omitempty"`
	ClearCacheOnExit   bool   `json:"clear_cache_on_exit,omitempty"`
	EnableNotification bool   `json:"enable_notification,omitempty"` // Enable push notifications

	// System Tray
	EnableTray      bool `json:"enable_tray,omitempty"`       // Enable system tray icon
	MinimizeToTray  bool `json:"minimize_to_tray,omitempty"`  // Minimize to tray instead of taskbar
	CloseToTray     bool `json:"close_to_tray,omitempty"`     // Close to tray instead of exit
	StartMinimized  bool `json:"start_minimized,omitempty"`   // Start minimized to tray
	EnableAutoStart bool `json:"enable_auto_start,omitempty"` // Show auto-start toggle in tray menu

	// Injection
	InjectCSS string `json:"inject_css,omitempty"`
	InjectJS  string `json:"inject_js,omitempty"`

	// Navigation
	Whitelist        []string `json:"whitelist,omitempty"`
	BlockExternalNav bool     `json:"block_external_nav,omitempty"`

	// Advanced
	DisableContextMenu bool `json:"disable_context_menu,omitempty"`
	DisableDevTools    bool `json:"disable_devtools,omitempty"`
}

// ConfigMarker adalah marker unik untuk menemukan config di tail binary
const ConfigMarker = "\n---W2APP_CONFIG_V1---\n"
