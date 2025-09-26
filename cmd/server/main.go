package main

import (
    "context"
    "database/sql"
    "embed"
    "encoding/csv"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "strconv"
    "strings"
    "syscall"
    "time"

    _ "modernc.org/sqlite"
    
    "100y-saas/internal/config"
    "100y-saas/internal/health"
    httphandlers "100y-saas/internal/http"
    "100y-saas/internal/jobs"
    "100y-saas/internal/logger"
)

//go:embed ../../web/*
var webFS embed.FS

//go:embed ../../internal/db/schema.sql
var schemaSQL string

type App struct {
    db     *sql.DB
    cfg    *config.Config
    log    *logger.Logger
}

func main() {
    cfg, err := config.Load()
    if err != nil {
        panic(err)
    }

    dsn := cfg.Database.Path
    if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
        panic(err)
    }

    db, err := sql.Open("sqlite", dsn+"?_busy_timeout=5000&_fk=1")
    if err != nil { panic(err) }
    if err := migrate(db, schemaSQL); err != nil { panic(err) }

    // Optional: tune connection pool (SQLite driver may ignore some of these)
    db.SetMaxOpenConns(cfg.Database.MaxOpenConnections)
    db.SetMaxIdleConns(cfg.Database.MaxIdleConnections)
    db.SetConnMaxLifetime(cfg.Database.ConnectionLifetime)

    app := &App{db: db, cfg: cfg, log: logger.New("server")}

    mux := http.NewServeMux()

    // static files
    fs := http.FS(webFS)
    mux.Handle("/", withSecurityHeaders(http.FileServer(fs)))

    // Health endpoints
    hc := health.NewHealthChecker(db)
    mux.Handle("/healthz", hc)
    mux.HandleFunc("/live", health.LivenessHandler)
    mux.HandleFunc("/ready", hc.ReadinessHandler)

    // api routes
    handlers := httphandlers.NewHandlers(db, cfg)
    withCORS := handlers.CORS
    withReqID := handlers.RequestID

    mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request){
        writeJSON(w, map[string]string{"pong":"ok", "time": time.Now().UTC().Format(time.RFC3339)})
    })

    // Auth
    mux.Handle("/api/auth/register", withCORS(withReqID(http.HandlerFunc(handlers.Register))))
    mux.Handle("/api/auth/login", withCORS(withReqID(http.HandlerFunc(handlers.Login))))
    mux.Handle("/api/auth/logout", withCORS(withReqID(http.HandlerFunc(handlers.Logout))))

    // Tenants
    mux.Handle("/api/tenants", withCORS(withReqID(http.HandlerFunc(handlers.RequireAuth(handlers.GetTenants)))))
    mux.Handle("/api/tenants/create", withCORS(withReqID(http.HandlerFunc(handlers.RequireAuth(handlers.CreateTenant)))))

    // Analytics
    mux.Handle("/api/analytics/stats", withCORS(withReqID(http.HandlerFunc(handlers.RequireTenant(handlers.GetAnalytics)))))

    // Export all data
    mux.Handle("/api/export-all", withCORS(withReqID(http.HandlerFunc(handlers.RequireTenant(handlers.ExportAll)))))

    // Legacy endpoints (for backward compatibility)
    mux.HandleFunc("/api/items", app.itemsHandler)
    mux.HandleFunc("/export", app.exportCSV)

    // Background jobs processor
    processor := jobs.NewJobProcessor(db)
    processor.Start()

    srv := &http.Server{
        Addr:         ":"+strconv.Itoa(cfg.Server.Port),
        Handler:      logRequests(mux),
        ReadTimeout:  cfg.Server.ReadTimeout,
        WriteTimeout: cfg.Server.WriteTimeout,
        IdleTimeout:  cfg.Server.IdleTimeout,
    }

    // Graceful shutdown
    go func() {
        app.log.Info("listening", map[string]interface{}{"addr": srv.Addr})
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            app.log.Error("server error", map[string]interface{}{"error": err.Error()})
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
    defer cancel()

    processor.Stop()

    if err := srv.Shutdown(ctx); err != nil {
        app.log.Error("shutdown error", map[string]interface{}{"error": err.Error()})
    }
}

func migrate(db *sql.DB, sqlText string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _, err := db.ExecContext(ctx, sqlText)
    return err
}

// minimal example entity
type Item struct {
    ID    int64  `json:"id"`
    Title string `json:"title"`
    Note  string `json:"note"`
}

func (a *App) itemsHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        rows, err := a.db.Query("SELECT id, title, note FROM items ORDER BY id")
        if err != nil { http.Error(w, err.Error(), 500); return }
        defer rows.Close()
        var out []Item
        for rows.Next(){ var it Item; if err:=rows.Scan(&it.ID,&it.Title,&it.Note); err!=nil { http.Error(w, err.Error(), 500); return }; out=append(out,it) }
        writeJSON(w, out)
    case http.MethodPost:
        var in Item
        if err := json.NewDecoder(r.Body).Decode(&in); err != nil { http.Error(w, "bad json", 400); return }
        if strings.TrimSpace(in.Title) == "" { http.Error(w, "title required", 400); return }
        res, err := a.db.Exec("INSERT INTO items(title,note) VALUES(?,?)", in.Title, in.Note)
        if err != nil { http.Error(w, err.Error(), 500); return }
        id, _ := res.LastInsertId(); in.ID = id
        writeJSON(w, in)
    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

func (a *App) exportCSV(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", "attachment; filename=items.csv")
    cw := csv.NewWriter(w)
    defer cw.Flush()
    cw.Write([]string{"id","title","note"})
    rows, err := a.db.Query("SELECT id, title, note FROM items ORDER BY id")
    if err != nil { http.Error(w, err.Error(), 500); return }
    defer rows.Close()
    for rows.Next(){ var id int64; var title, note string; rows.Scan(&id,&title,&note); cw.Write([]string{strconv.FormatInt(id,10), title, note}) }
}

func writeJSON(w http.ResponseWriter, v any) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    json.NewEncoder(w).Encode(v)
}

func withSecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("Referrer-Policy", "no-referrer")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        next.ServeHTTP(w, r)
    })
}

func logRequests(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
    })
}

func env(k, def string) string {
    if v := os.Getenv(k); v != "" { return v }
    return def
}
