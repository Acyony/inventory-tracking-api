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

// -----==^.^==----- Product => representation of the product in the database ------==^.^==------

type Product struct {
	gorm.Model
	Name        string
	Description string
	Price       float64
	Quantity    uint
	Category    string
}

// -----==^.^==----- CreateUpdateProductRequest => representation of the product from the request ------==^.^==------

type CreateUpdateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Quantity    uint    `json:"quantity"`
	Category    string  `json:"category"`
}

// -----==^.^==----- ListProductResponse => representation of the product in the response ------==^.^==------

type ListProductResponse struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Quantity    uint    `json:"quantity"`
	Category    string  `json:"category"`
}

// -----==^.^==----- Func => Creates a new product ------==^.^==------

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

// -----==^.^==----- Func => Delete products ------==^.^==------

func DeleteProduct(db *gorm.DB, productID uint) error {
	result := db.Where("id = ?", productID).Delete(&Product{})
	return result.Error
}

// -----==^.^==----- Func => UndoDelete products ------==^.^==------

func UndoDelete(db *gorm.DB, productID uint) error {
	tx := db.Exec("UPDATE products SET deleted_at = NULL WHERE id = ?", productID)
	return tx.Error
}

// -----==^.^==----- Func => List all products ------==^.^==------

func ListProducts(db *gorm.DB) ([]*Product, error) {
	products := []*Product{}
	result := db.Find(&products)
	return products, result.Error
}

func GetProduct(db *gorm.DB, productId uint) (*Product, error) {
	product := &Product{}
	result := db.Find(&product, "id = ?", productId)
	return product, result.Error
}

// -----==^.^==----- Func => List all deleted products ------==^.^==------

func ListAllDeletedProducts(db *gorm.DB) ([]*Product, error) {
	products := []*Product{}
	result := db.Raw("SELECT * FROM products WHERE deleted_at is not null").Scan(&products)
	return products, result.Error
}

// -----==^.^==----- Func => Update a product ------==^.^==------

func UpdateProduct(db *gorm.DB, productID uint, name string, description string, price float64, quantity uint, category string) error {
	product := &Product{}
	res := db.First(&product, productID)
	if res.Error != nil {
		return res.Error
	}

	product.Name = name
	product.Description = description
	product.Price = price
	product.Category = category
	product.Quantity = quantity
	res = db.Save(product)
	return res.Error
}

func main() {
	// -----==^.^==----- connecting to sqlite database ------==^.^==------
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

	http.Handle("/assets/",
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	// -----==^.^==----- Creates a new product ------==^.^==------
	http.HandleFunc("/new-product", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			fmt.Println("Not possible to create a product")
			return
		}

		productFromRequest := CreateUpdateProductRequest{}
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

	// -----==^.^==----- Delete a product ------==^.^==------
	http.HandleFunc("/delete-product", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			fmt.Println("Not possible to delete the product")
			http.Error(w, "Not possible to delete the product", http.StatusMethodNotAllowed)
			return
		}

		productIDStr := r.URL.Query().Get("id")
		productID, err := strconv.Atoi(productIDStr)
		if err != nil {
			http.Error(w, "unable to decode product from request", http.StatusBadRequest)
			return
		}

		if err := DeleteProduct(db, uint(productID)); err != nil {
			fmt.Println("unable to delete the product from DB")
			http.Error(w, "unable to delete the product from DB", http.StatusBadRequest)
			return
		}
	})

	// -----==^.^==----- Func => UndoDelete products ------==^.^==------
	http.HandleFunc("/undo-delete-product", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			fmt.Println("Not possible to restore the deleted product")
			http.Error(w, "Not possible to restore the deleted product", http.StatusMethodNotAllowed)
			return
		}

		productIDStr := r.URL.Query().Get("id")
		productID, err := strconv.Atoi(productIDStr)
		if err != nil {
			http.Error(w, "unable to decode product from request", http.StatusBadRequest)
			return
		}

		if err := UndoDelete(db, uint(productID)); err != nil {
			fmt.Println("Not possible to restore the deleted product")
			http.Error(w, "Not possible to restore the deleted product", http.StatusInternalServerError)
			return
		}

		http.Error(
			w,
			"product successfully restored",
			http.StatusBadRequest,
		)
	})
	// -----==^.^==----- Update a new product------==^.^==------
	http.HandleFunc("/update-product", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			fmt.Println("Not possible to update a product")
			http.Error(w, "Not possible to update a product", http.StatusMethodNotAllowed)
			return
		}

		productIDStr := r.URL.Query().Get("id")
		productID, err := strconv.Atoi(productIDStr)
		if err != nil {
			http.Error(w, "unable to decode product from request", http.StatusBadRequest)
			return
		}

		productFromRequest := CreateUpdateProductRequest{}
		if err := json.NewDecoder(r.Body).Decode(&productFromRequest); err != nil {
			fmt.Println("unable to decode product from request")
			http.Error(w, "unable to decode product from request", http.StatusBadRequest)
			return
		}

		if err := UpdateProduct(db, uint(productID), productFromRequest.Name, productFromRequest.Description, productFromRequest.Price, productFromRequest.Quantity, productFromRequest.Category); err == nil {
			_, _ = w.Write([]byte(strconv.Itoa(productID)))
			return
		}

		http.Error(
			w,
			"unable to save the product in the database",
			http.StatusBadRequest,
		)
	})

	// -----==^.^==----- List all products ------==^.^==------
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

	http.HandleFunc("/product", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		productIDStr := r.URL.Query().Get("id")
		productID, err := strconv.Atoi(productIDStr)
		if err != nil {
			http.Error(w, "unable to decode product from request", http.StatusBadRequest)
			return
		}

		productFromDatabase, err := GetProduct(db, uint(productID))
		if err != nil {
			http.Error(w, "unable to get the product", http.StatusInternalServerError)
			return
		}

		productResponse := ListProductResponse{
			ID:          productFromDatabase.ID,
			Name:        productFromDatabase.Name,
			Description: productFromDatabase.Description,
			Price:       productFromDatabase.Price,
			Quantity:    productFromDatabase.Quantity,
			Category:    productFromDatabase.Category,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(productResponse); err != nil {
			http.Error(w, "unable to encode product into response", http.StatusInternalServerError)
		}
	})

	// -----==^.^==----- Get deleted products ------==^.^==------
	http.HandleFunc("/deleted-products", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		deletedProductsFromDatabase, err := ListAllDeletedProducts(db)
		if err != nil {
			http.Error(w, "unable to list all deleted products", http.StatusInternalServerError)
			return
		}

		productsResponse := []ListProductResponse{}
		for _, pbc := range deletedProductsFromDatabase {
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
