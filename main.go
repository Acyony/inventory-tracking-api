package main

import (
	"encoding/json"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"net/http"
	"strconv"
)

// Product => representation of the product in the database.
type Product struct {
	gorm.Model
	Name        string
	Description string
	Price       float64
	Quantity    uint
	Category    string
}

// CreateProductRequest => representation of the product from the request.
type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Quantity    uint    `json:"uint"`
	Category    string  `json:"category"`
}

// ListProductResponse => representation of the product in the response.
type ListProductResponse struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Quantity    uint    `json:"uint"`
	Category    string  `json:"category"`
}

func AddNewProduct(
	db *gorm.DB, name string, description string, price float64, quantity uint, category string) (uint, error) {
	var err error

	p := Product{
		Name:        name,
		Description: description,
		Price:       price,
		Quantity:    quantity,
		Category:    category,
	}

	result := db.Create(&p)
	if result.Error != nil {
		return 0, result.Error
	}

	return p.ID, err
}

func ListProducts(db *gorm.DB) ([]*Product, error) {
	products := []*Product{}
	result := db.Find(&products)
	return products, result.Error
}

func ListProductByCategory(db *gorm.DB, category string) ([]*Product, error) {
	products := []*Product{}
	result := db.Find(&products, "category = ?", category)
	return products, result.Error
}

func main() {
	// connecting to sqlite database
	db, err := gorm.Open(sqlite.Open("database_gorm.sqlite3"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic(fmt.Sprintf("Not able to connect to database: %s", err.Error()))
	}

	// Create the required Products tables
	if err := db.AutoMigrate(&Product{}); err != nil {
		panic(fmt.Sprintf("Not able to create a table %s", err.Error()))
	}

	// Creates a new product
	http.HandleFunc("/new-product", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			fmt.Println("Not possible to create a product")
			return
		}

		productFromRequest := CreateProductRequest{}
		if err := json.NewDecoder(r.Body).Decode(&productFromRequest); err != nil {
			fmt.Println("unable to decode product from request")
			http.Error(w, "unable to decode product from request", http.StatusBadRequest)
			return
		}

		if id, err := AddNewProduct(
			db,
			productFromRequest.Name,
			productFromRequest.Description,
			productFromRequest.Price,
			productFromRequest.Quantity,
			productFromRequest.Category,
		); err == nil {
			_, _ = w.Write([]byte(strconv.Itoa(int(id))))
			return
		}

		http.Error(
			w,
			"unable to save the product in the database",
			http.StatusBadRequest,
		)
	})

	// List all products
	http.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		productsFromDatabase, err := ListProducts(db)
		if err != nil {
			http.Error(w, "unable to list all products", http.StatusInternalServerError)
			return
		}

		productsResponse := []ListProductResponse{}

		for _, p := range productsFromDatabase {
			productsResponse = append(productsResponse, ListProductResponse{
				ID:          p.ID,
				Name:        p.Name,
				Description: p.Description,
				Price:       p.Price,
				Quantity:    p.Quantity,
				Category:    p.Category,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(productsResponse); err != nil {
			http.Error(w, "unable to encode products into request", http.StatusInternalServerError)
		}
	})

	// Get one product by category
	http.HandleFunc("/products-by-category", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		category := r.URL.Query().Get("category")

		productsByCategoryFromDatabase, err := ListProductByCategory(db, category)
		if err != nil {
			http.Error(w, "unable to list all products", http.StatusInternalServerError)
			return
		}

		productsResponse := []ListProductResponse{}
		for _, pbc := range productsByCategoryFromDatabase {
			productsResponse = append(productsResponse, ListProductResponse{
				ID:          pbc.ID,
				Name:        pbc.Name,
				Description: pbc.Description,
				Price:       pbc.Price,
				Quantity:    pbc.Quantity,
				Category:    pbc.Category,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(productsResponse); err != nil {
			http.Error(w, "unable to encode products into request", http.StatusInternalServerError)
		}
	})

	fmt.Println("Press 'CONTROL' + 'C' to stop the server")
	http.Handle("/", http.FileServer(http.Dir("templates")))
	log.Fatal(http.ListenAndServe(":8724", nil))
}
