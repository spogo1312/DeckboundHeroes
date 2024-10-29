$(document).ready(function () {

    //#region Tooltip
    let tooltipTimeout;

    $('.info-icon').hover(function (e) {
        const $tooltip = $(this);
        clearTimeout(tooltipTimeout);

        tooltipTimeout = setTimeout(function () {
            showTooltip($tooltip, e);
        }, 300);
    }, function () {
        clearTimeout(tooltipTimeout);
        hideTooltip($(this));
    });

    function showTooltip($icon, event) {
        const tooltipText = $icon.attr('data-info');
        const $tooltip = $('<div class="custom-tooltip"></div>').text(tooltipText).appendTo('body');
        const iconOffset = $icon.offset();
        const iconWidth = $icon.outerWidth();
        const tooltipWidth = $tooltip.outerWidth();
        const tooltipHeight = $tooltip.outerHeight();

        let top = iconOffset.top - tooltipHeight / 2 + $icon.outerHeight() / 2;
        let left = iconOffset.left + iconWidth + 10;

        if (left + tooltipWidth > $(window).width()) {
            left = iconOffset.left - tooltipWidth - 10;
        }
        if (top + tooltipHeight > $(window).height()) {
            top = $(window).height() - tooltipHeight - 10;
        } else if (top < 0) {
            top = 10;
        }

        $tooltip.css({
            top: `${top}px`,
            left: `${left}px`,
            position: 'absolute',
            background: '#333',
            color: '#fff',
            padding: '5px',
            borderRadius: '5px',
            zIndex: 1000,
            whiteSpace: 'nowrap'
        }).fadeIn(200);
    }

    function hideTooltip($icon) {
        $('.custom-tooltip').remove();
    }
    //#endregion Tooltip

    //#region Character Creation
    let playerData = null;
    let selectedRace = null;
    let selectedClass = null;

    $('.raceb').click(function () {
        $('.raceb').removeClass('highlight');
        $(this).addClass('highlight');
        selectedRace = $(this).data('race');
    });

    $('.classb').click(function () {
        $('.classb').removeClass('highlight');
        $(this).addClass('highlight');
        selectedClass = $(this).data('class');
    });

    $('#create-character-form').submit(function (e) {
        e.preventDefault();
        let characterData = {
            name: $('#name').val(),
            class: selectedClass,
            race: selectedRace
        };

        if (!characterData.name || !characterData.class || !characterData.race) {
            alert("Please fill out all fields and select both a race and a class.");
            return;
        }

        $.ajax({
            url: "http://localhost:8080/create-character",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify(characterData),
            success: function (character) {
                if (character && character.stats) {
                    displayCharacterInfo(character);
                    $("#card-selection").show();
                    $("#character-creation").remove();
                    $("#character-overview").show();
                    $("#toggle-overview-btn").show();
                    $("#start-combat-btn").show();
                } else {
                    console.error("Character stats missing from server response.");
                }
            },
            error: function (xhr, status, error) {
                console.error("Error creating character:", status, error);
            }
        });
    });
    //#endregion Character Creation

    //#region Deck Building
    let playerDeck = [];

    function generateBuildDeck() {
        const cards = [
            { id: 1, name: "Fireball", damage: 10, manaCost: 5 },
            { id: 2, name: "Ice Shard", damage: 8, manaCost: 4 },
            { id: 3, name: "Healing Light", heal: 15, manaCost: 6 },
            { id: 4, name: "Shadow Strike", damage: 12, manaCost: 7 }
        ];

        $('#deck-builder').empty();
        cards.forEach(card => {
            const cardElement = $(`
                <div class="card" data-id="${card.id}">
                    <h3>${card.name}</h3>
                    <p>${card.damage ? `Damage: ${card.damage}` : `Heal: ${card.heal}`}</p>
                    <p>Mana Cost: ${card.manaCost}</p>
                    <button class="add-to-deck">Add to Deck</button>
                </div>
            `);

            $('#deck-builder').append(cardElement);
        });
    }

    $(document).on('click', '.add-to-deck', function () {
        const cardId = $(this).closest('.card').data('id');
        const cardName = $(this).siblings('h3').text();

        if (playerDeck.includes(cardId)) {
            playerDeck = playerDeck.filter(id => id !== cardId);
            $(this).text('Add to Deck');
            alert(`${cardName} removed from deck`);
        } else {
            playerDeck.push(cardId);
            $(this).text('Remove from Deck');
            alert(`${cardName} added to deck`);
        }
    });
    generateBuildDeck();
    //#endregion Deck Building

    //#region Combat
    let combatInProgress = false;

    function startCombat() {
        combatInProgress = true;
        $('#attack-btn, #select-card-btn, #use-card-btn').prop('disabled', false);
        executeCombatRound("start");
    }

    $('#start-combat-btn').click(function () {
        startCombat();
        $('#combat-controls').show();
    });

    // Show cards for selection when "Select Card" is clicked
    $('#select-card-btn').click(function () {
        $('#combat-hand').show();
        generateCombatHand();
    });

    function generateCombatHand() {
        $('#combat-cards').empty();
        playerDeck.forEach(cardId => {
            const card = getCardById(cardId);
            const cardElement = $(`
                <div class="card" data-id="${card.id}">
                    <h3>${card.name}</h3>
                    <p>${card.damage ? `Damage: ${card.damage}` : `Heal: ${card.heal}`}</p>
                    <p>Mana Cost: ${card.manaCost}</p>
                </div>
            `);

            $('#combat-cards').append(cardElement);
        });
    }

    $(document).on('click', '#combat-cards .card', function () {
        $('#combat-cards .card').removeClass('selected');
        $(this).addClass('selected');
    });

    $('#attack-btn').click(function () {
        executeCombatRound("attack");
    });

    $('#use-card-btn').click(function () {
        const selectedCardId = getSelectedCardId();
        if (selectedCardId) {
            executeCombatRound("castSpell", selectedCardId);
        } else {
            alert("Please select a card to cast a spell.");
        }
    });

    function executeCombatRound(action, cardId = null) {
        if (!combatInProgress) {
            alert("Combat has ended.");
            return;
        }

        const requestData = { action: action };
        if (cardId) {
            requestData.cardId = cardId;
        }

        $.ajax({
            url: "http://localhost:8080/start-combat",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify(requestData),
            success: function (response) {
                updateCombatLog(response.result);
                updateHPDisplay(response.playerHP, response.enemyHP);
                updateManaDisplay(response.playerMana);

                if (response.combatOver) {
                    combatInProgress = false;
                    $('#attack-btn, #use-card-btn, #select-card-btn').prop('disabled', true);
                    $('#combat-hand').hide();
                    alert("Combat has ended!");
                }
            },
            error: function (xhr, status, error) {
                console.error("Error processing combat round:", status, error);
            }
        });
    }

    function getSelectedCardId() {
        return $('#combat-cards .card.selected').data('id');
    }

    function updateCombatLog(message) {
        const logEntry = $('<p></p>').text(message);
        $('#combat-log').append(logEntry);
    }

    function updateHPDisplay(playerHP, enemyHP) {
        $('#player-hp').text(playerHP);
        $('#enemy-hp').text(enemyHP);
    }

    function updateManaDisplay(playerMana) {
        $('#mana-display').text(playerMana);
    }

    function getCardById(id) {
        const cards = [
            { id: 1, name: "Fireball", damage: 10, manaCost: 5 },
            { id: 2, name: "Ice Shard", damage: 8, manaCost: 4 },
            { id: 3, name: "Healing Light", heal: 15, manaCost: 6 },
            { id: 4, name: "Shadow Strike", damage: 12, manaCost: 7 }
        ];
        return cards.find(card => card.id === id);
    }
    //#endregion Combat

    //#region Save/Load Progress
    $('#save-progress-btn').click(function () {
        if (playerData) {
            saveProgress(playerData);
        } else {
            alert("No player data available to save.");
        }
    });

    function saveProgress(playerData) {
        $.ajax({
            url: "http://localhost:8080/save-progress",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify(playerData),
            success: function (response) {
                alert("Progress saved successfully!");
                console.log(response);
            },
            error: function (xhr, status, error) {
                console.error("Error saving progress:", status, error);
                alert("Error saving progress.");
            }
        });
    }

    $('#load-progress-btn').click(function () {
        let playerName = prompt("Enter your character name to load progress:");
        if (playerName) {
            loadProgress(playerName);
        }
    });

    function loadProgress(playerName) {
        $.ajax({
            url: "http://localhost:8080/load-progress",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify({ name: playerName }),
            success: function (loadedPlayerData) {
                playerData = loadedPlayerData;
                $('#character-creation').remove();
                $('#character-overview').show();
                $('#toggle-overview-btn').show();
                $('#start-combat-btn').show();
                displayCharacterInfo(playerData);
                alert("Progress loaded successfully!");
            },
            error: function (xhr, status, error) {
                console.error("Error loading progress:", status, error);
                alert("Error loading progress.");
            }
        });
    }
    //#endregion Save/Load Progress

    //#region Character Info Display
    function displayCharacterInfo(character) {
        playerData = character;
        $('#character-name').text(`${character.name}`);
        $('#character-class-race').text(`${character.race} ${character.class}`);
    
        $('#stat-strength').text(character.stats.strength);
        $('#stat-dexterity').text(character.stats.dexterity);
        $('#stat-intelligence').text(character.stats.intelligence);
        $('#stat-endurance').text(character.stats.endurance);
        $('#stat-perception').text(character.stats.perception);
        $('#stat-wisdom').text(character.stats.wisdom);
        $('#stat-agility').text(character.stats.agility);
        $('#stat-luck').text(character.stats.luck);
    
        $('#level-display').text(character.level);
        $('#xp-display').text(character.xp);
    
        $('#health-display').text(`${character.health} / ${character.maxHealth}`);
        $('#mana-display').text(`${character.mana} / ${character.maxMana}`);
    
        $('#armor-display').text(character.armor);
        $('#weapon-display').text(character.weapon);

        $('#gold-display').text(character.gold);
    }
    //#endregion Character Info Display
});
