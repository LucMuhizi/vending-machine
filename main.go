package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Product represents a product in the vending machine
type Product struct {
	ID              string `json:"id"`
	AmountAvailable int    `json:"amountAvailable"`
	Cost            int    `json:"cost"`
	ProductName     string `json:"productName"`
	SellerID        string `json:"sellerId"`
}

// User represents a user of the vending machine
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Deposit  int    `json:"deposit"`
	Role     string `json:"role"`
}

var products []Product
var users []User

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/users", createUser).Methods("POST")
	r.HandleFunc("/users/{id}", getUser).Methods("GET")
	r.HandleFunc("/users/{id}", updateUser).Methods("PUT")
	r.HandleFunc("/users/{id}", deleteUser).Methods("DELETE")

	r.HandleFunc("/products", getProducts).Methods("GET")
	r.HandleFunc("/products", createProduct).Methods("POST")
	r.HandleFunc("/products/{id}", updateProduct).Methods("PUT")
	r.HandleFunc("/products/{id}", deleteProduct).Methods("DELETE")

	r.HandleFunc("/deposit", deposit).Methods("POST")
	r.HandleFunc("/buy", buy).Methods("POST")
	r.HandleFunc("/reset", reset).Methods("POST")

	log.Fatal(http.ListenAndServe(":8080", r))
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var user User
	json.NewDecoder(r.Body).Decode(&user)
	users = append(users, user)
	json.NewEncoder(w).Encode(user)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for _, user := range users {
		if user.ID == params["id"] {
			json.NewEncoder(w).Encode(user)
			return
		}
	}
	json.NewEncoder(w).Encode(&User{})
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for index, user := range users {
		if user.ID == params["id"] {
			users = append(users[:index], users[index+1:]...)
			var updatedUser User
			json.NewDecoder(r.Body).Decode(&updatedUser)
			users = append(users, updatedUser)
			json.NewEncoder(w).Encode(updatedUser)
			return
		}
	}
	json.NewEncoder(w).Encode(users)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for index, user := range users {
		if user.ID == params["id"] {
			users = append(users[:index], users[index+1:]...)
			break
		}
	}
	json.NewEncoder(w).Encode(users)
}

func getProducts(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(products)
}

func createProduct(w http.ResponseWriter, r *http.Request) {
	var product Product
	json.NewDecoder(r.Body).Decode(&product)

	sellerID := r.Header.Get("X-User-ID")
	if sellerID == "" {
		http.Error(w, "Missing seller ID", http.StatusUnauthorized)
		return
	}
	product.SellerID = sellerID

	products = append(products, product)
	json.NewEncoder(w).Encode(product)
}

func updateProduct(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	sellerID := r.Header.Get("X-User-ID")

	for index, product := range products {
		if product.ID == params["id"] {
			if product.SellerID != sellerID {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			products = append(products[:index], products[index+1:]...)
			var updatedProduct Product
			json.NewDecoder(r.Body).Decode(&updatedProduct)
			updatedProduct.SellerID = sellerID
			products = append(products, updatedProduct)
			json.NewEncoder(w).Encode(updatedProduct)
			return
		}
	}

	http.Error(w, "Product not found", http.StatusNotFound)
}

func deleteProduct(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	sellerID := r.Header.Get("X-User-ID")

	for index, product := range products {
		if product.ID == params["id"] {
			if product.SellerID != sellerID {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			products = append(products[:index], products[index+1:]...)
			json.NewEncoder(w).Encode(products)
			return
		}
	}

	http.Error(w, "Product not found", http.StatusNotFound)
}

func deposit(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Missing user ID", http.StatusUnauthorized)
		return
	}

	var user *User
	for i := range users {
		if users[i].ID == userID {
			user = &users[i]
			break
		}
	}

	if user == nil || user.Role != "buyer" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	amount, err := strconv.Atoi(r.FormValue("amount"))
	if err != nil || (amount != 5 && amount != 10 && amount != 20 &&
		amount != 50 && amount != 100) {
		http.Error(w, "Invalid deposit amount", http.StatusBadRequest)
		return
	}

	user.Deposit += amount
	json.NewEncoder(w).Encode(user)
}

func buy(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Missing user ID", http.StatusUnauthorized)
		return
	}

	var user *User
	for i := range users {
		if users[i].ID == userID {
			user = &users[i]
			break
		}
	}

	if user == nil || user.Role != "buyer" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	productID := r.FormValue("productId")
	amount, err := strconv.Atoi(r.FormValue("amount"))
	if err != nil {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	var product *Product
	for i := range products {
		if products[i].ID == productID {
			product = &products[i]
			break
		}
	}

	if product == nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	if amount > product.AmountAvailable {
		http.Error(w, "Insufficient product quantity", http.StatusBadRequest)
		return
	}

	totalCost := product.Cost * amount
	if totalCost > user.Deposit {
		http.Error(w, "Insufficient funds", http.StatusPaymentRequired)
		return
	}

	product.AmountAvailable -= amount
	user.Deposit -= totalCost

	change, err := calculateChange(user.Deposit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user.Deposit = 0

	resp := struct {
		TotalSpent     int       `json:"totalSpent"`
		ProductsBought []Product `json:"productsBought"`
		Change         []int     `json:"change"`
	}{
		TotalSpent:     totalCost,
		ProductsBought: []Product{*product},
		Change:         change,
	}

	json.NewEncoder(w).Encode(resp)
}

func calculateChange(amount int) ([]int, error) {
	coins := []int{100, 50, 20, 10, 5}

	var change []int
	for _, coin := range coins {
		for amount >= coin {
			change = append(change, coin)
			amount -= coin
		}
	}

	if amount != 0 {
		return nil, errors.New("unable to make change")
	}

	return change, nil
}

func reset(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Missing user ID", http.StatusUnauthorized)
		return
	}

	for i := range users {
		if users[i].ID == userID {
			users[i].Deposit = 0
			json.NewEncoder(w).Encode(users[i])
			return
		}
	}

	http.Error(w, "User not found", http.StatusNotFound)
}
