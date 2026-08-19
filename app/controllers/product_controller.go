package controllers

import (
	"net/http"
	"strconv"
	"strings"

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

type productPayload struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// POST /api/v1/products
func (pc *ProductController) Store(c *core.Context) {
	var payload productPayload

	if strings.Contains(c.Request.Header.Get("Content-Type"), "application/json") {
		if err := c.BindJSON(&payload); err != nil {
			c.JSONError(http.StatusBadRequest, "invalid JSON payload")
			return
		}
	} else {
		payload.Name = c.Input("name")
		payload.Price, _ = strconv.ParseFloat(c.Input("price"), 64)
	}

	v := core.NewValidator(map[string]string{
		"name":  payload.Name,
		"price": strconv.FormatFloat(payload.Price, 'f', -1, 64),
	})
	v.Required("name").Required("price").Numeric("price")
	if payload.Price <= 0 {
		v.Required("price") // ensures price is provided and positive
	}
	if !v.Passes() {
		c.JSONError(http.StatusUnprocessableEntity, v.FirstError())
		return
	}

	product := models.Product{
		Name:   payload.Name,
		Price:  payload.Price,
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

	if strings.Contains(c.Request.Header.Get("Content-Type"), "application/json") {
		var payload productPayload
		if err := c.BindJSON(&payload); err != nil {
			c.JSONError(http.StatusBadRequest, "invalid JSON payload")
			return
		}
		if payload.Name != "" {
			product.Name = payload.Name
		}
		if payload.Price > 0 {
			product.Price = payload.Price
		}
	} else {
		if name := c.Input("name"); name != "" {
			product.Name = name
		}
		if priceStr := c.Input("price"); priceStr != "" {
			if price, err := strconv.ParseFloat(priceStr, 64); err == nil && price > 0 {
				product.Price = price
			}
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

