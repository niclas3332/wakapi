package routes

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/middlewares"
	"github.com/muety/wakapi/services"
)

const (
	// appShellPath is the location of the built SPA entry document inside the static fs.
	appShellPath = "assets/app/index.html"
	// appShellMarker is replaced with the server-injected head block. It must be the
	// first child of <head> in frontend/index.html.
	appShellMarker = "<!--wakapi:head-->"
	// appAssetBase is where the SPA's hashed chunks live, relative to the deployment base path.
	appAssetBase = "/assets/app/"
	// appMountPrefix stages the SPA next to the legacy pages while both exist. It becomes
	// "" at cutover, when the legacy handlers are removed and the SPA claims the real paths.
	appMountPrefix = "/app"
)

var errShellMarkerMissing = errors.New("spa shell is missing the " + appShellMarker + " marker")

// appRoutes are the paths owned by the single-page app. Each is registered for the
// exact path and for everything below it, so client-side routes survive a reload.
// Deliberately an explicit list: a catch-all would turn every typo into a 200.
var appRoutes = []string{
	"/summary",
	"/projects",
	"/settings",
	"/insights",
	"/wrapped",
	"/ai",
}

// appShell is the pre-split entry document: everything before the marker and
// everything after it, so a request only has to write three byte slices.
type appShell struct {
	head []byte
	tail []byte
}

type AppHandler struct {
	config       *conf.Config
	userSrvc     services.IUserService
	brandingSrvc services.IBrandingService
	static       fs.FS

	mu    sync.RWMutex
	shell *appShell
}

func NewAppHandler(userService services.IUserService, brandingService services.IBrandingService, static fs.FS) *AppHandler {
	h := &AppHandler{
		config:       conf.Get(),
		userSrvc:     userService,
		brandingSrvc: brandingService,
		static:       static,
	}
	// A missing build must not stop the server from booting - ServeShell reports it.
	if err := h.loadShell(); err != nil {
		conf.Log().Warn("spa shell not available, run 'npm run build:app'", "error", err)
	}
	return h
}

func (h *AppHandler) RegisterRoutes(router chi.Router) {
	authenticated := chi.NewRouter()
	authenticated.Use(middlewares.NewAuthenticateMiddleware(h.userSrvc).
		WithRedirectTarget(defaultErrorRedirectTarget()).
		WithRedirectErrorMessage("unauthorized").Handler,
	)
	authenticated.Get("/*", h.ServeShell)

	// While staged, everything lives below appMountPrefix so it cannot collide with the
	// legacy MVC routes. chi panics on a duplicate mount, which is the failure we want.
	staged := chi.NewRouter()
	for _, path := range appRoutes {
		staged.Mount(path, authenticated)
	}

	if h.config.App.LeaderboardEnabled {
		public := chi.NewRouter()
		auth := middlewares.NewAuthenticateMiddleware(h.userSrvc).
			WithRedirectTarget(defaultErrorRedirectTarget()).
			WithRedirectErrorMessage("unauthorized")
		if !h.config.App.LeaderboardRequireAuth {
			// isOptional matches r.URL.Path, which is mount-independent but also
			// base_path-independent (the reverse proxy strips base_path before us).
			auth = auth.WithOptionalFor(appMountPrefix + "/leaderboard")
		}
		public.Use(auth.Handler)
		public.Get("/*", h.ServeShell)
		staged.Mount("/leaderboard", public)
	}

	staged.Get("/", http.RedirectHandler(h.appBase()+"/summary", http.StatusFound).ServeHTTP)

	router.Mount(appMountPrefix, staged)
}

// appBase is the url prefix the client-side router lives under.
func (h *AppHandler) appBase() string {
	return h.config.Server.BasePath + appMountPrefix
}

// ServeShell writes the SPA entry document with the deployment-specific head block
// injected. It is never cached, because that block depends on server config.
func (h *AppHandler) ServeShell(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		if err := h.loadShell(); err != nil {
			conf.Log().Request(r).Warn("failed to reload spa shell", "error", err)
		}
	}

	h.mu.RLock()
	shell := h.shell
	h.mu.RUnlock()

	if shell == nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("wakapi frontend is not built - run 'npm run build:app'\n"))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	w.Write(shell.head)
	w.Write(h.headBlock())
	w.Write(shell.tail)
}

// headBlock is the markup injected in place of appShellMarker: a <base> tag that
// makes Vite's relative asset urls resolve under any base_path, plus the non
// user-specific server config the app needs synchronously at boot.
func (h *AppHandler) headBlock() []byte {
	var buf bytes.Buffer
	buf.WriteString(`<base href="`)
	buf.WriteString(h.config.Server.BasePath + appAssetBase)
	buf.WriteString(`">`)

	payload, err := json.Marshal(h.serverConfig())
	if err != nil {
		conf.Log().Error("failed to marshal spa server config", "error", err)
		payload = []byte("{}")
	}
	buf.WriteString(`<script>window.__WAKAPI__=`)
	// encoding/json already escapes <, > and &, but the script context is
	// unforgiving enough to warrant belt and braces.
	buf.WriteString(scriptEscaper.Replace(string(payload)))
	buf.WriteString(`;</script>`)
	return buf.Bytes()
}

// scriptEscaper keeps a "</script>" inside any string value from ending the tag.
// encoding/json already escapes these, but the script context is unforgiving enough
// to warrant belt and braces.
var scriptEscaper = strings.NewReplacer("<", `\u003c`, ">", `\u003e`, "&", `\u0026`)

type appServerConfig struct {
	BasePath            string            `json:"basePath"`
	AppBase             string            `json:"appBase"`
	Version             string            `json:"version"`
	PublicUrl           string            `json:"publicUrl"`
	DbType              string            `json:"dbType"`
	SupportContact      string            `json:"supportContact"`
	AvatarUrlTemplate   string            `json:"avatarUrlTemplate"`
	DataRetentionMonths int               `json:"dataRetentionMonths"`
	Branding            appServerBranding `json:"branding"`
	Features            appServerFeatures `json:"features"`
}

// appServerBranding is the subset of the instance branding the app itself renders.
// It ships in the shell rather than in a fetch so the header paints with the right
// name and logo on the first frame instead of flashing a placeholder.
type appServerBranding struct {
	InstanceName string `json:"instanceName"`
	LogoUrl      string `json:"logoUrl"`
	FooterText   string `json:"footerText"`
	FooterUrl    string `json:"footerUrl"`
}

type appServerFeatures struct {
	LeaderboardEnabled     bool `json:"leaderboardEnabled"`
	LeaderboardRequireAuth bool `json:"leaderboardRequireAuth"`
	InvitesEnabled         bool `json:"invitesEnabled"`
	SubscriptionsEnabled   bool `json:"subscriptionsEnabled"`
	ImportEnabled          bool `json:"importEnabled"`
	WebauthnEnabled        bool `json:"webauthnEnabled"`
	AllowSignup            bool `json:"allowSignup"`
}

func (h *AppHandler) serverConfig() *appServerConfig {
	c := h.config
	branding := h.brandingSrvc.Get()
	return &appServerConfig{
		BasePath:            c.Server.BasePath,
		AppBase:             h.appBase(),
		Version:             c.Version,
		PublicUrl:           c.Server.GetPublicUrl(),
		DbType:              strings.ToLower(c.Db.Type),
		SupportContact:      c.App.SupportContact,
		AvatarUrlTemplate:   c.App.AvatarURLTemplate,
		DataRetentionMonths: c.App.DataRetentionMonths,
		Branding: appServerBranding{
			InstanceName: branding.InstanceName,
			LogoUrl:      branding.LogoUrl,
			FooterText:   branding.FooterText,
			FooterUrl:    branding.FooterUrl,
		},
		Features: appServerFeatures{
			LeaderboardEnabled:     c.App.LeaderboardEnabled,
			LeaderboardRequireAuth: c.App.LeaderboardRequireAuth,
			InvitesEnabled:         c.Security.InviteCodes,
			SubscriptionsEnabled:   c.Subscriptions.Enabled,
			ImportEnabled:          c.App.ImportEnabled,
			WebauthnEnabled:        !c.Security.DisableWebAuthn,
			AllowSignup:            c.Security.AllowSignup,
		},
	}
}

func (h *AppHandler) loadShell() error {
	raw, err := fs.ReadFile(h.static, appShellPath)
	if err != nil {
		return err
	}

	head, tail, found := bytes.Cut(raw, []byte(appShellMarker))
	if !found {
		return errShellMarkerMissing
	}

	h.mu.Lock()
	h.shell = &appShell{head: head, tail: tail}
	h.mu.Unlock()
	return nil
}
