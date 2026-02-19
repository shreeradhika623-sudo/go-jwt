package main

import (
	"context"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Serve static files from the "static" directory
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Serve the CSS directory
	cssFs := http.FileServer(http.Dir("css"))
	http.Handle("/css/", http.StripPrefix("/css/", cssFs))

	// Login Handler
	// This tells the server: "When someone visits /login, run the loginHandler function"
	http.HandleFunc("/login", loginHandler)

	// Serve the index.html file at the root path
	// This catches everything else (like "/")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	log.Println("Server started on :8080")
	log.Println("Visit http://localhost:8080 to view the login page")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// loginHandler processes the login form submission
// It prints the username to the terminal and sends a success message to the browser
func loginHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Only allow POST requests (sending data)
	if r.Method != "POST" {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Parse the form data so we can read it
	// NOTE: We use http.StatusBadRequest (not https)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Could not parse form", http.StatusBadRequest)
		return
	}

	// 3. Get the username and password from the inputs
	username := r.FormValue("username")
	password := r.FormValue("password")

	// 4. Print it to the terminal
	log.Printf("Login attempt from user: %s\n", username)

	// MongoDB Connection URI
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Printf("DB Connection Error: %v", err)
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(context.TODO())

	// Define a struct to hold the user data
	var result struct {
		Username string `bson:"username"`
		Password string `bson:"password"`
	}

	// Get a handle for your collection
	collection := client.Database("test").Collection("users")

	// Create a filter to find the user
	filter := bson.M{"username": username, "password": password}

	// Find the user
	err = collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		// If err is present, user was not found or password incorrect
		http.Error(w, "Invalid Credentials", http.StatusUnauthorized)
		return
	}

	w.Write([]byte("Welcome, " + result.Username + "! Access Granted via MongoDB."))
}
