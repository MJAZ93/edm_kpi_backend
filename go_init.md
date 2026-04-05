# Go Backend Init Guide (Gin + GORM + JWT + Swagger + Postgres)

This guide describes a clean, production‑ready Go web backend architecture similar to this project, focusing on the layers (service/controller/dao/middlewares), the tooling (Gin, GORM, godotenv, Swagger), JWT authentication, `.env` structure, and PostgreSQL usage. It includes minimal examples you can copy to bootstrap a fresh project.

## 1) Tech Stack

- Gin: Fast HTTP router/framework for APIs.
- GORM: ORM for PostgreSQL with migrations via `AutoMigrate`.
- godotenv: Loads environment variables from `.env` at startup.
- Swagger (swaggo): API docs generator (`gin-swagger` + annotations).
- JWT (golang-jwt): Token-based auth for private routes.
- PostgreSQL: Primary database, configured by `.env`.

## 2) Project Structure

```
backend/
  .env
  go.mod
  go.sum
  main.go
  app/
    app.go        # env, db init, server start
    route.go      # route grouping (public/private) + swagger
  controller/     # HTTP handlers (bind/validate → call service/dao → respond)
  service/        # Route registration + orchestration glue
  dao/            # Data access (GORM queries)
  model/          # GORM models and hooks
  middleware/     # CORS, JWT, logging, etc.
  docs/           # generated swagger files (swag init)
```

Keep handlers thin in `controller`, put DB logic into `dao`, and only simple orchestration in `service`.

## 3) .env Structure

Create a `.env` at the repo root. Example:

```
# Server
IP=localhost
PORT=8000
SCHEME=http

# Primary Postgres
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=app_db

# Auth
JWT_PRIVATE_KEY=CHANGE_ME_SUPER_SECRET
TOKEN_TTL=3600  # seconds
```

Load this with `godotenv` during startup. Never commit real credentials.

## 4) main.go Bootstrap

```go
package main

import (
  "log"
  "your_project/app"
)

// @title           Your API
// @version         1.0
// @BasePath        /api/v1

func main() {
  app.LoadEnv()

  if err := app.LoadDatabase(); err != nil {
    log.Fatalf("database init failed: %v", err)
  }

  app.ServeApplication()
}
```

- Keep `main.go` minimal: load env, init DB, start HTTP.

## 5) App Init: env, DB, server

`app/app.go` — load `.env`, connect DB(s), run migrations, start Gin.

```go
package app

import (
  "fmt"
  "log"
  "os"
  "your_project/dao"
  "your_project/middleware"
  "your_project/model"

  "github.com/gin-gonic/gin"
  "github.com/joho/godotenv"
)

func LoadEnv() {
  if err := godotenv.Load(".env"); err != nil {
    log.Fatal("failed loading .env: ", err)
  }
}

func LoadDatabase() error {
  dao.Connect() // reads env and opens primary DB

  // AutoMigrate models
  if err := dao.Database.AutoMigrate(&model.User{}); err != nil { return err }
  // add more models here...

  return nil
}

func ServeApplication() {
  r := gin.Default()

  Swagger(r) // expose /swagger

  // Global/CORS headers, OPTIONS preflight, JSON Content-Type
  r.Use(middleware.DefaultAuthMiddleware())

  base := r.Group("/api/v1")

  // Public routes (no JWT)
  PublicRoutes(base.Group("/public"))

  // Private routes (JWT required)
  priv := base.Group("/private")
  priv.Use(middleware.JWTAuthMiddleware())
  PrivateRoutes(priv)

  ip, port := os.Getenv("IP"), os.Getenv("PORT")
  if err := r.Run(fmt.Sprintf("%s:%s", ip, port)); err != nil {
    log.Fatalf("gin failed to start: %v", err)
  }
}
```

`app/route.go` — define and register routes via `service` layer and enable Swagger:

```go
package app

import (
  "your_project/controller"
  "your_project/service"
  "github.com/gin-gonic/gin"
  swaggerfiles "github.com/swaggo/files"
  ginSwagger "github.com/swaggo/gin-swagger"
)

func PublicRoutes(r *gin.RouterGroup) {
  userSvc := service.UserService{Route: "user", Controller: controller.UserController{}}
  userSvc.Login(r, "login")
}

func PrivateRoutes(r *gin.RouterGroup) {
  userSvc := service.UserService{Route: "user", Controller: controller.UserController{}}
  userSvc.List(r, "list")
}

func Swagger(r *gin.Engine) {
  r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}
```

## 6) DAO: PostgreSQL + GORM

`dao/main.go` — single place to connect to Postgres using env vars.

```go
package dao

import (
  "fmt"
  "os"
  "gorm.io/driver/postgres"
  "gorm.io/gorm"
  "gorm.io/gorm/logger"
)

var Database *gorm.DB

func Connect() {
  dsn := fmt.Sprintf(
    "host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
    os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
    os.Getenv("DB_NAME"), os.Getenv("DB_PORT"),
  )

  db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{ Logger: logger.Default.LogMode(logger.Warn) })
  if err != nil { panic(err) }

  Database = db
}
```

- Use `AutoMigrate` in `app.LoadDatabase` to create/update tables.
- Create indexes via `Database.Exec("CREATE INDEX IF NOT EXISTS ...")` when needed.

### Example DAO

```go
package dao

import "your_project/model"

type UserDao struct{ Limit int }

func (d *UserDao) Create(u *model.User) error { return Database.Create(u).Error }

func (d *UserDao) GetByID(id uint) (model.User, error) {
  var u model.User
  err := Database.Where("id = ?", id).First(&u).Error
  return u, err
}

func (d *UserDao) List(page int) ([]model.User, error) {
  var list []model.User
  q := Database
  if d.Limit > 0 && page >= 0 {
    q = q.Offset(page * d.Limit).Limit(d.Limit)
  }
  return list, q.Find(&list).Error
}
```

## 7) Models: GORM + Hooks

`model/user.go` — example with constraints and a password hash hook.

```go
package model

import (
  "golang.org/x/crypto/bcrypt"
  "gorm.io/gorm"
)

type User struct {
  gorm.Model
  Name     string `gorm:"not null;size:128"`
  Username string `gorm:"not null;size:128;uniqueIndex"`
  Password string `gorm:"size:100"`
  Email    string `gorm:"size:100"`
  Type     string `gorm:"not null;size:14"` // e.g., ADMIN, MEMBER
}

func (u *User) BeforeSave(tx *gorm.DB) error {
  if u.Password == "" { return nil }
  b, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
  if err != nil { return err }
  u.Password = string(b)
  return nil
}
```

## 8) Middleware

`middleware/default_mdw.go` — CORS/headers and OPTIONS handling.

```go
package middleware

import (
  "net/http"
  "github.com/gin-gonic/gin"
)

func DefaultAuthMiddleware() gin.HandlerFunc {
  return func(c *gin.Context) {
    c.Header("Content-Type", "application/json")
    c.Header("Access-Control-Allow-Origin", "*")
    c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
    c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
    c.Header("Access-Control-Allow-Credentials", "true")

    if c.Request.Method == http.MethodOptions {
      c.AbortWithStatus(http.StatusOK)
      return
    }
    c.Next()
  }
}
```

`middleware/jwt_mdw.go` — enforce JWT on private routes.

```go
package middleware

import (
  "net/http"
  "github.com/gin-gonic/gin"
  "your_project/util"
)

func JWTAuthMiddleware() gin.HandlerFunc {
  return func(c *gin.Context) {
    if err := util.ValidateJWT(c); err != nil {
      c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": err.Error()})
      c.Abort()
      return
    }
    c.Next()
  }
}
```

## 9) JWT Utilities

`util/jwt.go` — generate and validate tokens using `JWT_PRIVATE_KEY` and `TOKEN_TTL`.

```go
package util

import (
  "errors"
  "fmt"
  "os"
  "strconv"
  "strings"
  "time"
  "github.com/gin-gonic/gin"
  "github.com/golang-jwt/jwt/v4"
)

var privateKey = []byte(os.Getenv("JWT_PRIVATE_KEY"))

func GenerateJWT(id uint, payload any) (string, error) {
  ttl, _ := strconv.Atoi(os.Getenv("TOKEN_TTL"))
  claims := jwt.MapClaims{
    "id":  id,
    "iat": time.Now().Unix(),
    "exp": time.Now().Add(time.Second * time.Duration(ttl)).Unix(),
    "data": payload,
  }
  token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
  return token.SignedString(privateKey)
}

func ValidateJWT(c *gin.Context) error {
  t, err := getToken(c)
  if err != nil { return err }
  if !t.Valid { return errors.New("invalid token") }
  return nil
}

func getToken(c *gin.Context) (*jwt.Token, error) {
  raw := getTokenFromRequest(c)
  return jwt.Parse(raw, func(token *jwt.Token) (interface{}, error) {
    if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
      return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"]) }
    return privateKey, nil
  })
}

func getTokenFromRequest(c *gin.Context) string {
  auth := c.Request.Header.Get("Authorization")
  parts := strings.Split(auth, " ")
  if len(parts) == 2 { return parts[1] }
  return ""
}
```

## 10) Controller: Thin HTTP Handlers

`controller/user_controller.go` — bind requests, call DAO/service, respond.

```go
package controller

import (
  "net/http"
  "strconv"
  "github.com/gin-gonic/gin"
  "your_project/dao"
  "your_project/model"
  "your_project/util"
)

type UserController struct{}

type LoginIn struct { Username string `json:"username"` Password string `json:"password"` }

type LoginOut struct { Token string `json:"token"` }

func (UserController) Login(c *gin.Context) {
  var in LoginIn
  if err := c.ShouldBindJSON(&in); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error":"bad_request","message":err.Error()})
    return
  }
  // validate user via DAO (pseudo)
  u, err := dao.UserDao{Limit:1}.GetByUsernameAndPassword(in.Username, in.Password)
  if err != nil { c.JSON(http.StatusUnauthorized, gin.H{"error":"invalid_credentials"}); return }

  tok, err := util.GenerateJWT(uint(u.ID), gin.H{"username": u.Username})
  if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"token_error"}); return }

  c.JSON(http.StatusOK, LoginOut{Token: tok})
}

func (UserController) Single(c *gin.Context) {
  id, _ := strconv.Atoi(c.Param("id"))
  u, err := dao.UserDao{Limit:1}.GetByID(uint(id))
  if err != nil { c.JSON(http.StatusNotFound, gin.H{"error":"not_found"}); return }
  c.JSON(http.StatusOK, u)
}
```

## 11) Service Layer: Route Glue + Swagger Annotations

`service/user_service.go` — register endpoints and add Swagger docs.

```go
package service

import (
  "github.com/gin-gonic/gin"
  "your_project/controller"
)

type UserService struct {
  Route string
  Controller controller.UserController
}

// Login godoc
// @Summary User login
// @Tags    User
// @Accept  json
// @Produce json
// @Success 200 {object} controller.LoginOut
// @Param   body body controller.LoginIn true "login"
// @Router  /public/user/login [post]
func (s UserService) Login(r *gin.RouterGroup, route string) {
  r.POST("/"+s.Route+"/"+route, s.Controller.Login)
}

// Single godoc
// @Summary Get user by ID
// @Tags    User
// @Produce json
// @Param   id path int true "ID"
// @Success 200 {object} model.User
// @Security BearerAuth
// @Router  /private/user/single/{id} [get]
func (s UserService) Single(r *gin.RouterGroup, route string) {
  r.GET("/"+s.Route+"/"+route+"/:id", s.Controller.Single)
}
```

Add a Swagger security definition in your generated docs (e.g., Bearer token) and expose `/swagger/index.html` at runtime.

## 12) Swagger Setup

- Install tools:
  - `go install github.com/swaggo/swag/cmd/swag@latest`
  - `go get github.com/swaggo/gin-swagger github.com/swaggo/files`
- Add top-level annotations in `main.go` (title, version, base path).
- Add per-route annotations in service files.
- Generate docs: `swag init` (creates/updates `docs/`).
- Run the server and open `http://localhost:8000/swagger/index.html`.

## 13) Integration Tests for APIs

Goal: exercise real routing, middleware (JWT, CORS), and GORM models end‑to‑end using `httptest`. For speed and simplicity, use an in‑memory SQLite database during tests while production uses Postgres. Alternatively, point tests to a dedicated Postgres database via a `.env.test` file.

### Add missing DAO helper for login

If you follow the controller example that calls `GetByUsernameAndPassword`, add this method to your `dao/user_dao.go`:

```go
// dao/user_dao.go
package dao

import (
  "your_project/model"
  "gorm.io/gorm"
  "golang.org/x/crypto/bcrypt"
)

func (d *UserDao) GetByUsernameAndPassword(username, password string) (model.User, error) {
  var u model.User
  if err := Database.Where("username = ?", username).First(&u).Error; err != nil {
    return model.User{}, err
  }
  if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
    return model.User{}, gorm.ErrRecordNotFound
  }
  return u, nil
}
```

### Example integration test

Create `controller/user_integration_test.go` (or `internal/integration/user_integration_test.go`). It bootstraps Gin, JWT middleware, and a test DB, then verifies login and a private route.

```go
package integration

import (
  "bytes"
  "encoding/json"
  "net/http"
  "net/http/httptest"
  "os"
  "testing"

  "github.com/gin-gonic/gin"
  "gorm.io/driver/sqlite"
  "gorm.io/gorm"

  "your_project/dao"
  "your_project/model"
  "your_project/middleware"
  "your_project/service"
  "your_project/controller"
)

func setupTestDB(t *testing.T) {
  db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
  if err != nil { t.Fatalf("db open: %v", err) }
  dao.Database = db
  if err := dao.Database.AutoMigrate(&model.User{}); err != nil { t.Fatalf("migrate: %v", err) }
  // seed a user (password gets hashed by model hook)
  u := model.User{Name: "Alice", Username: "alice", Password: "secret", Email: "a@x.io", Type: "ADMIN"}
  if err := dao.Database.Create(&u).Error; err != nil { t.Fatalf("seed: %v", err) }
}

func setupRouter() *gin.Engine {
  gin.SetMode(gin.TestMode)
  r := gin.Default()
  r.Use(middleware.DefaultAuthMiddleware())

  base := r.Group("/api/v1")
  pub := base.Group("/public")
  priv := base.Group("/private")
  priv.Use(middleware.JWTAuthMiddleware())

  userSvc := service.UserService{Route: "user", Controller: controller.UserController{}}
  userSvc.Login(pub, "login")
  userSvc.Single(priv, "single")
  return r
}

func TestLoginAndGetUser(t *testing.T) {
  os.Setenv("JWT_PRIVATE_KEY", "test_secret")
  os.Setenv("TOKEN_TTL", "3600")

  setupTestDB(t)
  r := setupRouter()

  // 1) Login
  reqBody, _ := json.Marshal(controller.LoginIn{Username: "alice", Password: "secret"})
  w := httptest.NewRecorder()
  req, _ := http.NewRequest(http.MethodPost, "/api/v1/public/user/login", bytes.NewReader(reqBody))
  req.Header.Set("Content-Type", "application/json")
  r.ServeHTTP(w, req)
  if w.Code != http.StatusOK { t.Fatalf("login status=%d body=%s", w.Code, w.Body.String()) }

  var out controller.LoginOut
  if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || out.Token == "" {
    t.Fatalf("invalid token response: %v body=%s", err, w.Body.String())
  }

  // 2) Private endpoint
  w2 := httptest.NewRecorder()
  req2, _ := http.NewRequest(http.MethodGet, "/api/v1/private/user/single/1", nil)
  req2.Header.Set("Authorization", "Bearer "+out.Token)
  r.ServeHTTP(w2, req2)
  if w2.Code != http.StatusOK { t.Fatalf("single status=%d body=%s", w2.Code, w2.Body.String()) }
}
```

Notes:
- Use SQLite in tests for speed and isolation; production still uses Postgres.
- Alternatively, create a `.env.test` and a `LoadEnvForTests()` that points to a test Postgres DB, then run migrations in `TestMain`.

## 14) End‑to‑End Flow (Example)

1) Create user (seed or API) → row in `users`.
2) `POST /api/v1/public/user/login` with JSON `{ "username":"alice", "password":"secret" }` → receive `{ "token": "…" }`.
3) Call a private endpoint with header `Authorization: Bearer <token>` → allowed by `JWTAuthMiddleware`.

Example curl:

```
# Login
curl -s -X POST http://localhost:8000/api/v1/public/user/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"secret"}'

# Use token for private route
curl -s http://localhost:8000/api/v1/private/user/single/1 \
  -H 'Authorization: Bearer YOUR_JWT_HERE'
```

## 15) Conventions and Tips

- Keep controllers thin; complex queries live in DAO.
- Centralize DB connection(s) in `dao` and pass only DAO methods to controllers/services.
- Prefer small `service` structs to group related endpoints and Swagger docs coherently.
- Use `AutoMigrate` during development; replace with migrations tooling for production.
- Use `logger.Warn` or `logger.Error` for GORM to avoid noisy logs.
- Never log or commit real secrets; use sample `.env.example` in public repos.

## 16) Dependencies (minimal go.mod)

```
module your_project

go 1.22

require (
  github.com/gin-gonic/gin v1.10.0
  github.com/golang-jwt/jwt/v4 v4.5.0
  github.com/joho/godotenv v1.5.1
  github.com/swaggo/files v1.0.1
  github.com/swaggo/gin-swagger v1.6.0
  gorm.io/driver/postgres v1.5.9
  gorm.io/gorm v1.25.7
)
```

---

With this structure and examples, you can bootstrap a clean Gin + GORM + JWT + Swagger service backed by PostgreSQL, with public/private routes split by middleware and a clear separation of concerns between controller, service, and DAO layers.
