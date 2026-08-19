# 📊 NovaFlow Go — QueryBuilder Documentation

**NovaFlow Go QueryBuilder** হলো একটি ফ্লুয়েন্ট (Fluent), নিরাপদ এবং অত্যন্ত লাইটওয়েট SQL কোয়েরি বিল্ডার ইঞ্জিন। এটি গো-এর স্ট্যান্ডার্ড লাইব্রেরি `database/sql`-কে র‍্যাপ করে তৈরি করা হয়েছে এবং ডেভেলপারদের কোনো প্রকার SQL ইনজেকশন (SQL Injection) ঝুঁকি ছাড়াই ডাটাবেসের কাজ করতে সাহায্য করে।

---

## 🛠️ কিভাবে কাজ করে (How it Works)

QueryBuilder মূলতঃ **Incremental Construction** পদ্ধতিতে কাজ করে। এর কাজ করার ধাপগুলো নিম্নরূপ:

1. **ইনিশিয়ালাইজেশন:** `app.DB.Table("users")` কল করার মাধ্যমে একটি নির্দিষ্ট টেবিলের জন্য `QueryBuilder` অবজেক্ট তৈরি হয়।
2. **চেইনিং (Chaining):** এরপর বিভিন্ন মেথড (যেমন: `Select`, `Where`, `OrderBy`, `Limit`) কল করে কুয়েরির বিভিন্ন শর্ত যুক্ত করা হয়। এগুলো সরাসরি ডাটাবেসে কুয়েরি পাঠায় না, বরং স্টেট মেমরিতে জমিয়ে রাখে।
3. **প্যারামিটার বাইন্ডিং (Secure Binding):** ফিল্টারিং করার সময় ভ্যালুগুলোকে সরাসরি স্ট্রিং কনক্যাটিনেট না করে প্লেসহোল্ডার (`?`) হিসেবে বাইন্ড করা হয়।
4. **এক্সিকিউশন (Compilation & Execution):** যখন কোনো টার্মিনেটিং মেথড (যেমন: `Get()`, `First()`, `Count()`, `Insert()`) কল করা হয়, তখন বিল্ডার জমিয়ে রাখা সব ফিল্টারকে একত্রিত করে একটি SQL স্ট্রিং এবং আর্গুমেন্ট স্লাইস তৈরি করে ডাটাবেসে পাঠায়।

---

## 📚 ব্যবহার বিধি (Usage Guide)

নিচে QueryBuilder-এর সকল গুরুত্বপূর্ণ ফিচারের ব্যবহারিক কোড এক্সাম্পল দেওয়া হলো:

### ১. সাধারণ সিলেকশন (Select Queries)

**সব ডাটা পড়া (`Get`):**
```go
// SELECT * FROM products
products, err := app.DB.Table("products").Get()
for _, p := range products {
    fmt.Println(p["name"], p["price"])
}
```

**নির্দিষ্ট কলাম নির্বাচন এবং প্রথম রো পড়া (`First`):**
```go
// SELECT id, name FROM products WHERE id = 1 LIMIT 1
product, err := app.DB.Table("products").
    Select("id", "name").
    Where("id", "=", 1).
    First()

if product != nil {
    fmt.Println("Product Name:", product["name"])
}
```

---

### ২. শর্ত যুক্ত করা (Where Clauses)

**একাধিক AND / OR শর্ত:**
```go
// WHERE status = 'active' AND price > 500 OR featured = 1
products, err := app.DB.Table("products").
    Where("status", "=", "active").
    Where("price", ">", 500).
    OrWhere("featured", "=", 1).
    Get()
```

**Where In (তালিকা অনুসন্ধান):**
```go
// WHERE category_id IN (1, 2, 3)
products, err := app.DB.Table("products").
    WhereIn("category_id", []interface{}{1, 2, 3}).
    Get()
```

---

### ৩. ডাটা সাজানো এবং সীমা নির্ধারণ (Sorting & Pagination)

```go
// SELECT * FROM products ORDER BY price DESC LIMIT 10 OFFSET 20
products, err := app.DB.Table("products").
    OrderBy("price", "DESC").
    Limit(10).
    Offset(20).
    Get()
```

---

### ৪. রিলেশনাল জয়েন (Joins)

আমাদের বিল্ডারে **InnerJoin**, **LeftJoin** এবং **RightJoin** সাপোর্ট রয়েছে:

```go
// SELECT products.*, categories.name as category_name 
// FROM products 
// LEFT JOIN categories ON products.category_id = categories.id
products, err := app.DB.Table("products").
    Select("products.*", "categories.name as category_name").
    LeftJoin("categories", "products.category_id", "=", "categories.id").
    Get()
```

---

### ৫. ডাটা এগ্রিগেশন এবং নির্দিষ্ট কলাম ডাটা সংগ্রহ (Aggregations & Pluck)

**রো সংখ্যা গণনা (`Count`):**
```go
// SELECT COUNT(*) as cnt FROM products WHERE price > 100
count, err := app.DB.Table("products").Where("price", ">", 100).Count()
fmt.Printf("Total products: %d\n", count)
```

**নির্দিষ্ট কলামের স্লাইস সংগ্রহ (`Pluck`):**
```go
// SELECT name FROM products
// রিটার্ন করবে: []interface{}{"Laptop", "Smartphone", ...}
names, err := app.DB.Table("products").Pluck("name")
```

---

### ৬. ডাটাবেসে ডাটা যুক্ত ও পরিবর্তন করা (CRUD Mutations)

**ডাটা ইনসার্ট করা (`Insert`):**
```go
// INSERT INTO products (name, price) VALUES ('New Phone', 299.99)
newID, err := app.DB.Table("products").Insert(map[string]interface{}{
    "name":  "New Phone",
    "price": 299.99,
})
fmt.Println("Created product ID:", newID)
```

**ডাটা আপডেট করা (`Update`):**
```go
// UPDATE products SET price = 250.00 WHERE id = 5
affectedRows, err := app.DB.Table("products").
    Where("id", "=", 5).
    Update(map[string]interface{}{
        "price": 250.00,
    })
```

**ডাটা ডিলিট করা (`Delete`):**
```go
// DELETE FROM products WHERE status = 'inactive'
rowsDeleted, err := app.DB.Table("products").Where("status", "=", "inactive").Delete()
```

---

### ৭. ডাটাবেস ট্রানজেকশন (Transactions - Tx)

যেকোনো কমপ্লেক্স কাজের জন্য নিরাপদ ট্রানজেকশন ব্লক ব্যবহার করতে পারেন, যাতে কোনো এরর হলে ডাটা অটোমেটিক রোলব্যাক হয়ে যায়:

```go
tx, err := app.DB.Begin()
if err != nil {
    log.Fatal(err)
}

// ট্রানজেকশনের ভেতরে কুয়েরি রান করা (অবশ্যই tx.Table ব্যবহার করবেন)
_, err = tx.Table("products").Insert(map[string]interface{}{
    "name":  "Tx Product",
    "price": 99.00,
})

if err != nil {
    tx.Rollback() // ভুল হলে রোলব্যাক
    fmt.Println("Transaction Rolled Back due to error:", err)
    return
}

// কাজ সফল হলে ডাটাবেসে সেভ করুন
tx.Commit()
fmt.Println("Transaction committed successfully!")
```

---

### 💡 ৮. কুয়েরি ক্লোনিং (`Clone` মেথড)

একটি বেস কুয়েরি অবজেক্টকে পরবর্তীতে ভিন্ন ভিন্ন ফিল্টারিংয়ের জন্য ব্যবহার করতে চাইলে ক্লোনিং মেথড ব্যবহার করা হয়। এটি মূল কুয়েরি অবজেক্টকে অপরিবর্তিত রাখে:

```go
// বেস কুয়েরি
baseQuery := app.DB.Table("products").Where("status", "=", "active")

// ক্লোন ১ (নতুন ফিল্টার)
expensiveProducts, _ := baseQuery.Clone().Where("price", ">", 1000).Get()

// ক্লোন ২ (আরেকটি ভিন্ন ফিল্টার, ক্লোন ১ এর ফিল্টার এখানে ইফেক্ট করবে না)
cheapProducts, _ := baseQuery.Clone().Where("price", "<", 100).Get()
```

---

## 🔒 নিরাপত্তা সতর্কতা (SQL Injection Prevention)

NovaFlow Go-এর কুয়েরি বিল্ডার শতভাগ **SQL Injection** মুক্ত। এর ইন্টারনাল আর্কিটেকচার প্রতিটি ইনপুটকে ডাটাবেস লেভেলে `Prepared Statement` ও আর্গুমেন্ট হিসেবে পাঠায়। 

⚠️ **সতর্কতা:** ডাটাবেস অপারেশনে `Raw()` মেথড ব্যবহারের ক্ষেত্রে স্ট্রিং কনক্যাটিনেশন এড়িয়ে চলুন:
*   ❌ **ভুল ও বিপজ্জনক:** `app.DB.Raw("SELECT * FROM users WHERE email = '" + email + "'")`
*   ✅ **সঠিক ও নিরাপদ:** `app.DB.Raw("SELECT * FROM users WHERE email = ?", email)`
