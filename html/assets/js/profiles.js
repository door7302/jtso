$(document).ready(function () {
  $('#ListProfiles').DataTable({
    paging: false, // Enable pagination
    searching: true, // Enable filtering
    ordering: true, // Enable column sorting
    info: false, // Show table info (e.g., "Showing 1 to 10 of 50 entries")
    responsive: true, // Make the table responsive
    language: {
      search: "Filter:", // Customize the search box label
      lengthMenu: "Show _MENU_ entries",
    },
    columnDefs: [
      { orderable: false, targets: 2 } // Disable sorting on the "Action" column
    ]
  });
});

function showInfo() {
  alertify.alert("JSTO...", "CSV file must include these following fields with the ';' separator:</br></br>[shortName];[profileName1];[profileName2];...</br>");
}

function addAsso() {

  var r = document.getElementById("router").value.trim();
  var all_selected = document.getElementById('profiles').options
  var selected = [];
  var raw_selected = ""
  for (var option of all_selected) {
    if (option.selected) {
      selected.push(option.value);
      raw_selected = raw_selected + option.value + " ; "
    }
  }


  if (selected.length == 0) {
    alertify.alert("JSTO...", "Please select at least one Profile in the list.");
  } else {
    raw_selected = raw_selected.slice(0, -3);
    var dataToSend = {
      "shortname": r,
      "profiles": selected
    };
    waitingDialog.show();
    // send data
    $(function () {
      $.ajax({
        type: 'POST',
        url: "/addprofile",
        data: JSON.stringify(dataToSend),
        contentType: "application/json",
        dataType: "json",
        success: function (json) {
          if (json.status == "OK") {
            const table = $("#ListProfiles").DataTable();

            table.row.add([
              r,
              raw_selected,
              `
                <div class="d-xxl-flex justify-content-xxl-center">
                    <button class="btn btn-danger" onclick="removeAsso('${r}', this)">
                        <i class="fa fa-trash"></i>
                    </button>
                </div>
              `
            ]);
            table.draw(false);

            waitingDialog.hide();

            alertify.success("Profile(s) have been successfulfy added to router " + r)

          } else {
            waitingDialog.hide();
            alertify.alert("JSTO...", json.msg);

          }
        },
        error: function (xhr, ajaxOptions, thrownError) {
          waitingDialog.hide();
          alertify.alert("JSTO...", "Unexpected error");

        }
      });
    });
  }
}

function removeAsso(name, td) {
  var dataToSend = {
    "shortname": name
  };
  // send data
  $(function () {
    $.ajax({
      type: 'POST',
      url: "/delprofile",
      data: JSON.stringify(dataToSend),
      contentType: "application/json",
      dataType: "json",
      success: function (json) {
        if (json.status == "OK") {
          const table = $("#ListProfiles").DataTable();
          table.row($(td).closest("tr")).remove().draw(false);

          alertify.success("Router " + name + " has been successfulfy removed")
        } else {
          alertify.alert("JSTO...", json.msg);
        }
      },
      error: function (xhr, ajaxOptions, thrownError) {
        alertify.alert("JSTO...", "Unexpected error");
      }
    });
  });
}


function importCSV() {
  const fileInput = document.getElementById('fileInput');

  fileInput.click();

  fileInput.addEventListener('change', async (e) => {
    const file = e.target.files[0];
    if (!file) {
      alertify.alert("JSTO...", "No file selected.");
      return;
    }

    // Validate file extension
    const validExtensions = ['csv'];
    const fileExtension = file.name.split('.').pop().toLowerCase();
    if (!validExtensions.includes(fileExtension)) {
      alertify.alert("JSTO...", "Invalid file type. Please upload a CSV file.");
      return;
    }

    // Validate file content for non-binary data
    const arrayBuffer = await file.slice(0, 1024).arrayBuffer();
    const text = new TextDecoder().decode(new Uint8Array(arrayBuffer));
    if (/[\x00-\x08\x0E-\x1F]/.test(text)) {
      alertify.alert("JSTO...", "Binary files are not allowed")
      return;
    }

    const formData = new FormData();
    formData.append('csvFile', file);

    try {
      waitingDialog.show();
      const response = await fetch('/uploadprofilecsv', {
        method: 'POST',
        body: formData
      });

      const result = await response.json();

      waitingDialog.hide();
      alertify.confirm(result.msg, function (e) {
        window.location.reload();
      }).setHeader('JSTO...');

    } catch (error) {
      alertify.alert("JSTO...", "An error occurred while uploading the file: "+JSON.stringify(error))
      waitingDialog.hide();
    }
  });
}


async function loadConfig(fileName) {
  try {
    const response = await fetch(fileName);
    if (!response.ok) {
      throw new Error(`Failed to load ${fileName}: ${response.statusText}`);
    }

    const tomlContent = await response.text();

    // Add syntax highlighting
    const highlightedToml = Prism.highlight(tomlContent, Prism.languages.toml, 'toml');

    // Update modal content with highlighted TOML
    document.getElementById('modalcore').innerHTML = `<pre><code class="language-toml">${highlightedToml}</code></pre>`;

    // Show the modal
    const modal = new bootstrap.Modal(document.getElementById('config'));
    modal.show();
  } catch (error) {
    alertify.alert("JSTO...", "Error loading config:" + error);
    document.getElementById('modalcore').textContent = 'Error loading configuration.';
  }
}