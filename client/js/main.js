$(document).ready(function () {
    
    //#region Tooltip
    let tooltipTimeout;

    $('.info-icon').hover(function (e) {
        const $tooltip = $(this);
        
        // Clear any previous timeout to prevent multiple tooltips
        clearTimeout(tooltipTimeout);

        // Delay showing the tooltip by 300ms
        tooltipTimeout = setTimeout(function () {
            showTooltip($tooltip, e);
        }, 300);
    }, function () {
        // Clear timeout when the hover ends
        clearTimeout(tooltipTimeout);
        hideTooltip($(this));
    });

    // Function to show the tooltip
    function showTooltip($icon, event) {
        const tooltipText = $icon.attr('data-info');
        const $tooltip = $('<div class="custom-tooltip"></div>').text(tooltipText).appendTo('body');

        // Position the tooltip
        const iconOffset = $icon.offset();
        const iconWidth = $icon.outerWidth();
        const tooltipWidth = $tooltip.outerWidth();
        const tooltipHeight = $tooltip.outerHeight();

        let top = iconOffset.top - tooltipHeight / 2 + $icon.outerHeight() / 2;
        let left = iconOffset.left + iconWidth + 10;  // Add some space between the icon and tooltip

        // Ensure tooltip does not go off-screen (right side)
        if (left + tooltipWidth > $(window).width()) {
            left = iconOffset.left - tooltipWidth - 10;  // Place it on the left side of the icon
        }

        // Ensure tooltip does not go off-screen (top/bottom)
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
        }).fadeIn(200);  // Smooth fade-in effect
    }

    // Function to hide the tooltip
    function hideTooltip($icon) {
        $('.custom-tooltip').remove();
    }
    //#endregion

    let selectedCards = [];
    let currentStat = null;
    let selectedRace = null;
    let selectedClass = null;


    // Toggle character overview visibility
    $('#toggle-overview-btn').click(function () {
        $('#character-overview').toggle();  // Toggle the visibility of the character overview
        const isVisible = $('#character-overview').is(':visible');  // Check if it's currently visible
        $(this).text(isVisible ? 'Hide Character Overview' : 'Show Character Overview');  // Update button text
    });


    // Handle race selection
    $('.raceb').click(function () {
        $('.raceb').removeClass('highlight'); // Remove highlight from all race buttons
        $(this).addClass('highlight');        // Highlight the clicked race button
        selectedRace = $(this).data('race');
    });

    // Handle class selection
    $('.classb').click(function () {
        $('.classb').removeClass('highlight'); // Remove highlight from all class buttons
        $(this).addClass('highlight');         // Highlight the clicked class button
        selectedClass = $(this).data('class');
    });

    // Handle character creation form submission
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

        // Ensure data is sent and received correctly
        $.ajax({
            url: "http://localhost:8080/create-character",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify(characterData),
            success: function (character) {
                if (character && character.stats) {  // Ensure that stats are properly returned
                    displayCharacterInfo(character);
                    $('#card-selection').show();
                    $('#character-creation').remove();  // Remove character creation from DOM to prevent bugs
                    $('#character-overview').show();   
                    $('#toggle-overview-btn').show();
                } else {
                    console.error("Character stats missing from server response.");
                }
            },
            error: function (xhr, status, error) {
                console.error("Error creating character:", status, error);
            }
        });
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

        // Disable the card buttons to prevent multiple clicks
        $('.card').prop('disabled', true);

        // Get random card value from the backend
        $.ajax({
            url: "http://localhost:8080/randomize-card",
            type: "GET",
            contentType: "application/json",
            success: function (RandCardValue) {
                const cardValue = RandCardValue;
                selectedCards.push(cardValue);

                // Send card and stat to backend
                sendCardSelection(cardValue, currentStat, function() {
                    highlightStatBoost(currentStat, cardValue); // Highlight the boosted stat
                    // Remove highlight from stat after boost
                    $(`.statb[data-statb='${currentStat}']`).removeClass('highlight');
                    
                    // Re-enable the card buttons for the second round (if needed)
                    if (selectedCards.length === 1) {
                        $('#step-indicator').text("Choose Your Second Stat");
                        currentStat = null; // Reset current stat for second round
                        $('#card-container').hide();
                        $('.card').prop('disabled', false); // Re-enable cards
                    } else if (selectedCards.length === 2) {
                        $('#card-selection').hide();
                        $('#step-indicator').text("Stat boosts applied!");
                    }
                });
            },
            error: function (xhr, status, error) {
                console.error("Error getting the randomCardValue:", status, error);
                $('.card').prop('disabled', false); // Re-enable if error occurs
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

    function highlightStatBoost(statName, boostValue) {
        const statElement = $(`#stat-${statName.toLowerCase()}`);
        
        const currentStatValue = parseInt(statElement.text()); // Get current displayed value
    
        // Create a floating "+value" element
        const floatingValue = $(`<span class="floating-boost">+${boostValue}</span>`).appendTo('body');
    
        // Get the position of the stat element to position the floating value
        const statOffset = statElement.offset();
        const statWidth = statElement.outerWidth();
    
        // Position the floating value slightly above and to the right of the stat
        floatingValue.css({
            position: 'absolute',
            top: statOffset.top - 10, // Slightly above the stat
            left: statOffset.left + statWidth + 10, // Slightly to the right of the stat
            fontSize: '14px',
            color: 'green', // Use green color for the boost
            fontWeight: 'bold',
            zIndex: 1000
        });
    
        // Animate the floating value upwards and fade out
        floatingValue.animate({ top: '-=20', opacity: 0 }, 1500, function () {
            // Remove the floating value after the animation completes
            $(this).remove();
        });
    
        // Highlight the stat change for a few seconds
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
