package app

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"erp/internal/apigen"
	"erp/internal/auth"
	"erp/internal/biz"
	"erp/internal/config"
	"erp/internal/middleware"
	"erp/internal/persistence"
	"erp/internal/server/health"
)

type App struct {
	Cfg *config.Config
	DB  *persistence.DB
	Eng *gin.Engine
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
	eng := gin.New()
	eng.Use(gin.Recovery(), gin.Logger(), middleware.CORS(cfg.CORS.AllowOrigins))

	permit := []string{
		"/api/v1/health",
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
	}
	eng.Use(middleware.JWT(cfg.JWT.Secret, permit))
	eng.Use(middleware.Audit(db.SQL))

	// ensure automation tables/columns/seed on existing DBs
	biz.EnsureAutomationSchema(db.SQL)
	biz.EnsureHRPermSchema(db.SQL)
	biz.EnsureHROpsSchema(db.SQL)

	v1 := eng.Group("/api/v1")
	health.Register(v1, db.Driver)
	auth.Register(v1, &auth.Handler{DB: db.SQL, Cfg: cfg})

	engine := apigen.NewEngine(db.SQL, db.Driver)
	apigen.RegisterGenerated(v1, engine)

	// coverage helper: dump gin routes to scripts/gin_routes.json
	v1.GET("/_debug/routes", func(c *gin.Context) {
		routes := apigen.DumpRoutes(eng)
		_ = writeRoutesJSON(routes)
		apiOK := map[string]interface{}{"code": 1, "msg": "ok", "data": gin.H{"routes": routes, "count": len(routes)}}
		c.JSON(200, apiOK)
	})

	return &App{Cfg: cfg, DB: db, Eng: eng}, nil
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
	// dump routes at startup for coverage offline
	_ = writeRoutesJSON(apigen.DumpRoutes(a.Eng))
	log.Printf("erp-api listening on %s (driver=%s)", a.Cfg.Server.Addr, a.DB.Driver)
	return a.Eng.Run(a.Cfg.Server.Addr)
}

func (a *App) Close() {
	_ = a.DB.Close()
}
