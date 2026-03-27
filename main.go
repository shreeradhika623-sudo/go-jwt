package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var client *mongo.Client
var userCollection *mongo.Collection
var jwtSecret = []byte("mysecretkey")
var blacklist = make(map[string]bool)
var refreshStore = make(map[string]string) // refreshToken
func main() {

	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

	var err error
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	userCollection = client.Database("test").Collection("users")

	log.Println("MongoDB connected")

	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/profile", authMiddleware(profileHandler))
	http.HandleFunc("/admin", authMiddleware(adminHandler))
	http.HandleFunc("/premium", authMiddleware(premiumHandler))
	http.HandleFunc("/logout", logoutHandler)
        http.HandleFunc("/refresh", refreshHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	log.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}

// ===== REGISTER =====

func registerHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}

	filter := bson.M{"username": username}
	err := userCollection.FindOne(context.TODO(), filter).Err()

	if err == nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	user := bson.M{
		"username": username,
		"password": string(hash),
		"role":     "user",
		"plan":     "free", // default plan
	}

	userCollection.InsertOne(context.TODO(), user)

	w.Write([]byte(`{"status":"success","message":"User registered"}`))
}

// ===== LOGIN =====

func loginHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()

	username := r.FormValue("username")
	password := r.FormValue("password")

	filter := bson.M{"username": username}

	var result bson.M

	err := userCollection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	storedHash := result["password"].(string)

	role := "user"
	if r, ok := result["role"].(string); ok {
		role = r
	}

	plan := "free"
	if p, ok := result["plan"].(string); ok {
		plan = p
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

// ===== ACCESS TOKEN (short life) =====
accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
	"username": username,
	"role":     role,
	"plan":     plan,
	"exp":      time.Now().Add(15 * time.Minute).Unix(),
})

accessTokenString, _ := accessToken.SignedString(jwtSecret)

// ===== REFRESH TOKEN (long life) =====
refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
	"username": username,
	"exp":      time.Now().Add(24 * time.Hour).Unix(),
})

refreshTokenString, _ := refreshToken.SignedString(jwtSecret)

// store refresh token
refreshStore[refreshTokenString] = username

// response
w.Header().Set("Content-Type", "application/json")
w.Write([]byte(`{
	"status":"success",
	"access_token":"` + accessTokenString + `",
	"refresh_token":"` + refreshTokenString + `"
}`))

}

// ===== MIDDLEWARE =====

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		if blacklist[tokenString] {
			http.Error(w, "Token is logged out", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims := token.Claims.(jwt.MapClaims)

		r.Header.Set("X-User", claims["username"].(string))
		r.Header.Set("X-Role", claims["role"].(string))
		r.Header.Set("X-Plan", claims["plan"].(string))

		next(w, r)
	}
}

// ===== PROFILE =====

func profileHandler(w http.ResponseWriter, r *http.Request) {

	user := r.Header.Get("X-User")
	plan := r.Header.Get("X-Plan")

	w.Write([]byte(`{"status":"success","user":"` + user + `","plan":"` + plan + `"}`))
}

// ===== ADMIN =====

func adminHandler(w http.ResponseWriter, r *http.Request) {

	role := r.Header.Get("X-Role")

	if role != "admin" {
		http.Error(w, "Access denied: admin only", http.StatusForbidden)
		return
	}

	w.Write([]byte(`{"status":"success","message":"Welcome Admin"}`))
}

// ===== PREMIUM =====

func premiumHandler(w http.ResponseWriter, r *http.Request) {

	plan := r.Header.Get("X-Plan")

	if plan != "premium" {
		http.Error(w, "Upgrade to premium required", http.StatusForbidden)
		return
	}

	w.Write([]byte(`{"status":"success","message":"Welcome Premium User"}`))
}

// ===== LOGOUT =====

func logoutHandler(w http.ResponseWriter, r *http.Request) {
authHeader := r.Header.Get("Authorization")
tokenString := strings.TrimPrefix(authHeader, "Bearer ")

   // blacklist access token
       blacklist[tokenString] = true

   //  ALSO remove all refresh tokens of this user
       for rt, user := range refreshStore {
	if user == r.Header.Get("X-User") {
		delete(refreshStore, rt)
	}
 }

     w.Write([]byte("Logout successful"))

}

func refreshHandler(w http.ResponseWriter, r *http.Request) {

	tokenString := r.Header.Get("Authorization")

	if tokenString == "" {
		http.Error(w, "Refresh token required", http.StatusUnauthorized)
		return
	}

	// check if exists
	username, ok := refreshStore[tokenString]
	if !ok {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	// new access token
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(15 * time.Minute).Unix(),
	})

	newTokenString, _ := newToken.SignedString(jwtSecret)

	w.Write([]byte(`{"access_token":"` + newTokenString + `"}`))
}
