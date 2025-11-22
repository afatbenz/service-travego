# Service TraveGO

Service backend untuk aplikasi TraveGO menggunakan Go Fiber dengan clean architecture.

## Struktur Project

```
service-travego/
├── main.go                 # Entry point aplikasi
├── go.mod                  # Go module dependencies
├── go.sum                  # Go module checksums
├── .gitignore             # Git ignore rules
│
├── routes/                 # Routing layer
│   └── routes.go          # Setup semua routes
│
├── handler/                # HTTP handlers (handle request, call service)
│   └── user_handler.go
│
├── service/                # Business logic layer
│   └── user_service.go
│
├── repository/             # Database access layer
│   └── user_repository.go
│
├── model/                  # Data models/entities
│   └── user.go
│
├── helper/                 # Utility functions
│   ├── validator.go       # Validation helper
│   ├── response.go        # Response helper
│   ├── error.go           # Error helper
│   ├── hash.go            # Password hashing
│   └── string.go          # String utilities
│
├── configs/                # Configuration loader (code)
│   └── config.go
│
├── db/                     # Database files
│   ├── migrations/        # SQL migration files
│   │   └── 001_create_users_table.sql
│   └── seed.sql           # Seed data
│
├── assets/                 # Media files (images, videos, documents, etc.)
│
└── config/                 # Configuration files (JSON)
    ├── app.json           # Application configuration
    ├── general-config.json # General configuration (company info)
    └── web-menu.json      # Web menu structure
```

## Arsitektur

Proyek ini menggunakan **Clean Architecture** dengan layer-layered structure:

1. **Handler Layer** (`handler/`)
   - Menangani HTTP request/response
   - Validasi input
   - Memanggil service layer

2. **Service Layer** (`service/`)
   - Business logic
   - Validasi bisnis
   - Koordinasi antar repository

3. **Repository Layer** (`repository/`)
   - Akses database
   - CRUD operations
   - Query database

4. **Model Layer** (`model/`)
   - Data models/entities
   - Struktur data

5. **Helper Layer** (`helper/`)
   - Utility functions
   - Validation helpers
   - Response helpers
   - Password hashing
   - String utilities

6. **Configs Layer** (`configs/`)
   - Configuration loader (code)
   - Environment configuration

7. **Routes Layer** (`routes/`)
   - Setup routing
   - Middleware configuration
   - Route grouping

8. **Config Folder** (`config/`)
   - JSON configuration files
   - Application settings

## Setup & Installation

### Prerequisites

- Go 1.21 atau lebih baru
- PostgreSQL atau MySQL
- Git

### Installation Steps

1. **Clone repository**
```bash
git clone <repository-url>
cd service-travego
```

2. **Install dependencies**
```bash
go mod download
```

3. **Setup database**
   - Buat database baru
   - Update konfigurasi di `config/app.json`
   - Jalankan migration SQL di `db/migrations/`

4. **Setup Environment Variables**
   
   Buat file `.env.dev` (untuk development) atau `.env.production` (untuk production) di root project:
   
   ```bash
   # Application Configuration
   APP_NAME=Service TraveGO
   PORT=8080
   APP_ENV=development
   APP_ALLOW_ORIGINS=*
   
   # Note: PORT is prioritized over APP_PORT (common in cloud platforms like Heroku, Railway, etc.)
   
   # Database Configuration
   DB_DRIVER=postgres
   DB_HOST=localhost
   DB_PORT=5432
   DB_USERNAME=postgres
   DB_PASSWORD=postgres
   DB_DATABASE=travego_db
   DB_SSL_MODE=disable
   
   # JWT Configuration
   JWT_SECRET=your-secret-key-change-in-production
   JWT_EXPIRATION=24
   
   # Email SMTP Configuration (REQUIRED)
   EMAIL_FROM=your-email@gmail.com
   EMAIL_PASSWORD=your-app-password
   EMAIL_SMTP_HOST=smtp.gmail.com
   EMAIL_SMTP_PORT=587
   
   # Redis Configuration
   REDIS_HOST=localhost
   REDIS_PORT=6379
   REDIS_PASSWORD=
   REDIS_DB=0
   ```
   
   **Catatan Penting:**
   - Email configuration **HARUS** di-set melalui environment variables untuk keamanan
   - Jangan masukkan email credentials di `config/app.json`
   - Untuk Gmail, gunakan App Password (bukan password biasa)
   - File `.env.*` sudah di-ignore oleh git untuk keamanan

5. **Run application**
```bash
go run main.go
```

Server akan berjalan di `http://localhost:3000`

## API Endpoints

Semua endpoint diawali dengan `/api`

### Health Check
- `GET /api/health` - Check service status

### General
- `GET /api/general/config` - Get general configuration (company name, address, email)
- `GET /api/general/web-menu` - Get web menu structure (dashboard & landing page menus)

### Users
- `GET /api/users` - Get all users
- `GET /api/users/:id` - Get user by ID
- `POST /api/users` - Create new user
- `PUT /api/users/:id` - Update user
- `DELETE /api/users/:id` - Delete user

## Development

### Menambah Feature Baru

1. **Buat Model** di `model/`
   ```go
   type YourModel struct {
       ID uint `json:"id" gorm:"primaryKey"`
       // fields...
   }
   ```

2. **Buat Repository** di `repository/`
   ```go
   type YourRepository struct {
       db *gorm.DB
   }
   ```

3. **Buat Service** di `service/`
   ```go
   type YourService struct {
       repo *YourRepository
   }
   ```

4. **Buat Handler** di `handler/`
   ```go
   type YourHandler struct {
       service *YourService
   }
   ```

5. **Setup Routes** di `routes/routes.go`
   ```go
   yourHandler := handler.NewYourHandler(yourService)
   yourRoutes := api.Group("/your-resource")
   yourRoutes.Get("/", yourHandler.GetAll)
   ```

### Database Migrations

SQL migration files disimpan di `db/migrations/`. Untuk menjalankan migration, eksekusi file SQL secara manual atau gunakan migration tool seperti golang-migrate.

## Environment Variables

Aplikasi menggunakan environment variables untuk konfigurasi. File `.env.dev` (development) atau `.env.production` (production) akan otomatis di-load berdasarkan `APP_ENV`.

### Required Environment Variables

#### Email SMTP Configuration (Required)
- `EMAIL_FROM` - Email pengirim (contoh: your-email@gmail.com)
- `EMAIL_PASSWORD` - App password untuk email (untuk Gmail, gunakan App Password)
- `EMAIL_SMTP_HOST` - SMTP host (contoh: smtp.gmail.com)
- `EMAIL_SMTP_PORT` - SMTP port (contoh: 587 untuk TLS, 465 untuk SSL)

**Catatan:** Email configuration harus di-set melalui environment variables. Aplikasi akan error jika email config tidak lengkap.

### Optional Environment Variables

Lihat file `.env.example` untuk daftar lengkap environment variables yang didukung.

## Dependencies

- **Fiber v2** - Web framework
- **GORM** - ORM untuk database
- **validator/v10** - Struct validation
- **PostgreSQL/MySQL** - Database drivers
- **godotenv** - Environment variable loader

## Project Structure Guidelines

- **Handler**: Hanya handle HTTP, validasi input, call service
- **Service**: Business logic, validasi bisnis
- **Repository**: Hanya akses database, tidak ada business logic
- **Model**: Struktur data saja
- **Helper**: Reusable utility functions
- **Config**: Configuration management
- **Routes**: Routing setup dan middleware

## License

MIT License
