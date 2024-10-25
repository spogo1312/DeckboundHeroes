package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	_ "modernc.org/sqlite" // Import the SQLite driver
)

type Card struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Damage     int    `json:"damage"`
	ManaCost   int    `json:"manaCost"`
	HealthCost int    `json:"healthCost"`
	Type       string `json:"type"` // e.g., "spell" or "attack"
	Effect     string `json:"effect"`
}

type Enemy struct {
	Name             string `json:"name"`
	Health           int    `json:"health"`
	Strength         int    `json:"strength"`
	Dexterity        int    `json:"dexterity"`
	Intelligence     int    `json:"intelligence"`
	Armor            int    `json:"armor"`
	Weapon           string `json:"weapon"`
	Level            int    `json:"level"`
	ExperienceReward int    `json:"experienceReward"` // Reward given upon defeat
}

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
	Name      string `json:"name"`
	Class     string `json:"class"`
	Race      string `json:"race"`
	Level     int    `json:"level"`
	XP        int    `json:"xp"`
	Health    int    `json:"health"`
	MaxHealth int    `json:"maxHealth"`
	Mana      int    `json:"mana"`
	MaxMana   int    `json:"maxMana"`
	Gold      int    `json:"gold"`
	Armor     string `json:"armor"`
	Weapon    string `json:"weapon"`
	Stats     Stats  `json:"stats"`
}

var currentEnemy *Enemy
var player Character

var db *sql.DB

func main() {
	fs := http.FileServer(http.Dir("./client"))
	http.Handle("/", fs)

	// Add CORS middleware to handle the preflight requests
	http.HandleFunc("/create-character", withCORS(CreateCharacterHandler))
	http.HandleFunc("/character", withCORS(CharacterHandler))
	http.HandleFunc("/apply-stat-boost", withCORS(ApplyStatBoostHandler))
	http.HandleFunc("/randomize-card", withCORS(RandomizeCard))
	http.HandleFunc("/start-combat", withCORS(StartCombatHandler))
	http.HandleFunc("/use-card", withCORS(UseCardHandler))
	http.HandleFunc("/save-progress", withCORS(SaveProgressHandler))
	http.HandleFunc("/load-progress", withCORS(LoadProgressHandler))

	fmt.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Initialize the random seed
func init() {
	rand.NewSource(time.Now().UnixNano())
	// Open the database connection
	var err error
	db, err = sql.Open("sqlite", "./game.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create the players table if it doesn't exist
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS players (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL UNIQUE,
        data TEXT NOT NULL
    );
    `
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
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

func SavePlayerToDB(p Character) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	// Insert or replace the player data
	query := `
    INSERT INTO players (name, data) VALUES (?, ?)
    ON CONFLICT(name) DO UPDATE SET data=excluded.data;
    `
	_, err = db.Exec(query, p.Name, string(data))
	return err
}

func LoadPlayerFromDB(name string) (Character, error) {
	var p Character
	query := `SELECT data FROM players WHERE name = ?;`
	row := db.QueryRow(query, name)

	var data string
	err := row.Scan(&data)
	if err != nil {
		return p, err
	}

	err = json.Unmarshal([]byte(data), &p)
	return p, err
}

func SaveProgressHandler(w http.ResponseWriter, r *http.Request) {
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

	// Unmarshal the request body into a Character struct
	var receivedPlayer Character
	err = json.Unmarshal(body, &receivedPlayer)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Save the player data to the database
	err = SavePlayerToDB(receivedPlayer)
	if err != nil {
		http.Error(w, "Error saving progress", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Progress saved successfully"))
}

func LoadProgressHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// For simplicity, we'll accept the player name in the request body for POST method
	var requestData struct {
		Name string `json:"name"`
	}

	if r.Method == "POST" {
		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
	} else {
		// For GET requests, get the name from the query parameter
		requestData.Name = r.URL.Query().Get("name")
	}

	if requestData.Name == "" {
		http.Error(w, "Player name is required", http.StatusBadRequest)
		return
	}

	// Load the player data from the database
	loadedPlayer, err := LoadPlayerFromDB(requestData.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Player not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error loading progress", http.StatusInternalServerError)
		}
		return
	}

	// Update the global player variable
	player = loadedPlayer

	// Send the player data back to the client
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
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

	// Save the new character to the database
	err = SavePlayerToDB(player)
	if err != nil {
		http.Error(w, "Error saving new character", http.StatusInternalServerError)
		return
	}

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

	// Set initial health and mana values and their maximums
	character.Level = 1
	character.XP = 0
	character.Gold = 100
	character.Stats = stats
	character.MaxHealth = stats.Endurance * 10
	character.MaxMana = stats.Intelligence * 5
	character.Health = character.MaxHealth // Start at full health
	character.Mana = character.MaxMana     // Start at full mana

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
		// Check if Mana should be restored to new MaxMana only if it was already at MaxMana
		wasAtMaxMana := character.Mana == character.MaxMana
		character.MaxMana = character.Stats.Intelligence * 5 // Recalculate MaxMana
		if wasAtMaxMana {
			character.Mana = character.MaxMana // Restore to new max only if already at max
		}
	case "endurance":
		character.Stats.Endurance += boost
		// Check if Health should be restored to new MaxHealth only if it was already at MaxHealth
		wasAtMaxHealth := character.Health == character.MaxHealth
		character.MaxHealth = character.Stats.Endurance * 10 // Recalculate MaxHealth
		if wasAtMaxHealth {
			character.Health = character.MaxHealth // Restore to new max only if already at max
		}
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

	// Debug output for tracking max values and current stats
	fmt.Printf("Updated MaxHealth: %d, Health: %d, MaxMana: %d, Mana: %d\n", character.MaxHealth, character.Health, character.MaxMana, character.Mana)
}

func CharacterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

func CombatRound(player *Character, enemy *Enemy) map[string]interface{} {
	// Player attacks first
	playerAttack := player.Stats.Strength * 2
	enemy.Health -= playerAttack
	result := fmt.Sprintf("Player attacks %s for %d damage!", enemy.Name, playerAttack)

	// Check if enemy is defeated
	if enemy.Health <= 0 {
		result += fmt.Sprintf(" %s defeated! You gain %d XP.", enemy.Name, enemy.ExperienceReward)
		player.XP += enemy.ExperienceReward
		currentEnemy = nil // Reset for new encounter
		return map[string]interface{}{
			"result":     result,
			"playerHP":   player.Health,
			"enemyHP":    enemy.Health,
			"player":     player,
			"combatOver": true,
		}
	}

	// Enemy's turn
	enemyAttack := enemy.Strength * 2
	player.Health -= enemyAttack
	result += fmt.Sprintf(" %s attacks player for %d damage!", enemy.Name, enemyAttack)

	// Check if player is defeated
	if player.Health <= 0 {
		result += " Player defeated! Game over."
		return map[string]interface{}{
			"result":     result,
			"playerHP":   player.Health,
			"enemyHP":    enemy.Health,
			"player":     player,
			"combatOver": true,
		}
	}

	// Return ongoing combat status
	return map[string]interface{}{
		"result":     result,
		"playerHP":   player.Health,
		"enemyHP":    enemy.Health,
		"player":     player,
		"combatOver": false,
	}
}

func StartCombatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Initialize enemy only if there is no active enemy or the current enemy is defeated
	if currentEnemy == nil || currentEnemy.Health <= 0 {
		currentEnemy = &Enemy{
			Name:             "Goblin",
			Health:           30,
			Strength:         8,
			Dexterity:        6,
			Intelligence:     4,
			Armor:            2,
			Weapon:           "Rusty Knife",
			Level:            1,
			ExperienceReward: 50,
		}
	}

	// Decode action from the request
	var actionData struct {
		Action string `json:"action"`
	}
	err := json.NewDecoder(r.Body).Decode(&actionData)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Execute round based on action
	var response map[string]interface{}
	if actionData.Action == "attack" {
		response = CombatRound(&player, currentEnemy)
	} else if actionData.Action == "card" {
		// Placeholder for card-based attack logic
		// e.g., response = UseCardRound(&player, currentEnemy)
	} else if actionData.Action == "start" {
		currentEnemy = &Enemy{
			Name:             "Goblin",
			Health:           30,
			Strength:         8,
			Dexterity:        6,
			Intelligence:     4,
			Armor:            2,
			Weapon:           "Rusty Knife",
			Level:            1,
			ExperienceReward: 50,
		}
		response = map[string]interface{}{
			"result":     "fight started",
			"playerHP":   player.Health,
			"enemyHP":    currentEnemy.Health,
			"player":     player,
			"combatOver": false,
		}
	}

	// Send response to frontend
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func UseCardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var cardData struct {
		CardID int `json:"cardId"`
	}
	err := json.NewDecoder(r.Body).Decode(&cardData)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Example card retrieval (from deck or predefined card list)
	card := Card{
		ID:       cardData.CardID,
		Name:     "Fireball",
		Damage:   10,
		ManaCost: 5,
		Type:     "spell",
		Effect:   "Burns the enemy for extra damage",
	}

	// Verify player has enough mana to use the card
	if player.Mana < card.ManaCost {
		http.Error(w, "Not enough mana", http.StatusBadRequest)
		return
	}

	// Apply card effect
	player.Mana -= card.ManaCost
	currentEnemy.Health -= card.Damage

	// Return the result of the card action
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     fmt.Sprintf("Player uses %s on enemy for %d damage", card.Name, card.Damage),
		"enemyHealth": currentEnemy.Health,
		"playerMana":  player.Mana,
	})
}
