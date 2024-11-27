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
          { orderable: false, targets: 2} // Disable sorting on the "Action" column
      ]
  });
});

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
  raw_selected = raw_selected.slice(0, -3);

  if (selected.length == 0) {
    alertify.alert("JSTO...", "Please select at least one Profile in the list.");
  } else {
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
            const tableBody = $("#ListProfiles tbody");
            const newRow = `
                <tr>
                    <td>${r}</td>
                    <td>${raw_selected}</td>
                    <td class="d-xxl-flex justify-content-xxl-center">
                        <!-- Delete Button -->
                        <button class="btn btn-danger" onclick="removeAsso('${s}', this)">
                            <i class="fa fa-trash"></i>
                        </button>
                    </td>
                </tr>
            `;
            alertify.success("Profile(s) have been successfulfy added to router " + r)
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
          $(td).closest("tr").remove();
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
  alertify.alert("JSTO...", "Feature not supported yet.");
}