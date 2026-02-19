package main

import (
	"log"
	"net/http"
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
	// password := r.FormValue("password") // You can use this later

	// 4. Print it to the terminal (so you can see it works!)
	log.Printf("Login attempt from user: %s\n", username)

	// 5. Send a simple message back to the browser
	w.Write([]byte("Login request received! Welcome " + username))
}
