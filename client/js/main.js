$(document).ready(function () {
    let selectedCards = [];
    let currentStat = null;

    // Function to explain what each stat does when clicked
    function explainStat(statName) {
        let explanation = '';
        switch (statName) {
            case 'Strength':
                explanation = 'Strength increases your physical attack damage.';
                break;
            case 'Dexterity':
                explanation = 'Dexterity improves your accuracy and evasion.';
                break;
            case 'Intelligence':
                explanation = 'Intelligence increases your mana and spell power.';
                break;
            case 'Endurance':
                explanation = 'Endurance increases your health points (HP).';
                break;
            case 'Perception':
                explanation = 'Perception improves your ranged attack accuracy.';
                break;
            case 'Wisdom':
                explanation = 'Wisdom improves mana regeneration and spell casting.';
                break;
            case 'Agility':
                explanation = 'Agility increases your movement speed and reflexes.';
                break;
            case 'Luck':
                explanation = 'Luck increases your critical hit chance and loot quality.';
                break;
        }
        alert(explanation); // Show explanation
    }

    // Start the stat boost process when character is created
    $('#create-character-form').submit(function (e) {
        e.preventDefault();

        let characterData = {
            name: $('#name').val(),
            class: $('#class').val(),
            race: $('#race').val()
        };

        $.ajax({
            url: "http://localhost:8080/create-character",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify(characterData),
            success: function (character) {
                displayCharacterInfo(character);
                $('#card-selection').show(); // Show the card selection process
            },
            error: function (xhr, status, error) {
                console.error("Error creating character:", status, error);
            }
        });
    });

    // Handle stat click for explanation (before boosting)
    $('.stat').click(function () {
        const statName = $(this).data('stat');
        explainStat(statName);
    });

    // Stat boost process
    $('.statb').click(function () {
        $('.statb').removeClass('highlight'); // Remove highlight from all buttons
        $(this).addClass('highlight');        // Highlight the clicked button
        currentStat = $(this).data('statb');  // Store selected stat
        $('#step-indicator').text(`Choose a card to boost ${currentStat}`);
        $('#card-container').show();          // Show the card selection after selecting stat
    });
    

    // Handle card selection (boost stat based on selected card)
    $('.card').click(function () {
        if (!currentStat) {
            alert("Please select a stat first.");
            return;
        }

        const cardValue = $(this).data('value');
        selectedCards.push(cardValue);

        // Send card and stat to backend
        sendCardSelection(cardValue, currentStat, function() {
            highlightStatBoost(currentStat, cardValue); // Highlight the boosted stat
            if (selectedCards.length === 1) {
                $('#step-indicator').text("Choose Your Second Stat");
                currentStat = null; // Reset current stat for second round
                $('#card-container').hide();
            } else if (selectedCards.length === 2) {
                $('#card-selection').hide();
                $('#step-indicator').text("Stat boosts applied!");
            }
        });
    });

    // Send the selected card and stat to the backend to apply the boost
    function sendCardSelection(card, chosenStat, callback) {
        $.ajax({
            url: "http://localhost:8080/apply-stat-boost",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify({
                card: card,
                chosenStat: chosenStat
            }),
            success: function (updatedCharacter) {
                displayCharacterInfo(updatedCharacter);
                callback(); // Proceed to next step
            },
            error: function (xhr, status, error) {
                console.error("Error applying stat boost:", status, error);
            }
        });
    }

    // Highlight the stat that was boosted and update the value in the character overview
    function highlightStatBoost(statName, boostValue) {
        const statElement = $(`#stat-${statName.toLowerCase()}`);
        const originalValue = parseInt(statElement.text());
        const newValue = originalValue + boostValue;

        statElement.text(newValue); // Update stat value

        // Highlight the change for a few seconds
        statElement.addClass('boost-highlight');
        setTimeout(function () {
            statElement.removeClass('boost-highlight');
        }, 2000); // 2 seconds
    }

    // Display the character information in appropriate sections
    function displayCharacterInfo(character) {
        // Basic info
        $('#character-name').text(`${character.name}`);
        $('#character-class-race').text(`${character.race} ${character.class}`);

        // Stats
        $('#stat-strength').text(character.stats.strength);
        $('#stat-dexterity').text(character.stats.dexterity);
        $('#stat-intelligence').text(character.stats.intelligence);
        $('#stat-endurance').text(character.stats.endurance);
        $('#stat-perception').text(character.stats.perception);
        $('#stat-wisdom').text(character.stats.wisdom);
        $('#stat-agility').text(character.stats.agility);
        $('#stat-luck').text(character.stats.luck);

        // Level and XP
        $('#level-display').text(character.level);
        $('#xp-display').text(character.xp);

        // Health and Mana
        $('#health-display').text(character.health);
        $('#mana-display').text(character.mana);

        // Armor and Weapon
        $('#armor-display').text(character.armor);
        $('#weapon-display').text(character.weapon);

        // Gold in HUD
        $('#gold-display').text(character.gold);
    }
});
