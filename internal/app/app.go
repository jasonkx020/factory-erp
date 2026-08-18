package app

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gin-gonic/gin"

	"erp/internal/apigen"
	"erp/internal/auth"
	"erp/internal/biz"
	"erp/internal/config"
	"erp/internal/middleware"
	erpmqtt "erp/internal/mqtt"
	"erp/internal/notify"
	"erp/internal/persistence"
	"erp/internal/server/health"
	"erp/internal/webui"
)

type App struct {
	Cfg        *config.Config
	DB         *persistence.DB
	Eng        *gin.Engine
	mqttHub    *erpmqtt.Hub
	notifyStop chan struct{}
}

func New(cfgPath string) (*App, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	db, err := persistence.Open(cfg)
	if err != nil {
		return nil, err
	}
	gin.SetMode(gin.ReleaseMode)
	// Windows 控制台常无法解析 ANSI 颜色码，表现为 [97;42m 一类乱码
	if runtime.GOOS == "windows" {
		gin.DisableConsoleColor()
	}
	eng := gin.New()
	eng.Use(gin.Recovery(), gin.Logger(), middleware.CORS(cfg.CORS.AllowOrigins))

	permit := []string{
		"/api/v1/health",
		"/api/v1/ready",
		"/api/v1/live",
		"/api/v1/metrics",
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
		"/api/v1/auth/oauth/token",
		"/api/v1/mqtt/auth",
		"/api/v1/mqtt/superuser",
		"/api/v1/mqtt/acl",
		"/files",
	}
	eng.Use(middleware.Metrics())
	eng.Use(middleware.JWT(cfg.JWT.Secret, db.SQL, permit, webui.IsStaticPath))
	eng.Use(middleware.Audit(db.SQL))
	_ = os.MkdirAll(filepath.Join("data", "uploads"), 0o755)
	eng.Static("/files", "data")

	// Schema is owned by migrations/erp (erp-db / init_schema). Do not Ensure DDL at startup.
	biz.EnsureDomainPermissions(db.SQL)
	biz.EnsureDemoRoleUsers(db.SQL)
	if cfg.Seed.DemoEnabled() {
		biz.EnsureCleanDevWageRates(db.SQL)
		biz.EnsureDemoData(db.SQL)
	}
	biz.EnsureFounderSuperuser(db.SQL)

	hub := erpmqtt.NewHub(cfg)
	notifySvc := notify.New(db.SQL, cfg, hub)
	mqttAuth := erpmqtt.NewAuthHandler(cfg)
	mqttAuth.Register(eng)

	v1 := eng.Group("/api/v1")
	health.Register(v1, db.SQL, db.Driver)
	middleware.RegisterMetrics(v1)
	auth.Register(v1, &auth.Handler{DB: db.SQL, Cfg: cfg})

	engine := apigen.NewEngine(db.SQL, db.Driver)
	if cfg.Trace.HMACSecret != "" {
		engine.Biz.TraceHMACSecret = cfg.Trace.HMACSecret
	}
	engine.Biz.OCREnabled = cfg.OCR.Enabled
	engine.Biz.OCRProvider = cfg.OCR.Provider
	engine.Biz.Notify = notifySvc
	apigen.RegisterGenerated(v1, engine)
	apigen.RegisterHRExtra(v1, engine)
	apigen.RegisterPayrollExtra(v1, engine)
	apigen.RegisterApprovalExtra(v1, engine)
	apigen.RegisterPurchaseExtra(v1, engine)
	apigen.RegisterProductionExtra(v1, engine)
	apigen.RegisterSalesExtra(v1, engine)
	apigen.RegisterInventoryExtra(v1, engine)
	apigen.RegisterFinanceExtra(v1, engine)
	apigen.RegisterWorkflowExtra(v1, engine)

	stop := make(chan struct{})
	go notifySvc.StartPublisher(stop)

	uiFS, uiErr := webui.Open(cfg.Server.WebRoot)
	if uiErr != nil {
		log.Printf("webui: open failed: %v (continuing without UI)", uiErr)
	} else {
		webui.Register(eng, uiFS)
		if uiFS.Source == webui.SourceExternal {
			log.Printf("webui: serving external root %s", uiFS.Root)
		} else {
			log.Printf("webui: serving embedded dist")
		}
	}

	// 生产关闭 debug；demo/开发仅 sys_admin 可访问
	if cfg.Seed.DemoEnabled() {
		v1.GET("/_debug/routes", func(c *gin.Context) {
			claims := middleware.Claims(c)
			if claims == nil || !biz.ClaimsIsSysAdmin(claims.Roles, claims.Permissions) {
				c.AbortWithStatusJSON(403, gin.H{"code": 0, "msg": "PERM_DENIED"})
				return
			}
			routes := apigen.DumpRoutes(eng)
			_ = writeRoutesJSON(routes)
			c.JSON(200, gin.H{"code": 1, "msg": "ok", "data": gin.H{"routes": routes, "count": len(routes)}})
		})
	}

	return &App{Cfg: cfg, DB: db, Eng: eng, mqttHub: hub, notifyStop: stop}, nil
}

func writeRoutesJSON(routes []map[string]string) error {
	b, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join("scripts", "gin_routes.json")
	_ = os.MkdirAll("scripts", 0o755)
	return os.WriteFile(path, b, 0o644)
}

func (a *App) Run() error {
	_ = writeRoutesJSON(apigen.DumpRoutes(a.Eng))
	log.Printf("erp-api listening on %s (driver=%s mqtt=%v)", a.Cfg.Server.Addr, a.DB.Driver, a.Cfg.Mqtt.Enabled)
	return a.Eng.Run(a.Cfg.Server.Addr)
}

func (a *App) Close() {
	if a.notifyStop != nil {
		close(a.notifyStop)
	}
	if a.mqttHub != nil {
		a.mqttHub.Close()
	}
	_ = a.DB.Close()
}
