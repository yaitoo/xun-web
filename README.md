# Xun Web Starter

A modern web application starter featuring the Xun framework, htmx for interactivity, SQLite for data persistence, and TailwindCSS for styling.

## Tech Stack

- **Xun** - Go web framework with SSR support
- **htmx** - Dynamic HTML updates without full page reloads
- **SQLite** - Zero-configuration file-based database via yaitoo/sqle
- **TailwindCSS** - Utility-first CSS framework with dark tech theme

## Project Structure

```
xun-web/
├── package.json              # Shared Node tooling (Tailwind)
├── tailwind.config.js       # Dark tech theme configuration
├── tailwind.css              # Base styles
├── Makefile                  # Dev workflow automation
├── go.mod                    # Go module
└── cmd/app/
    ├── main.go               # Application entry point
    ├── app.yml               # Configuration
    └── app/
        ├── pages/            # SSR page templates (PageRoute)
        ├── layouts/          # Base and dashboard layouts
        ├── components/       # Reusable UI components
        ├── views/            # htmx partial view templates
        ├── public/           # Static assets
        └── migration/        # sqle migration files
```

## Quick Start

### 1. Install Dependencies

```bash
# Install Node modules (TailwindCSS)
npm install

# Install Go dependencies
go mod tidy
```

### 2. Run Migrations

```bash
# Run database migrations to create tables
go run cmd/app/main.go -migrate
```

### 3. Start Development Server

```bash
# Option 1: Using Make
make dev

# Option 2: Manual
# Terminal 1: Watch Tailwind CSS
npm run watch

# Terminal 2: Run Go app
go run cmd/app/main.go
```

Visit http://localhost:8080

### 4. Production Build

```bash
# Compile Tailwind CSS
npm run build

# Build Go binary
go build -o bin/xun-web ./cmd/app
```

## Features

### Server-Side Rendering (SSR)
Pages are rendered server-side using Xun's PageRoute system for fast initial loads and SEO-friendly HTML.

### htmx Integration
Interactive features like user creation and editing update the page partially without full reloads:

- User list updates inline when creating/editing users
- Form validation errors display without page refresh
- Smooth transitions between states

### Asset Hashing
Compiled assets include cache-busting hashes via Xun's asset helper:

```html
<link rel="stylesheet" href="{{asset "/public/assets/app.css"}}">
```

### Dark Tech Theme
TailwindCSS configured with a dark color palette:

| Color | Hex | Usage |
|-------|-----|-------|
| Cyan | `#00d9ff` | Primary accents |
| Purple | `#7c3aed` | Secondary accents |
| Green | `#10b981` | Success states |
| Dark | `#0f1117` | Backgrounds |

## Demo Account

After running migrations, a demo user is created:

- Email: `demo@example.com`
- Password: `password` (SHA256 hash in DB)

## Available Routes

| Route | Method | Description |
|-------|--------|-------------|
| `/` | GET | Landing page |
| `/login` | GET/POST | Login page |
| `/register` | GET/POST | Registration |
| `/logout` | POST | Logout |
| `/dashboard` | GET | Dashboard overview |
| `/dashboard/users` | GET/POST | User list & create |
| `/dashboard/users/:id` | PUT/DELETE | User update/delete |
| `/dashboard/views/users/*` | GET | HTMX partial views |

## API Commands

```bash
# Run migrations only
go run cmd/app/main.go -migrate

# Start on custom port
go run cmd/app/main.go -addr :3000
```

## Database

SQLite file: `xun-web.db`

The `users` table structure:
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL
);
```

## Makefile Commands

```bash
make install   # Install Node dependencies
make dev       # Development mode
make build     # Production CSS build
make watch     # Watch Tailwind CSS
make run       # Run Go app
make migrate   # Run migrations
make clean     # Clean compiled assets
make tidy      # Tidy Go modules
```

## License

MIT
