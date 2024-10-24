package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type Stats struct {
	Strength     int `json:"strength"`
	Dexterity    int `json:"dexterity"`
	Intelligence int `json:"intelligence"`
	Endurance    int `json:"endurance"`
	Perception   int `json:"perception"`
	Wisdom       int `json:"wisdom"`
	Agility      int `json:"agility"`
	Luck         int `json:"luck"`
}

type Character struct {
	Name   string `json:"name"`
	Class  string `json:"class"`
	Race   string `json:"race"`
	Level  int    `json:"level"`
	XP     int    `json:"xp"`
	Health int    `json:"health"`
	Mana   int    `json:"mana"`
	Gold   int    `json:"gold"`
	Armor  string `json:"armor"`
	Weapon string `json:"weapon"`
	Stats  Stats  `json:"stats"`
}

var player Character

func main() {
	fs := http.FileServer(http.Dir("./client"))
	http.Handle("/", fs)

	// Add CORS middleware to handle the preflight requests
	http.HandleFunc("/create-character", withCORS(CreateCharacterHandler))
	http.HandleFunc("/character", withCORS(CharacterHandler))
	http.HandleFunc("/apply-stat-boost", withCORS(ApplyStatBoostHandler))
	http.HandleFunc("/randomize-card", withCORS(RandomizeCard))

	fmt.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Initialize the random seed
func init() {
	rand.NewSource(time.Now().UnixNano())
}

// CORS middleware
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

	// Assign stats based on race and class
	newCharacter = calculateStats(newCharacter)

	player = newCharacter

	// Set default values and initialize the character
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

func calculateStats(character Character) Character {
	var stats Stats

	// Base stats for each race, total points for each race = 80
	switch character.Race {
	case "Human":
		stats = Stats{Strength: 10, Dexterity: 10, Intelligence: 10, Endurance: 10, Perception: 10, Wisdom: 10, Agility: 10, Luck: 10}
	case "Elf":
		stats = Stats{Strength: 8, Dexterity: 14, Intelligence: 12, Endurance: 8, Perception: 10, Wisdom: 10, Agility: 14, Luck: 4}
	case "Dwarf":
		stats = Stats{Strength: 14, Dexterity: 8, Intelligence: 8, Endurance: 14, Perception: 8, Wisdom: 10, Agility: 6, Luck: 12}
	case "Orc":
		stats = Stats{Strength: 16, Dexterity: 8, Intelligence: 6, Endurance: 14, Perception: 8, Wisdom: 6, Agility: 10, Luck: 12}
	case "Gnome":
		stats = Stats{Strength: 6, Dexterity: 10, Intelligence: 14, Endurance: 8, Perception: 14, Wisdom: 10, Agility: 10, Luck: 8}
	}

	// Adjust stats and add base gear based on class
	switch character.Class {
	case "Warrior":
		stats.Strength += 5
		stats.Endurance += 3
		character.Armor = "Wooden Barrel Plate"
		character.Weapon = "Training Wooden Sword"
	case "Mage":
		stats.Intelligence += 5
		stats.Luck += 3
		character.Armor = "Old Teared Cloak"
		character.Weapon = "Stale Tree Branch"
	case "Rogue":
		stats.Dexterity += 5
		stats.Agility += 3
		character.Armor = "Faded Leather Jacket"
		character.Weapon = "Splintered Butter Knife"
	}

	// Set default values for Character
	character.Level = 1
	character.XP = 0
	character.Gold = 100
	character.Stats = stats
	character.Health = stats.Endurance * 10
	character.Mana = stats.Intelligence * 5

	// Debug to ensure stats are calculated correctly
	fmt.Printf("Stats calculated for character: %+v\n", character)

	return character
}

// Apply stat boost handler
func ApplyStatBoostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Can't read body", http.StatusBadRequest)
		return
	}

	// Parse the request body
	var boostData struct {
		Card       int    `json:"card"`
		ChosenStat string `json:"chosenStat"`
	}
	err = json.Unmarshal(body, &boostData)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Validate the card value (should be between 1 and 4)
	if boostData.Card < 1 || boostData.Card > 4 {
		http.Error(w, "Invalid card value", http.StatusBadRequest)
		return
	}

	// Apply the stat boost based on the selected card and stat
	applyStatBoost(&player, boostData.ChosenStat, boostData.Card)

	// Respond with the updated character data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

func RandomizeCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Randomize a card value between 1 and 4
	randomCardValue := rand.Intn(4) + 1

	// Send the randomized card value to the frontend
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(randomCardValue)
}

// Apply stat boost based on the selected stat and card value
func applyStatBoost(character *Character, statName string, boost int) {
	statName = strings.ToLower(statName) // Ensure case-insensitive comparison

	fmt.Printf("Applying a boost of %d to %s\n", boost, statName) // Debug line to track boosts

	switch statName {
	case "strength":
		character.Stats.Strength += boost
	case "dexterity":
		character.Stats.Dexterity += boost
	case "intelligence":
		character.Stats.Intelligence += boost
	case "endurance":
		character.Stats.Endurance += boost
	case "perception":
		character.Stats.Perception += boost
	case "wisdom":
		character.Stats.Wisdom += boost
	case "agility":
		character.Stats.Agility += boost
	case "luck":
		character.Stats.Luck += boost
	default:
		fmt.Printf("Invalid stat: %s\n", statName) // Debugging invalid stat names
	}
}

func CharacterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}
