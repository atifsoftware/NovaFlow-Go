package models

// Product is an example entity showing how NovaFlow's generic
// core.Repository[T] gives you Active-Record-like methods without an ORM
// generator: core.NewRepository[Product](db).All() / .Find(id) / .Create(&p) ...
type Product struct {
	ID     int64   `db:"id"`
	Name   string  `db:"name"`
	Price  float64 `db:"price"`
	Status string  `db:"status"`
}

// TableName satisfies core.Model.
func (Product) TableName() string { return "products" }
