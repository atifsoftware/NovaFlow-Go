# 🗄️ NovaFlow Go — Models & Repository Pattern Documentation

**NovaFlow Go ORM Engine** জেনেরিক টাইপ-সেফটি (Go Generics) এবং রিফ্লেকশন (Reflection) ব্যবহার করে তৈরি করা হয়েছে। এটি ডাটাবেসের রো (DB Row) এবং গো-এর স্ট্রাক্ট (Go Struct)-এর মধ্যে অটোমেটিক ডাটা ম্যাপিং করে। এর মাধ্যমে আপনি কুয়েরির রিটার্ন ডাটা সরাসরি টাইপ-সেফ স্ট্রাক্ট হিসেবে পাবেন, কোনো প্রকার কাস্টিং ছাড়াই।

---

## 📂 ১. মডেল তৈরি করার নিয়ম (Declaring a Model)

NovaFlow Go-তে যেকোনো ডাটাবেস টেবিলকে ম্যাপ করার জন্য একটি গো স্ট্রাক্ট তৈরি করতে হয় এবং স্ট্রাক্টের প্রতিটি ফিল্ডে `db:"column_name"` ট্যাগ দিতে হয়।

### মডেলের শর্ত:
প্রতিটি মডেলকে অবশ্যই `TableName() string` মেথডটি ইমপ্লিমেন্ট করতে হবে, যাতে ওআরএম জানতে পারে এটি কোন টেবিল থেকে ডাটা রিড করবে।

### মডেল স্ট্রাক্ট এক্সাম্পল:
```go
package models

import "time"

type Product struct {
	ID          int64     `db:"id"`
	Name        string    `db:"name"`
	Price       float64   `db:"price"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
}

// TableName মেথডটি মডেলকে ডাটাবেস টেবিলের সাথে যুক্ত করে
func (Product) TableName() string {
	return "products"
}
```

---

## ⚡ ২. রিপোজিটরি ইনিশিয়ালাইজেশন (Initializing Repository)

একটি মডেল নিয়ে কাজ করার জন্য প্রথমে তার জেনেরিক রিপোজিটরি অবজেক্ট তৈরি করতে হয়। 

```go
import (
	"novaflow/app/models"
	"novaflow/core"
)

// Product রিপোজিটরি তৈরি
productRepo := core.NewRepository[models.Product](app.DB)
```

---

## 💾 ৩. বেসিক CRUD অপারেশন (Basic CRUD Operations)

রিপোজিটরির মাধ্যমে ডাটাবেসে ডাটা রিড, রাইট, আপডেট এবং ডিলিট করার নিয়ম নিচে দেওয়া হলো:

### ক. আইডি দিয়ে ডাটা খোঁজা (`Find`)
```go
// SELECT * FROM products WHERE id = 5 LIMIT 1
product, err := productRepo.Find(5)
if err != nil {
	log.Fatal(err)
}

if product != nil {
	fmt.Printf("Product Found: %s ($%.2f)\n", product.Name, product.Price)
} else {
	fmt.Println("Product not found.")
}
```

### খ. সব ডাটা একত্রে পড়া (`All`)
```go
// SELECT * FROM products
products, err := productRepo.All()
for _, p := range products {
	fmt.Printf("[%d] %s - $%.2f\n", p.ID, p.Name, p.Price) // সরাসরি টাইপড ফিল্ড অ্যাক্সেস
}
```

### গ. নতুন ডাটা সংরক্ষণ করা (`Create`)
আইডি জিরো ভ্যালু (`0`) থাকলে ডাটাবেস লেভেলে তা অটো-ইনক্রিমেন্ট (Auto Increment) হয়ে সেভ হবে।
```go
newProduct := &models.Product{
	Name:        "Gaming Keyboard",
	Price:       89.99,
	Description: "RGB Mechanical Keyboard",
}

newID, err := productRepo.Create(newProduct)
if err != nil {
	log.Fatal(err)
}
fmt.Println("New Product Inserted with ID:", newID)
```

### ঘ. ডাটা আপডেট করা (`Update`)
আপডেট করার জন্য স্ট্রাক্ট অবজেক্টে অবশ্যই `ID` ফিল্ডের মান থাকতে হবে।
```go
// প্রথমে ডাটা রিড করা
product, _ := productRepo.Find(1)

// মান পরিবর্তন করা
product.Price = 1200.00
product.Name = "MacBook Pro M3 Max"

// ডাটাবেসে সেভ করা
rowsAffected, err := productRepo.Update(product)
```

### ঙ. ডাটা ডিলিট করা (`Delete`)
```go
// DELETE FROM products WHERE id = 12
rowsAffected, err := productRepo.Delete(12)
```

---

## 🔍 ৪. টাইপ-সেফ কোয়েরি ফিল্টারিং (`Where` & `TypedQuery`)

যদি আপনি কুয়েরিতে ফিল্টারিং, সর্টিং বা লিমিট ব্যবহার করতে চান এবং রিটার্ন ভ্যালু ম্যাপের পরিবর্তে সরাসরি গো স্ট্রাক্ট স্লাইস পেতে চান, তবে `Where` চেইনিং মেথড ব্যবহার করতে পারেন:

### ক. কন্ডিশনাল গেট (`Get`)
```go
// SELECT * FROM products WHERE price > 500 AND price <= 2000 ORDER BY price ASC LIMIT 5
products, err := productRepo.Where("price", ">", 500).
	Where("price", "<=", 2000).
	OrderBy("price", "ASC").
	Limit(5).
	Get() // রিটার্ন টাইপ: []models.Product

for _, p := range products {
	fmt.Println(p.Name, p.Price)
}
```

### খ. কন্ডিশনাল ফার্স্ট (`First`)
```go
// SELECT * FROM products WHERE name = 'iPad Air' LIMIT 1
product, err := productRepo.Where("name", "=", "iPad Air").First() // রিটার্ন টাইপ: *models.Product
if product != nil {
	fmt.Println("Description:", product.Description)
}
```

---

## 🛠️ ৫. CLI দিয়ে অটো-মডেল জেনারেশন (Scaffolding Models)

আপনাকে কষ্ট করে সব ফাইল হাতে তৈরি করতে হবে না। **NovaFlow CLI** কমান্ডের সাহায্যে আপনি সেকেন্ডের মধ্যে নতুন মডেল স্ট্রাকচার ফাইল জেনারেট করে নিতে পারেন:

```bash
go run ./cli make:model Category
```

**আউটপুট:**
এই কমান্ডটি `app/models/category.go` নামে একটি নতুন ফাইল তৈরি করবে যা টেবিল `categories` এর সাথে যুক্ত থাকবে:
```go
package models

type Category struct {
	ID int64 `db:"id"`
}

func (Category) TableName() string { return "categories" }
```
এরপর আপনি আপনার প্রয়োজন অনুযায়ী এর ভেতরে নতুন ফিল্ড এবং ডাটাবেস ট্যাগ ডিফাইন করতে পারেন।
