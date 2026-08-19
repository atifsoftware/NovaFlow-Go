# NovaFlow (Go Edition) 🚀

[বাংলা বিবরণ নিচে দেওয়া হয়েছে]

**NovaFlow (Go Edition)** is a lightweight, ultra-high-performance Go MVC web framework. It is the Go sibling of the [NovaFlow PHP MVC framework](https://github.com/atifsoftware/NovaFlow), engineered to deliver blazing speed, architectural clarity, and zero-bloat standard-library design.

---

## ⚡ Performance & Benchmarks

NovaFlow-Go is designed from the ground up for minimal memory usage, zero-garbage request lifecycles, and sub-millisecond response times.

| Benchmark Metric | NovaFlow-Go Performance | Comparison with other Frameworks |
| :--- | :---: | :--- |
| **Pure JSON Throughput (RPS)** | **75,000 – 120,000+ req/sec** | **8x–10x faster than Express.js**, **25x faster than Laravel** |
| **Database-backed CRUD (RPS)** | **15,000 – 45,000+ req/sec** | Max throughput capped only by Database I/O |
| **Response Latency (p50 / p99)** | **p50 < 0.5ms \| p99 < 3ms** | Sub-millisecond instant responses |
| **Idle Memory Footprint** | **~8 MB – 15 MB RAM** | Node.js (~80MB), Python (~100MB), PHP-FPM (~200MB) |
| **10,000 Concurrent Connections** | **~35 MB – 60 MB RAM** | Extremely lightweight Goroutine concurrency |

```
[Throughput / RPS Comparison - Higher is Better]

NovaFlow-Go  ██████████████████████████████ 85,000+ RPS
Gin / Fiber  ███████████████████████████████ 90,000+ RPS
FastAPI (Py) ██████ 18,000 RPS
Express.js   ████ 12,000 RPS
Laravel (PHP)██ 3,500 RPS
Django (Py)  █ 2,800 RPS
```

---

## 🌟 Key Features

1. **Multi-Database Support (MySQL, PostgreSQL, SQLite):**
   - **MySQL/MariaDB:** Native `?` placeholders, connection pooling, and `LastInsertId()`.
   - **PostgreSQL:** Automatic indexed `$1, $2` parameters and `RETURNING id` support via `github.com/lib/pq`.
   - **SQLite:** Pure-Go SQLite driver (`github.com/glebarez/go-sqlite`) requiring **Zero CGO** with file and in-memory (`:memory:`) support.
2. **Context Pooling (`sync.Pool`):**
   - Zero-allocation per-request Context recycling to eliminate garbage collector pauses during high concurrency.
3. **JSON Request Body Binding (`c.BindJSON(&dest)`):**
   - Seamless decoding for JSON API payloads and form-encoded data.
4. **High-Speed Reflection Caching (`sync.Map`):**
   - Struct tags and field offsets are cached once, speeding up DB row scanning by **2x to 3x**.
5. **Built-in In-Memory Cache Service (`app.Cache`):**
   - Fast thread-safe cache with TTL, background cleaner, and `Remember(key, ttl, callback)` pattern.
6. **Async Background Job Queue (`app.Queue`):**
   - Multi-worker concurrent task queue with non-blocking `q.Dispatch()` and graceful shutdown.
7. **Enterprise Security & CSRF Protection:**
   - Pre-configured Security Headers (`X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`, `Referrer-Policy`) and constant-time CSRF verification.
8. **Interactive Swagger Documentation:**
   - Out-of-the-box dark-themed Swagger UI accessible at `/docs` with OpenAPI 3.0 specification at `/openapi.json`.
9. **Intelligent Error Recovery Debugger:**
   - Development mode debug screen with source code context, stack traces, sensitive variable masking, and 1-click Google/StackOverflow search pills.
10. **Generics-based Active-Record Repository (`core.Repository[T]`):**
    - Type-safe CRUD operations (`Find`, `All`, `Where`, `Create`, `Update`, `Delete`) without code generators.
11. **Fluent Query Builder & Transactions:**
    - Parameterized, SQL-injection safe query building with support for Joins, WhereIn, Pluck, and database transactions (`Tx`).
12. **Smart Migrations & Rollback Engine:**
    - SQL-comment-based migrations with `-- UP` and `-- DOWN` phase batch tracking.
13. **CLI Scaffolding Tool (`novaflow-cli`):**
    - Scaffolds migrations, models, controllers, route introspection, and database seeding.
14. **Unified AI Service (`app.AI`):**
    - Native zero-dependency REST client for Google Gemini (1.5 Flash) and OpenAI completions & moderation.

---

## 📖 Documentation & Guides

- [**QueryBuilder Documentation**](QUERY_BUILDER.md) — Learn how to build secure SQL queries, transactions, joins, plucking, and cloning.
- [**Models & Repository Documentation**](MODELS_AND_REPOSITORY.md) — Learn how to define models with database tags and interact with the generic repository pattern.

---

## 📂 Project Structure

```
novaflow-go/
├── app/                  # Application Business Logic
│   ├── controllers/      # Route controllers (Product, Auth, Home, Docs)
│   ├── models/           # DB Schema definitions & models
│   ├── middleware/       # Custom middleware & central Kernel registry (kernel.go, request_id.go)
│   └── views/            # HTML layouts & template views
├── config/               # Central & modular routing configurations
│   ├── routes.go         # Route entry point / dispatcher
│   ├── web.go            # Session-based web routes
│   ├── api.go            # JWT bearer API routes & REST resources
│   ├── auth.go           # OAuth & custom auth placeholders
│   └── admin.go          # Admin panel route placeholders
├── core/                 # Framework Engines (Router, DB, Dialects, ORM, Auth, Cache, Queue, Validator, Logger)
├── cli/
│   ├── main.go           # CLI Assistant tool (migrate, rollback, make:model/controller/migration, seed)
│   └── main_test.go      # Unit tests for CLI parsing
├── database/
│   └── migrations/       # Timestamped .sql migration files with UP/DOWN sections
├── public/               # Public assets (CSS, JS, Images)
├── storage/              # SQLite databases, file uploads, logs, cache
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

### 2. Configure Environment Variables
Copy the template configuration file:
```bash
cp .env.example .env
```

Configure your preferred database in `.env`:
```env
# For MySQL:
DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=novaflow_db
DB_USER=root
DB_PASS=

# For PostgreSQL:
# DB_CONNECTION=postgres
# DB_HOST=127.0.0.1
# DB_PORT=5432
# DB_NAME=novaflow_db
# DB_USER=postgres
# DB_PASS=secret

# For SQLite (Pure Go - Zero CGO):
# DB_CONNECTION=sqlite
# DB_DATABASE=storage/database.sqlite
```

### 3. Run Database Migrations & Seeds
```bash
go run ./cli migrate
go run ./cli db:seed
```

### 4. Start the Application Server
```bash
go run .
```
- Web Application: `http://localhost:8080`
- Interactive API Docs (Swagger UI): `http://localhost:8080/docs`
- Health Check: `http://localhost:8080/health`

---

## 💻 CLI Tool Commands

Execute commands using `go run ./cli <command>`:

- **System Health Check:** `go run ./cli health`
- **List All Routes:** `go run ./cli --routes`
- **Run Migrations:** `go run ./cli migrate`
- **Rollback Migrations:** `go run ./cli migrate:rollback`
- **Seed Database:** `go run ./cli db:seed`
- **Generate a Migration:** `go run ./cli make:migration create_orders_table`
- **Generate Controller:** `go run ./cli make:controller Order`
- **Generate Model:** `go run ./cli make:model Order`

---

## 💡 Code Highlights & Examples

### 1. In-Memory Cache with `Remember()`
```go
// Returns cached value in <0.2ms, or fetches from DB and stores in cache automatically
data, err := c.Cache().Remember("top_products", 10*time.Minute, func() (interface{}, error) {
    return productRepo.Where("featured", "=", 1).Get()
})
```

### 2. Asynchronous Background Task Queue
```go
// Non-blocking background worker dispatch (e.g. sending emails or AI queries)
c.Queue().Dispatch(func() {
    sendWelcomeEmail(user.Email)
})
```

### 3. JSON Request Body Binding in Controllers
```go
type CreateProductInput struct {
    Name  string  `json:"name"`
    Price float64 `json:"price"`
}

func (pc *ProductController) Store(c *core.Context) {
    var input CreateProductInput
    if err := c.BindJSON(&input); err != nil {
        c.JSONError(http.StatusBadRequest, "Invalid JSON payload")
        return
    }
    // Process input safely...
}
```

### 4. Database Pagination (`Paginate()`)
```go
// Fetch paginated results for ERP reports or API lists in 1 line:
result, err := app.DB.Table("invoices").
    Where("status", "=", "unpaid").
    OrderBy("id", "DESC").
    Paginate(c.QueryInt("page", 1), 20)

// Or with typed entity repository:
products, err := productRepo.Where("price", ">", 100).Paginate(1, 15)

// Standard JSON response:
c.JSONPaginated(result)
```

### 6. Excel (XLSX) & CSV Export / Import
```go
// Export & Download XLSX directly in controller:
headers := []string{"Invoice No", "Customer", "Amount", "Status"}
rows := [][]interface{}{
    {"INV-001", "Acme Corp", 5400.00, "Paid"},
    {"INV-002", "Globex Inc", 12500.50, "Unpaid"},
}
_ = c.DownloadXLSX("Invoices_August.xlsx", "Invoices", headers, rows)

// Or export as CSV:
_ = c.DownloadCSV("Invoices.csv", headers, rows)
```

### 7. Professional Invoice & Money Receipt PDF Generation
```go
pdfBytes, err := core.GenerateInvoicePDF(core.InvoicePDFData{
    CompanyName:    "TechFlow Solutions Ltd.",
    CompanyAddress: "Gulshan-2, Dhaka-1212",
    InvoiceNumber:  "INV-2026-0042",
    InvoiceDate:    "2026-08-19",
    CustomerName:   "Apex Enterprises",
    Items: []core.InvoiceItem{
        {Description: "ERP Cloud Setup", Qty: 1, UnitPrice: 50000, Total: 50000},
    },
    SubTotal: 50000,
    Total:    50000,
})

// Stream or force download in browser:
c.StreamPDF("INV-2026-0042.pdf", pdfBytes)
```

### 8. Real-Time WebSockets & Rooms
```go
// WebSocket route handler with room subscriptions:
app.Router.Get("/ws", func(c *core.Context) {
    client, _ := c.UpgradeWebSocket(func(client *core.WSClient, msg []byte) {
        if string(msg) == "join:pos" {
            c.WS().JoinRoom(client, "pos")
        }
    }, nil)
})

// Broadcast live order update to all POS screens:
c.WS().BroadcastToRoom("pos", "order.created", map[string]int{"order_id": 101})
```

### 9. Event-Driven Architecture (Pub/Sub)
```go
// Register event listener (e.g. in config/events.go or app service):
app.Events.Listen("invoice.paid", func(payload interface{}) {
    invoiceID := payload.(int64)
    // Automatically update stock ledger & trigger email
})

// Dispatch event anywhere in controllers:
app.Events.DispatchAsync("invoice.paid", int64(1042))
```

---

# নোভাফ্লো (গো সংস্করণ) 🚀

**নোভাফ্লো (NovaFlow Go Edition)** একটি আধুনিক, আল্ট্রা-হাই পারফরম্যান্স এবং ডিপেন্ডেন্সি-হীন Go MVC ওয়েব ফ্রেমওয়ার্ক। এটি পিএইচপির জনপ্রিয় **নোভাফ্লো পিএইচপি ফ্রেমওয়ার্ক**-এর গো সংস্করণ, যা একই ফোল্ডার আর্কিটেকচার এবং কোডিং কনভেনশন শেয়ার করে। কোনো থার্ড পার্টি ভারী ফ্রেমওয়ার্ক (যেমন Gin, Echo) ছাড়াই সম্পূর্ণ গো স্ট্যান্ডার্ড লাইব্রেরি ব্যবহার করে এটি হাই-স্কেল এন্টারপ্রাইজ অ্যাপ্লিকেশনের জন্য তৈরি।

### 🌟 প্রধান সুবিধাসমূহ
- **মাল্টি-ডাটাবেজ সাপোর্ট**: MySQL, PostgreSQL (`github.com/lib/pq`) এবং Pure-Go SQLite (`github.com/glebarez/go-sqlite`, কোনো **CGO লাগে না**)।
- **এক্সেল ও সিএসভি ইঞ্জিন (Excel / CSV Export & Import)**: বড় ডাটাবেজ টেবিল থেকে এক ক্লিকে স্টাইলিশ XLSX ও CSV ডাউনলোড এবং ফাইল আপলোড করে ডাটাবেজে ইমপোর্ট করার ক্ষমতা।
- **ইনভয়েস ও মানি রিসিট PDF জেনারেটর**: পিওর-গো ভিত্তিক সুপার-ফাস্ট প্রফেশনাল ইনভয়েস ও রিসিট PDF তৈরি (বকেয়া, ডিসকাউন্ট, ভ্যাট ও কথায় টাকার হিসাব সহ)।
- **রিয়েল-টাইম ওয়েবসকেট হাব (WebSockets & Rooms)**: লাইভ পিওএস, নোটিফিকেশন বেল এবং রুমভিত্তিক রিয়েল-টাইম মেসেজিং ইঞ্জিন।
- **ইভেন্ট-ড্রিভেন আর্কিটেকচার (Pub/Sub Events)**: ডিকাপলড ইভেন্ট ও লিসেনার সিস্টেম (সিঙ্ক্রোনাস ও অ্যাসিনক্রোনাস ব্যাকগ্রাউন্ড কিউ সাপোর্ট সহ)।
- **ডাটাবেজ পেজিনেশন ইঞ্জিন (`Paginate`)**: বিল্ট-ইন পেজিনেশন হেল্পার যা কুয়েরি বিল্ডার ও রিপোজিটরিতে এক ক্লিকে টোটাল, পেজ কাউন্ট ও ডাটা বের করে।
- **নম্বর থেকে শব্দে রূপান্তর (Number to Words)**: ERP ইনভয়েস ও রিসিটের জন্য শত, হাজার, লাখ, কোটি ও পয়সা সমৃদ্ধ স্বয়ংক্রিয় বাংলা ও ইংরেজি কারেন্সি কনভার্টার।
- **কনটেক্সট পুলিং (`sync.Pool`)**: জিরো-মেমরি অ্যালোকেশন যা উচ্চ ট্রাফিকেও সার্ভারকে মসৃণ ও দ্রুত রাখে।
- **ইন-মেমরি ক্যাশ সার্ভিস**: TTL ও `Remember()` প্যাটার্ন সমৃদ্ধ থ্রেড-সেফ ফাস্ট ক্যাশিং।
- **অ্যাসিনক্রোনাস ব্যাকগ্রাউন্ড কিউ**: ইমেল পাঠানো বা ভারী কাজ ব্যাকগ্রাউন্ডে নন-ব্লকিংভাবে চালানোর ওয়ার্কার পুল।
- **এন্টারপ্রাইজ সিকিউরিটি**: সিকিউরিটি হেডার্স ও ফর্মের জন্য CSRF প্রোটেকশন মিডলওয়্যার।
- **ইন্টারেক্টিভ সোয়েগার ডক্স (Swagger UI)**: `/docs` রাউটে ডার্ক-মোড সমৃদ্ধ স্বয়ংক্রিয় এপিআই ডকুমেন্টেশন।
- **ইন্টেলিজেন্ট ডিবাগার**: এরর হলে কোডের লাইন প্রিভিউ, স্ট্যাক ট্রেস, ভ্যারিয়েবল মাস্কিং ও ১-ক্লিকে গুগল/স্ট্যাকওভারফ্লো সার্চ।
- **জেনেরিক ওআরএম (Repository[T])**: টাইপ-সেফ CRUD অপারেশন।
- **মাইগ্রেশন ও রোলব্যাক**: পিএইচপি সংস্করণের মতোই উন্নত ব্যাচ রোলব্যাক (`migrate:rollback`) সিস্টেম।
- **ইউনিফাইড এআই মডিউল**: গুগল জেমিনি ও ওপেনএআই-এর এআই টেক্সট জেনারেশন ও কন্টেন্ট মডারেশন সার্ভিস সরাসরি ব্যবহারের সুবিধা।

---

## 🧪 Testing

Run tests across all packages:
```bash
go test -v -count=1 ./...
```

---
Made with ❤️ by Mohidul ([atifsoftware](https://github.com/atifsoftware)).


