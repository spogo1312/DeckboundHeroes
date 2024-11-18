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

type Effect struct {
	Type        string                 `json:"type"`        // The type of effect, e.g., "damage", "heal", "buff", etc.
	Target      string                 `json:"target"`      // Who the effect applies to, e.g., "self", "enemy", "allies"
	Parameters  map[string]interface{} `json:"parameters"`  // Additional parameters specific to the effect
	Description string                 `json:"description"` // Textual description of the effect
}

type Card struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	ManaCost int      `json:"manaCost"`
	Type     string   `json:"type"`    // e.g., "spell", "attack", "minion"
	Effects  []Effect `json:"effects"` // List of effects this card has
}

type Enemy struct {
	Entity
	Name             string `json:"name"`
	Health           int    `json:"health"`
	MaxHealth        int    `json:"maxHealth"`
	Strength         int    `json:"strength"`
	Dexterity        int    `json:"dexterity"`
	Intelligence     int    `json:"intelligence"`
	Armor            int    `json:"armor"`
	Weapon           string `json:"weapon"`
	Level            int    `json:"level"`
	ExperienceReward int    `json:"experienceReward"` // Reward given upon defeat

	ActiveDoTs   []DoT
	ActiveHoTs   []HoT
	ActiveBuffs  []Buff
	ActiveStatus []StatusEffect
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

type Entity struct {
	Name         string
	Health       int
	MaxHealth    int
	ActiveDoTs   []DoT
	ActiveHoTs   []HoT
	ActiveBuffs  []Buff
	ActiveStatus []StatusEffect
	// Add other necessary fields
}

type Character struct {
	Entity
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

	ActiveDoTs   []DoT
	ActiveHoTs   []HoT
	ActiveBuffs  []Buff
	ActiveStatus []StatusEffect
}

type DoT struct {
	Amount   int
	Duration int
}

type HoT struct {
	Amount   int
	Duration int
}

type Buff struct {
	Stat     string
	Modifier float64
	Duration int
}

type StatusEffect struct {
	EffectName string
	Chance     float64
	Duration   int
}

type Target interface {
	ApplyDoT(amount int, duration int)
	ApplyHoT(amount int, duration int)
	ApplyBuff(stat string, modifier float64, duration int)
	ApplyStatusEffect(effectName string, chance float64, duration int)
	ReceiveDamage(amount int) int
	GetName() string
	GetHealth() int
	SetHealth(int)
	GetMaxHealth() int
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

func (e Entity) ApplyDoT(amount int, duration int) {
	e.ActiveDoTs = append(e.ActiveDoTs, DoT{Amount: amount, Duration: duration})
	currentEnemy.ActiveDoTs = append(currentEnemy.ActiveDoTs, DoT{Amount: amount, Duration: duration})
}

func (e *Entity) ApplyHoT(amount int, duration int) {
	e.ActiveHoTs = append(e.ActiveHoTs, HoT{Amount: amount, Duration: duration})

}

func (e *Entity) ApplyBuff(stat string, modifier float64, duration int) {
	e.ActiveBuffs = append(e.ActiveBuffs, Buff{Stat: stat, Modifier: modifier, Duration: duration})
}

func (e *Entity) ApplyStatusEffect(effectName string, chance float64, duration int) {
	e.ActiveStatus = append(e.ActiveStatus, StatusEffect{EffectName: effectName, Chance: chance, Duration: duration})
}

func (e *Entity) ReceiveDamage(amount int) int {
	e.Health -= amount
	if e.Health < 0 {
		e.Health = 0
	}
	return amount
}
func (c *Character) ApplyDoT(amount int, duration int) {
	c.ActiveDoTs = append(c.ActiveDoTs, DoT{Amount: amount, Duration: duration})
}

func (c *Character) ApplyHoT(amount int, duration int) {
	c.ActiveHoTs = append(c.ActiveHoTs, HoT{Amount: amount, Duration: duration})
}

func (c *Character) ApplyBuff(stat string, modifier float64, duration int) {
	c.ActiveBuffs = append(c.ActiveBuffs, Buff{Stat: stat, Modifier: modifier, Duration: duration})
}

func (c *Character) ApplyStatusEffect(effectName string, chance float64, duration int) {
	c.ActiveStatus = append(c.ActiveStatus, StatusEffect{EffectName: effectName, Chance: chance, Duration: duration})
}

func (c *Character) ReceiveDamage(amount int) int {
	c.Health -= amount
	if c.Health < 0 {
		c.Health = 0
	}
	return amount
}

func (c *Character) GetName() string {
	return c.Name
}

func (c *Character) GetHealth() int {
	return c.Health
}

func (c *Character) SetHealth(h int) {
	c.Health = h
}

func (c *Character) GetMaxHealth() int {
	return c.MaxHealth
}

func (e *Enemy) ApplyDoT(amount int, duration int) {
	e.ActiveDoTs = append(e.ActiveDoTs, DoT{Amount: amount, Duration: duration})
}

func (e *Enemy) ApplyHoT(amount int, duration int) {
	e.ActiveHoTs = append(e.ActiveHoTs, HoT{Amount: amount, Duration: duration})
}

func (e *Enemy) ApplyBuff(stat string, modifier float64, duration int) {
	e.ActiveBuffs = append(e.ActiveBuffs, Buff{Stat: stat, Modifier: modifier, Duration: duration})
}

func (e *Enemy) ApplyStatusEffect(effectName string, chance float64, duration int) {
	e.ActiveStatus = append(e.ActiveStatus, StatusEffect{EffectName: effectName, Chance: chance, Duration: duration})
}

func (e *Enemy) ReceiveDamage(amount int) int {
	e.Health -= amount
	if e.Health < 0 {
		e.Health = 0
	}
	return amount
}

func (e *Enemy) GetName() string {
	return e.Name
}

func (e *Enemy) GetHealth() int {
	return e.Health
}

func (e *Enemy) SetHealth(h int) {
	e.Health = h
}

func (e *Enemy) GetMaxHealth() int {
	return e.MaxHealth
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

func CombatRound(player *Character, enemy *Enemy, action string, card *Card) map[string]interface{} {
	var result string
	var combatOver bool

	// Create Entity instances for player and enemy
	playerEntity := &Entity{
		Name:         player.Name,
		Health:       player.Health,
		MaxHealth:    player.MaxHealth,
		ActiveDoTs:   player.Entity.ActiveDoTs,
		ActiveHoTs:   player.ActiveHoTs,
		ActiveBuffs:  player.ActiveBuffs,
		ActiveStatus: player.ActiveStatus,
	}

	enemyEntity := &Entity{
		Name:         enemy.Name,
		Health:       enemy.Health,
		MaxHealth:    enemy.MaxHealth,
		ActiveDoTs:   enemy.ActiveDoTs,
		ActiveHoTs:   enemy.ActiveHoTs,
		ActiveBuffs:  enemy.ActiveBuffs,
		ActiveStatus: enemy.ActiveStatus,
	}

	// Process ongoing effects for the player
	playerResult := processOngoingEffects(playerEntity)
	if playerResult != "" {
		result += playerResult
	}

	// Update player's effects
	player.ActiveDoTs = playerEntity.ActiveDoTs
	player.ActiveHoTs = playerEntity.ActiveHoTs
	player.ActiveBuffs = playerEntity.ActiveBuffs
	player.ActiveStatus = playerEntity.ActiveStatus
	player.Health = playerEntity.Health

	// Check if the player is still alive after processing effects
	if player.Health <= 0 {
		result += " Player defeated by ongoing effects! Game over."
		combatOver = true
		return map[string]interface{}{
			"result":     result,
			"playerHP":   player.Health,
			"enemyHP":    enemy.Health,
			"playerMana": player.Mana,
			"combatOver": combatOver,
		}
	}

	// Process ongoing effects for the enemy
	enemyResult := processOngoingEffects(enemyEntity)
	if enemyResult != "" {
		result += enemyResult
	}

	// Update enemy's effects
	enemy.ActiveDoTs = enemyEntity.ActiveDoTs
	enemy.ActiveHoTs = enemyEntity.ActiveHoTs
	enemy.ActiveBuffs = enemyEntity.ActiveBuffs
	enemy.ActiveStatus = enemyEntity.ActiveStatus
	enemy.Health = enemyEntity.Health

	// Check if the enemy is still alive after processing effects
	if enemy.Health <= 0 {
		result += fmt.Sprintf(" %s defeated by ongoing effects! You gain %d XP.", enemy.Name, enemy.ExperienceReward)
		player.XP += enemy.ExperienceReward
		currentEnemy = nil // Reset for new encounter
		combatOver = true
		return map[string]interface{}{
			"result":     result,
			"playerHP":   player.Health,
			"enemyHP":    enemy.Health,
			"playerMana": player.Mana,
			"combatOver": combatOver,
		}
	}

	// Handle player's action
	if action == "attack" {
		// Basic Attack
		playerAttack := player.Stats.Strength * 2 // Example strength-based attack
		enemy.Health -= playerAttack
		result += fmt.Sprintf(" Player attacks %s for %d damage!", enemy.Name, playerAttack)
	} else if action == "castSpell" && card != nil {
		// Spell Casting
		if player.Mana >= card.ManaCost { // Check if player has enough mana
			player.Mana -= card.ManaCost
			// Apply the card's effects
			cardResult := applyCardEffects(card, player, enemy)
			result += cardResult
		} else {
			result += " Not enough mana to cast this spell."
		}
	}

	// Check if enemy is defeated after player's action
	if enemy.Health <= 0 {
		result += fmt.Sprintf(" %s defeated! You gain %d XP.", enemy.Name, enemy.ExperienceReward)
		player.XP += enemy.ExperienceReward
		currentEnemy = nil // Reset for new encounter
		combatOver = true
		return map[string]interface{}{
			"result":     result,
			"playerHP":   player.Health,
			"enemyHP":    enemy.Health,
			"playerMana": player.Mana,
			"combatOver": combatOver,
		}
	}

	// Enemy's turn to attack if still alive and combat is not over
	combatOver = player.Health <= 0 || enemy.Health <= 0
	if enemy.Health > 0 && !combatOver {
		// Wrap enemy in Entity for isStunned check
		if isStunned(enemyEntity) {
			result += fmt.Sprintf(" %s is stunned and cannot act!", enemy.Name)
		} else {
			enemyAttack := enemy.Strength * 2 // Basic enemy attack logic
			player.Health -= enemyAttack
			result += fmt.Sprintf(" %s attacks you for %d damage!", enemy.Name, enemyAttack)
		}
	}

	// Check if player is defeated after enemy's action
	if player.Health <= 0 {
		result += " Player defeated! Game over."
		combatOver = true
	}

	return map[string]interface{}{
		"result":     result,
		"playerHP":   player.Health,
		"enemyHP":    enemy.Health,
		"playerMana": player.Mana,
		"combatOver": combatOver,
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
			Health:           300,
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
		CardID int    `json:"cardId"` // Add CardID for spell casting if needed
	}
	err := json.NewDecoder(r.Body).Decode(&actionData)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Execute round based on action
	var response map[string]interface{}
	switch actionData.Action {
	case "attack":
		response = CombatRound(&player, currentEnemy, "attack", nil)
	case "castSpell":
		card := getCardByID(actionData.CardID)
		response = CombatRound(&player, currentEnemy, "castSpell", card)
	case "start":
		// Setup combat
		response = map[string]interface{}{
			"result": "fight started",
			// other relevant fields
		}
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	// Send response to frontend
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Example helper function to retrieve a card by ID (implement this based on your card management)
func getCardByID(cardID int) *Card {
	cards := []Card{
		{
			ID:       1,
			Name:     "Fireball",
			ManaCost: 5,
			Type:     "spell",
			Effects: []Effect{
				{
					Type:        "damage",
					Target:      "enemy",
					Parameters:  map[string]interface{}{"amount": 10},
					Description: "Deals 10 damage to the enemy.",
				},
				{
					Type:        "damageOverTime",
					Target:      "enemy",
					Parameters:  map[string]interface{}{"amount": 3, "duration": 3},
					Description: "Burns the enemy for 3 damage over 3 turns.",
				},
			},
		},
		{
			ID:       2,
			Name:     "Ice Shard",
			ManaCost: 4,
			Type:     "spell",
			Effects: []Effect{
				{
					Type:        "damage",
					Target:      "enemy",
					Parameters:  map[string]interface{}{"amount": 8},
					Description: "Deals 8 damage to the enemy.",
				},
				{
					Type:        "statusEffect",
					Target:      "enemy",
					Parameters:  map[string]interface{}{"effect": "freeze", "chance": 0.5, "duration": 3},
					Description: "50% chance to freeze the enemy, potentially skipping their turn for 3 rounds.",
				},
			},
		},
		{
			ID:       3,
			Name:     "Healing Light",
			ManaCost: 6,
			Type:     "spell",
			Effects: []Effect{
				{
					Type:        "heal",
					Target:      "self",
					Parameters:  map[string]interface{}{"amount": 20},
					Description: "Heals yourself for 20 health.",
				},
				{
					Type:        "healOverTime",
					Target:      "self",
					Parameters:  map[string]interface{}{"amount": 5, "duration": 2},
					Description: "Heals yourself for 5 health over 2 turns.",
				},
			},
		},
		{
			ID:       4,
			Name:     "Shadow Strike",
			ManaCost: 7,
			Type:     "attack",
			Effects: []Effect{
				{
					Type:        "damage",
					Target:      "enemy",
					Parameters:  map[string]interface{}{"amount": 12},
					Description: "Deals 12 damage to the enemy.",
				},
				{
					Type:        "buff",
					Target:      "self",
					Parameters:  map[string]interface{}{"stat": "attack", "modifier": 1.5, "duration": 2},
					Description: "Increases your attack by 50% for 2 turns.",
				},
			},
		},
	}

	// Search for the card with the matching ID
	for _, card := range cards {
		if card.ID == cardID {
			return &card
		}
	}

	// Return nil if no card with the specified ID is found
	return nil
}

func applyCardEffects(card *Card, player *Character, enemy *Enemy) string {
	var result string

	for _, effect := range card.Effects {
		var target Target

		switch effect.Target {
		case "self", "player":
			target = player
		case "enemy":
			target = enemy
		default:
			result += fmt.Sprintf("Invalid target '%s' for effect %s.", effect.Target, effect.Type)
			continue
		}

		switch effect.Type {
		case "damage":
			amount, ok := getFloatParameter(effect.Parameters, "amount")
			if !ok {
				result += " Invalid 'amount' parameter for damage effect."
				continue
			}
			target.ReceiveDamage(int(amount))
			result += fmt.Sprintf(" %s takes %d damage.", target.GetName(), int(amount))

		case "heal":
			amount, ok := getFloatParameter(effect.Parameters, "amount")
			if !ok {
				result += " Invalid 'amount' parameter for heal effect."
				continue
			}
			newHealth := target.GetHealth() + int(amount)
			if newHealth > target.GetMaxHealth() {
				newHealth = target.GetMaxHealth()
			}
			target.SetHealth(newHealth)
			result += fmt.Sprintf(" %s heals for %d health.", target.GetName(), int(amount))

		case "damageOverTime":
			amount, ok := getFloatParameter(effect.Parameters, "amount")
			duration, okDur := getFloatParameter(effect.Parameters, "duration")
			if !ok || !okDur {
				result += " Invalid parameters for damage over time effect."
				continue
			}
			target.ApplyDoT(int(amount), int(duration))
			result += fmt.Sprintf(" %s is afflicted with damage over time.", target.GetName())

		case "healOverTime":
			amount, ok := getFloatParameter(effect.Parameters, "amount")
			duration, okDur := getFloatParameter(effect.Parameters, "duration")
			if !ok || !okDur {
				result += " Invalid parameters for heal over time effect."
				continue
			}
			target.ApplyHoT(int(amount), int(duration))
			result += fmt.Sprintf(" %s will heal over time.", target.GetName())

		case "buff":
			stat, okStat := effect.Parameters["stat"].(string)
			modifier, okMod := getFloatParameter(effect.Parameters, "modifier")
			duration, okDur := getFloatParameter(effect.Parameters, "duration")
			if !okStat || !okMod || !okDur {
				result += " Invalid parameters for buff effect."
				continue
			}
			target.ApplyBuff(stat, modifier, int(duration))
			result += fmt.Sprintf(" %s's %s is increased.", target.GetName(), stat)

		case "statusEffect":
			effectName, okEffect := effect.Parameters["effect"].(string)
			chance, okChance := getFloatParameter(effect.Parameters, "chance")
			duration, okDur := getFloatParameter(effect.Parameters, "duration")
			if !okEffect || !okChance || !okDur {
				result += " Invalid parameters for status effect."
				continue
			}
			target.ApplyStatusEffect(effectName, chance, int(duration))
			result += fmt.Sprintf(" %s is affected by %s.", target.GetName(), effectName)

		case "lifeSteal":
			amount, ok := getFloatParameter(effect.Parameters, "amount")
			if !ok {
				result += " Invalid 'amount' parameter for lifeSteal effect."
				continue
			}
			damageDealt := enemy.ReceiveDamage(int(amount))
			playerHealth := player.GetHealth() + damageDealt
			if playerHealth > player.GetMaxHealth() {
				playerHealth = player.GetMaxHealth()
			}
			player.SetHealth(playerHealth)
			result += fmt.Sprintf(" %s steals %d health from %s.", player.GetName(), damageDealt, enemy.GetName())

		default:
			result += fmt.Sprintf(" Effect type %s not implemented.", effect.Type)
		}
	}

	return result
}

func getFloatParameter(parameters map[string]interface{}, key string) (float64, bool) {
	value, exists := parameters[key]
	if !exists {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	default:
		return 0, false
	}
}

func isStunned(entity *Entity) bool {
	for _, status := range entity.ActiveStatus {
		if status.EffectName == "stun" || status.EffectName == "freeze" {
			// Roll chance to determine if the effect takes place
			if rand.Float64() < status.Chance {
				return true
			}
		}
	}
	return false
}
func processOngoingEffects(entity *Entity) string {
	var result string

	// Process DoTs
	for i := 0; i < len(entity.ActiveDoTs); {
		dot := &entity.ActiveDoTs[i]
		entity.Health -= dot.Amount
		result += fmt.Sprintf(" %s takes %d damage.", entity.Name, dot.Amount)
		dot.Duration--
		if dot.Duration <= 0 {
			// Remove expired DoT
			entity.ActiveDoTs = append(entity.ActiveDoTs[:i], entity.ActiveDoTs[i+1:]...)
		} else {
			i++
		}
	}

	// Process HoTs
	for i := 0; i < len(entity.ActiveHoTs); {
		hot := &entity.ActiveHoTs[i]
		entity.Health += hot.Amount
		if entity.Health > entity.MaxHealth {
			entity.Health = entity.MaxHealth
		}
		result += fmt.Sprintf(" %s heals %d health.", entity.Name, hot.Amount)
		hot.Duration--
		if hot.Duration <= 0 {
			// Remove expired HoT
			entity.ActiveHoTs = append(entity.ActiveHoTs[:i], entity.ActiveHoTs[i+1:]...)
		} else {
			i++
		}
	}

	// Process Buffs (if needed)

	// Process Status Effects (e.g., reduce duration)
	for i := 0; i < len(entity.ActiveStatus); {
		status := &entity.ActiveStatus[i]
		status.Duration--
		if status.Duration <= 0 {
			// Remove expired status effect
			entity.ActiveStatus = append(entity.ActiveStatus[:i], entity.ActiveStatus[i+1:]...)
		} else {
			i++
		}
	}

	return result
}

func getTarget(targetType string, player *Character, enemy *Enemy) *Entity {
	switch targetType {
	case "self":
		return &Entity{
			Name:      player.Name,
			Health:    player.Health,
			MaxHealth: player.MaxHealth,
			// Add other fields if necessary
		}
	case "enemy":
		return &Entity{
			Name:      enemy.Name,
			Health:    enemy.Health,
			MaxHealth: enemy.MaxHealth,
			// Add other fields if necessary
		}
	// Implement other target types as needed
	default:
		return nil
	}
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

	// Retrieve the card by ID
	card := getCardByID(cardData.CardID)
	if card == nil {
		http.Error(w, "Card not found", http.StatusNotFound)
		return
	}

	// Verify player has enough mana
	if player.Mana < card.ManaCost {
		http.Error(w, "Not enough mana", http.StatusBadRequest)
		return
	}

	// Deduct mana cost
	player.Mana -= card.ManaCost

	// Apply card effects
	result := applyCardEffects(card, &player, currentEnemy)

	// Return the result of the card action
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     fmt.Sprintf("Player uses %s: %s", card.Name, result),
		"enemyHealth": currentEnemy.Health,
		"playerMana":  player.Mana,
	})
}
