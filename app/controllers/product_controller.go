package controllers

import (
	"net/http"
	"strconv"

	"novaflow/app/models"
	"novaflow/core"
)

// ProductController is a REST resource controller: it satisfies
// core.ResourceController so it can be wired in one line with
// router.Resource("/api/v1/products", controller).
type ProductController struct {
	repo *core.Repository[models.Product]
}

func NewProductController(db *core.DB) *ProductController {
	return &ProductController{repo: core.NewRepository[models.Product](db)}
}

// GET /api/v1/products
func (pc *ProductController) Index(c *core.Context) {
	products, err := pc.repo.All()
	if err != nil {
		c.JSONError(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSONSuccess(products)
}

// GET /api/v1/products/:id
func (pc *ProductController) Show(c *core.Context) {
	product, err := pc.repo.Find(c.Param("id"))
	if err != nil {
		c.JSONError(http.StatusInternalServerError, err.Error())
		return
	}
	if product == nil {
		c.JSONError(http.StatusNotFound, "product not found")
		return
	}
	c.JSONSuccess(product)
}

// POST /api/v1/products
func (pc *ProductController) Store(c *core.Context) {
	v := core.NewValidator(map[string]string{
		"name":  c.Input("name"),
		"price": c.Input("price"),
	})
	v.Required("name").Required("price").Numeric("price")
	if !v.Passes() {
		c.JSONError(http.StatusUnprocessableEntity, v.FirstError())
		return
	}

	price, _ := strconv.ParseFloat(c.Input("price"), 64)
	product := models.Product{
		Name:   c.Input("name"),
		Price:  price,
		Status: "active",
	}
	id, err := pc.repo.Create(&product)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, err.Error())
		return
	}
	product.ID = id
	c.JSON(http.StatusCreated, map[string]interface{}{"success": true, "data": product})
}

// PUT /api/v1/products/:id
func (pc *ProductController) Update(c *core.Context) {
	product, err := pc.repo.Find(c.Param("id"))
	if err != nil {
		c.JSONError(http.StatusInternalServerError, err.Error())
		return
	}
	if product == nil {
		c.JSONError(http.StatusNotFound, "product not found")
		return
	}
	if name := c.Input("name"); name != "" {
		product.Name = name
	}
	if priceStr := c.Input("price"); priceStr != "" {
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			product.Price = price
		}
	}
	if _, err := pc.repo.Update(product); err != nil {
		c.JSONError(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSONSuccess(product)
}

// DELETE /api/v1/products/:id
func (pc *ProductController) Destroy(c *core.Context) {
	rows, err := pc.repo.Delete(c.Param("id"))
	if err != nil {
		c.JSONError(http.StatusInternalServerError, err.Error())
		return
	}
	if rows == 0 {
		c.JSONError(http.StatusNotFound, "product not found")
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"success": true, "message": "product deleted"})
}
