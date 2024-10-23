package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Character struct {
	Name   string `json:"name"`
	Class  string `json:"class"`
	Level  int    `json:"level"`
	XP     int    `json:"xp"`
	Health int    `json:"health"`
	Mana   int    `json:"mana"`
	Gold   int    `json:"gold"`
	Armor  string `json:"armor"`
	Weapon string `json:"weapon"`
}

var player Character

func main() {
	fs := http.FileServer(http.Dir("./client"))
	http.Handle("/", fs)

	// Add CORS middleware to handle the preflight requests
	http.HandleFunc("/create-character", withCORS(CreateCharacterHandler))
	http.HandleFunc("/character", withCORS(CharacterHandler))

	fmt.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// CORS middleware function to handle preflight requests
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Pass the request to the next handler
		next(w, r)
	}
}

// Modify the CreateCharacterHandler to accept input
func CreateCharacterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Can't read body", http.StatusBadRequest)
		return
	}

	// Unmarshal the request body into a character struct
	var newCharacter Character
	err = json.Unmarshal(body, &newCharacter)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if newCharacter.Class == "Warrior" {
		player = Character{
			Name:   newCharacter.Name,
			Class:  newCharacter.Class,
			Level:  1,
			XP:     0,
			Health: 150,
			Mana:   10,
			Gold:   100,
			Armor:  "Wooden Barrel Plate",
			Weapon: "Training Wooden Sword",
		}
	}
	if newCharacter.Class == "Mage" {
		player = Character{
			Name:   newCharacter.Name,
			Class:  newCharacter.Class,
			Level:  1,
			XP:     0,
			Health: 100,
			Mana:   50,
			Gold:   100,
			Armor:  "Old Teared Cloak",
			Weapon: "Stale Tree Branch",
		}

	}
	if newCharacter.Class == "Rogue" {
		player = Character{
			Name:   newCharacter.Name,
			Class:  newCharacter.Class,
			Level:  1,
			XP:     0,
			Health: 120,
			Mana:   10,
			Gold:   100,
			Armor:  "Faded Leather Jacket",
			Weapon: "Splintered Butter Knife",
		}

	}
	// Set default values and initialize the character

	// Send back the newly created character
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

func CharacterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}
