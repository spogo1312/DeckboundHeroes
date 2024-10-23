$(document).ready(function () {
    // Handle character creation form submission
    $('#create-character-form').submit(function (e) {
        e.preventDefault();

        // Get the form data
        let characterData = {
            name: $('#name').val(),
            class: $('#class').val()
        };

        // Send the data to the backend using POST
        $.ajax({
            url: "http://localhost:8080/create-character",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify(characterData),
            success: function (character) {
                displayCharacterInfo(character);
            },
            error: function (xhr, status, error) {
                console.error("Error creating character:", status, error);
            }
        });
    });

    function displayCharacterInfo(character) {
        $('#character-info').html(`
            <h2>${character.name} the ${character.class}</h2>
            <p>Level: ${character.level}</p>
            <p>XP: ${character.xp}</p>
            <p>Health: ${character.health}</p>
            <p>Mana: ${character.mana}</p>
            <p>Gold: ${character.gold}</p>
            <p>Armor: ${character.armor}</p>
            <p>Weapon: ${character.weapon}</p>
        `);
    }
});
