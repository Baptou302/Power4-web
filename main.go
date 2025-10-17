package main

import (
	"fmt"
	"log"
	"net/http"
	"power4/game/game"
)

func main() {
	addr := ":3000"

	game.ConnectDB() // 🔥 Initialise la base SQLite

	fmt.Println("✅ Serveur Power4 Web sur http://localhost" + addr)

	mux := http.NewServeMux()
	mux.HandleFunc("/", game.HandleIndex)
	mux.HandleFunc("/play", game.HandlePlay)
	mux.HandleFunc("/reset", game.HandleReset)
	mux.HandleFunc("/login", game.HandleLogin)
	mux.HandleFunc("/register", game.HandleRegister)

	mux.Handle("/style/", http.StripPrefix("/style/", http.FileServer(http.Dir("style"))))
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	log.Fatal(http.ListenAndServe(addr, mux))
}
