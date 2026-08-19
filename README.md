# NovaFlow (Go Edition) 🚀

[বাংলা বিবরণ নিচে দেওয়া হয়েছে]

**NovaFlow (Go Edition)** is a lightweight, dependency-minimal, high-performance Go MVC web framework. It is the Go sibling of the [NovaFlow PHP MVC framework](https://github.com/atifsoftware/NovaFlow), designed to share the same folder conventions, naming standards, and architectural clarity. Built entirely from scratch using the Go standard library (without Gin, Echo, or Fiber), it only relies on the MySQL driver.

---

## 🌟 Key Features

1. **MVC Architecture:** Structured and clean separation of `app/controllers`, `app/models`, and `app/views`.
2. **Custom Router:** Supports dynamic parameters (`:id`), wildcards, prefix routing groups, and per-group/per-route middleware.
3. **Generics-based ORM (Repository):** Active-Record-style repository pattern `core.Repository[T]` for easy database CRUD operations without writing raw SQL.
4. **Fluent Query Builder:** Construct secure parameterized SQL queries safely and programmatically.
5. **CLI Scaffolding Tool:** Helper utility (`cli/main.go`) to check health, list routes, generate migrations, controllers, models, and run rollbacks.
6. **Smart Migrations with Rollback:** SQL-comment-based migrations with support for `-- UP` and `-- DOWN` phases executed in managed batches.
7. **Structured Logging (`log/slog`):** Out-of-the-box structured logging. Automatically outputs JSON logs in `production` and developer-friendly key-value pairs locally.
8. **Struct-Tag Validation:** Perform declarative validator tag parsing (`validate:"required,email,min=6"`) utilizing Go reflection.
9. **Authentication Modules:** Cookie-based session authentication for web routes, and HS256 JWT-based token authentication for REST APIs.
10. **Rate Limiting Middleware:** Simple memory-based IP rate limiter to protect endpoints from brute-force attacks.

---

## 📖 Documentation & Guides

Detailed documentation and guides for core framework features:
- [**QueryBuilder Documentation**](QUERY_BUILDER.md) — Learn how to build secure SQL queries, transactions, joins, plucking, and cloning.
- [**Models & Repository Documentation**](MODELS_AND_REPOSITORY.md) — Learn how to define models with database tags and interact with the generic repository pattern.

---

## 📂 Project Structure

```
novaflow-go/
├── app/                  # Application Business Logic
│   ├── controllers/      # Route controllers (Product, Auth, Home)
│   ├── models/           # DB Schema definitions & models
│   ├── middleware/       # Custom middleware & central Kernel registry (kernel.go, request_id.go)
│   └── views/            # HTML layouts & template views
├── config/               # Central & modular routing configurations
│   ├── routes.go         # Route entry point / dispatcher
│   ├── web.go            # Session-based web routes
│   ├── api.go            # JWT bearer API routes
│   ├── auth.go           # OAuth & custom auth placeholders
│   └── admin.go          # Admin panel route placeholders
├── core/                 # Framework Engines (Router, DB, ORM, Auth, Validator, Logger)
├── cli/
│   ├── main.go           # CLI Assistant tool (migrate, rollback, make:model/controller/migration)
│   └── main_test.go      # Unit tests for CLI parsing
├── database/
│   └── migrations/       # Timestamped .sql migration files with UP/DOWN sections
├── public/               # Public assets (CSS, JS, Images)
├── storage/              # File uploads, logs, cache
├── tests/                # Core and route unit tests
├── .env                  # Local environment file
├── go.mod                # Module definition
└── main.go               # Framework entry point
```

---

## 🛠️ Installation & Setup

### 1. Clone the Repository
```bash
git clone https://github.com/atifsoftware/NovaFlow-Go.git myproject
cd myproject
```

### 2. Configure Environment variables
Copy the template configuration file:
```bash
cp .env.example .env
```
Update database connection settings (`DB_HOST`, `DB_NAME`, `DB_USER`, `DB_PASS`) in `.env`.

### 3. Run Database Migrations
Use the built-in migration runner to automatically initialize the database schema:
```bash
go run ./cli migrate
```

### 4. Start the Application Server
```bash
go run .
```
Visit `http://localhost:8080` in your web browser.

---

## 💻 CLI Tool Commands

The CLI assistant is located in `cli/main.go`. Execute commands using `go run ./cli <command>`:

- **System Health Check:** Check database ping and environment status.
  ```bash
  go run ./cli health
  ```
- **List All Routes:** See all mapped URLs, HTTP methods, and routes.
  ```bash
  go run ./cli --routes
  ```
- **Run Migrations:** Execute all pending `-- UP` SQL statements.
  ```bash
  go run ./cli migrate
  ```
- **Rollback Migrations:** Revert the last applied batch of migrations using `-- DOWN` SQL statements.
  ```bash
  go run ./cli migrate:rollback
  ```
- **Generate a Migration:** Create a new timestamped migration file pre-scaffolded with `-- UP`/`-- DOWN` sections.
  ```bash
  go run ./cli make:migration create_orders_table
  ```
- **Generate Controller:** Scaffold a new controller.
  ```bash
  go run ./cli make:controller Order
  ```
- **Generate Model:** Scaffold a new entity model.
  ```bash
  go run ./cli make:model Order
  ```

---

## 📖 Practical Examples

### 1. Struct-Tag Validation
Instead of manual validation, validate incoming request bodies directly using reflection:

```go
type RegisterRequest struct {
    Name     string `validate:"required"`
    Email    string `validate:"required,email"`
    Password string `validate:"required,min=6"`
}

// Inside your controller:
req := RegisterRequest{
    Name:     c.Input("name"),
    Email:    c.Input("email"),
    Password: c.Input("password"),
}

v := core.NewValidator(nil).ValidateStruct(req)
if !v.Passes() {
    c.JSONError(http.StatusUnprocessableEntity, v.FirstError())
    return
}
```

### 2. Migration Formatting
Every migration under `database/migrations/` consists of separate blocks for upgrading and downgrading:

```sql
-- UP
CREATE TABLE IF NOT EXISTS products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    price DECIMAL(10,2) NOT NULL
);

-- DOWN
DROP TABLE IF EXISTS products;
```

---

---

# নোভাফ্লো (গো সংস্করণ) 🚀

**নোভাফ্লো (NovaFlow Go Edition)** একটি দ্রুতগতির, হালকা এবং ডিপেন্ডেন্সি-হীন Go MVC ওয়েব ফ্রেমওয়ার্ক। এটি পিএইচপির জনপ্রিয় **নোভাফ্লো পিএইচপি ফ্রেমওয়ার্ক**-এর গো সংস্করণ, যা একই ফোল্ডার আর্কিটেকচার এবং কোডিং কনভেনশন শেয়ার করে। কোনো থার্ড পার্টি ফ্রেমওয়ার্ক (যেমন Gin, Echo) ছাড়াই সম্পূর্ণ গো স্ট্যান্ডার্ড লাইব্রেরি ব্যবহার করে এটি স্ক্র্যাচ থেকে ডিজাইন করা হয়েছে।

### 🌟 প্রধান সুবিধাসমূহ
- **ক্লিন MVC আর্কিটেকচার**: আলাদা বিজনেস লজিক, মডেল ও ভিউ কন্ট্রোল।
- **কাস্টম রাউটিং**: ডাইনামিক রাউট (`:id`) ও গ্রুপ রাউটিং সাপোর্ট।
- **জেনেরিক ও আর এম (ORM)**: `core.Repository[T]` ব্যবহার করে SQL না লিখে অবজেক্ট ওরিয়েন্টেড উপায়ে CRUD অপারেশন।
- **মাইগ্রেশন ও রোলব্যাক**: পিএইচপি সংস্করণের মতোই উন্নত রোলব্যাক (`migrate:rollback`) সিস্টেম।
- **স্ট্রাকচার্ড লগিং**: স্ট্যান্ডার্ড `log/slog` সমৃদ্ধ প্রোডাকশন রেডি জেসন ও টেক্সট লগিং।
- **রিফ্লেক্টিভ ভ্যালিডেশন**: গো স্ট্রাক্ট ফিল্ডে `validate:"required"` ট্যাগের সহজ ব্যবহার।

---

## 📖 ডকুমেন্টেশন ও গাইডসমূহ

ফ্রেমওয়ার্কের মূল ফিচারগুলোর বিস্তারিত গাইড এখানে দেখুন:
- [**কোয়েরি বিল্ডার (QueryBuilder) ডকুমেন্টেশন**](QUERY_BUILDER.md) — ডাটাবেস কোয়েরি, ট্রানজেকশন, জয়েন এবং ক্লোনিং ব্যবহারের নিয়ম।
- [**মডেল ও রিপোজিটরি (Models & Repository) ডকুমেন্টেশন**](MODELS_AND_REPOSITORY.md) — টাইপ-সেফ ডাটা মডেল তৈরি এবং রিপোজিটরি ব্যবহারের নিয়ম।

---

## 🧪 Testing

Run tests across all packages:
```bash
go test ./...
```

---
Made with ❤️ by Mohidul (atifsoftware).
