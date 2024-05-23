package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func setupRouter() *mux.Router {
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

	return r
}

func createTestUser(username, password, role string) User {
	user := User{
		ID:       uuid.NewString(),
		Username: username,
		Password: password,
		Role:     role,
	}
	users = append(users, user)
	return user
}

func createTestProduct(sellerID, productName string, cost, amountAvailable int) Product {
	product := Product{
		ID:              uuid.NewString(),
		ProductName:     productName,
		Cost:            cost,
		AmountAvailable: amountAvailable,
		SellerID:        sellerID,
	}
	products = append(products, product)
	return product
}

func TestCreateUser(t *testing.T) {
	router := setupRouter()

	payload := []byte(`{"id":"1","username":"testuser","password":"testpass","role":"buyer"}`)
	req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(payload))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var user User
	err := json.Unmarshal(resp.Body.Bytes(), &user)
	if err != nil || user.Username != "testuser" {
		t.Errorf("handler returned unexpected body: got %v want %v", resp.Body.String(), `{"id":"1","username":"testuser","password":"testpass","role":"buyer"}`)
	}
}

func TestGetUser(t *testing.T) {
	router := setupRouter()
	user := createTestUser("testuser", "testpass", "buyer")

	req, _ := http.NewRequest("GET", "/users/"+user.ID, nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var fetchedUser User
	err := json.Unmarshal(resp.Body.Bytes(), &fetchedUser)
	if err != nil || fetchedUser.ID != user.ID {
		t.Errorf("handler returned unexpected body: got %v want %v", resp.Body.String(), `{"id":"`+user.ID+`","username":"testuser","password":"testpass","role":"buyer"}`)
	}
}

func TestCreateProduct(t *testing.T) {
	router := setupRouter()
	seller := createTestUser("selleruser", "sellerpass", "seller")

	payload := []byte(`{"productName":"Soda","cost":100,"amountAvailable":10}`)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", seller.ID)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var product Product
	err := json.Unmarshal(resp.Body.Bytes(), &product)
	if err != nil || product.ProductName != "Soda" {
		t.Errorf("handler returned unexpected body: got %v want %v", resp.Body.String(), `{"productName":"Soda","cost":100,"amountAvailable":10}`)
	}
}

func TestDeposit(t *testing.T) {
	router := setupRouter()
	buyer := createTestUser("testbuyer", "testpass", "buyer")

	payload := []byte(`{"amount":100}`)
	req, _ := http.NewRequest("POST", "/deposit", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", buyer.ID)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var user User
	err := json.Unmarshal(resp.Body.Bytes(), &user)
	if err != nil || user.Deposit != 100 {
		t.Errorf("handler returned unexpected body: got %v want %v", resp.Body.String(), `{"id":"`+buyer.ID+`","username":"testbuyer","password":"testpass","role":"buyer","deposit":100}`)
	}
}

func TestBuy(t *testing.T) {
	router := setupRouter()
	seller := createTestUser("selleruser", "sellerpass", "seller")
	buyer := createTestUser("testbuyer", "testpass", "buyer")

	// Create a product
	product := createTestProduct(seller.ID, "Soda", 50, 10)

	// Deposit money
	payload := []byte(`{"amount":100}`)
	req, _ := http.NewRequest("POST", "/deposit", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", buyer.ID)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Buy product
	payload = []byte(`{"productId":"` + product.ID + `","amount":1}`)
	req, _ = http.NewRequest("POST", "/buy", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", buyer.ID)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response struct {
		TotalSpent     int       `json:"totalSpent"`
		ProductsBought []Product `json:"productsBought"`
		Change         []int     `json:"change"`
	}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	if err != nil || response.TotalSpent != 50 {
		t.Errorf("handler returned unexpected body: got %v want %v", resp.Body.String(), `{"totalSpent":50}`)
	}
}

func TestReset(t *testing.T) {
	router := setupRouter()
	buyer := createTestUser("testbuyer", "testpass", "buyer")

	// Deposit money
	payload := []byte(`{"amount":100}`)
	req, _ := http.NewRequest("POST", "/deposit", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", buyer.ID)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Reset deposit
	req, _ = http.NewRequest("POST", "/reset", nil)
	req.Header.Set("X-User-ID", buyer.ID)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var user User
	err := json.Unmarshal(resp.Body.Bytes(), &user)
	if err != nil || user.Deposit != 0 {
		t.Errorf("handler returned unexpected body: got %v want %v", resp.Body.String(), `{"id":"`+buyer.ID+`","username":"testbuyer","password":"testpass","role":"buyer","deposit":0}`)
	}
}
