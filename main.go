package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "test")
		fmt.Fprintln(w, "test")
	})

	fmt.Println("Serveur démarré sur le port 3000...")
	http.ListenAndServe(":3000", nil)
}
