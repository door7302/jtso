function updateDoc() {
  var p = document.getElementById("profiles").value.trim();
  var img = document.getElementById("profileImg");
  var desc = document.getElementById("profileDesc");
  var tele = document.getElementById("profileTele");
  var graf = document.getElementById("profileGraf");
  var kapa = document.getElementById("profileKapa");

  if (p == "default") {
    img.setAttribute('src', "img/default.png");
    desc.innerHTML = "N/A";
    tele.innerHTML = "N/A";
    graf.innerHTML = "N/A";
    kapa.innerHTML = "N/A";
  } else {
    var dataToSend = {
      "profile": p
    };
    waitingDialog.show();
    // send data
    $(function () {
      $.ajax({
        type: 'POST',
        url: "/updatedoc",
        data: JSON.stringify(dataToSend),
        contentType: "application/json",
        dataType: "json",
        success: function (json) {
          if (json.status == "OK") {
            img.setAttribute('src', "img/" + json.img);
            desc.innerHTML = json.desc.trim();
            tele.innerHTML = json.tele.trim();
            graf.innerHTML = json.graf.trim();
            kapa.innerHTML = json.kapa.trim();
            waitingDialog.hide();

          } else {
            alertify.alert("JSTO...", json.msg);
            waitingDialog.hide();
          }
        },
        error: function (xhr, ajaxOptions, thrownError) {
          alertify.alert("JSTO...", "Unexpected error");
          waitingDialog.hide();
        }
      });
    });
  }
}

async function loadConfig(fileName) {
  try {
    const response = await fetch(fileName);
    if (!response.ok) {
      throw new Error(`Failed to load ${fileName}: ${response.statusText}`);
    }

    const jsonContent = await response.json(); // Parse JSON

    // Pretty-print JSON
    const formattedJson = JSON.stringify(jsonContent, null, 2);

    // Add syntax highlighting
    const highlightedJson = Prism.highlight(formattedJson, Prism.languages.json, 'json');

    // Update modal content with highlighted JSON
    document.getElementById('modalcore').innerHTML = `<pre><code class="language-json">${highlightedJson}</code></pre>`;

    // Show the modal
    const modal = new bootstrap.Modal(document.getElementById('config'));
    modal.show();
  } catch (error) {
    alertify.alert("JSTO...", "Error loading config: " + error);
    document.getElementById('modalcore').textContent = 'Error loading configuration.';
  }
}


document.querySelector('#config .close').addEventListener('click', function () {
  const modal = document.getElementById('config');
  modal.classList.remove('show'); // Remove the `show` class
  modal.style.display = 'none';  // Hide the modal
  document.body.classList.remove('modal-open'); // Remove the modal-open class from body
  const backdrop = document.querySelector('.modal-backdrop');
  if (backdrop) backdrop.remove(); // Remove the backdrop element
});