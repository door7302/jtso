$(document).ready(function () {
  $('#ListRtrs').DataTable({
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
          { orderable: false, targets: 5 } // Disable sorting on the "Delete" column
      ]
  });
});

function addRouter() {
  var h = document.getElementById("Hostname").value.trim();
  var s = document.getElementById("Shortname").value.trim();


  var dataToSend = {
    "hostname": h,
    "shortname": s
  };
  waitingDialog.show();
  // send data
  $(function () {
    $.ajax({
      type: 'POST',
      url: "/addrouter",
      data: JSON.stringify(dataToSend),
      contentType: "application/json",
      dataType: "json",
      success: function (json) {
        if (json.status == "OK") {
          var row = $("<tr><td>" + s + "</td><td>" + h + "</td><td>" + json.family + "</td><td>" + json.model + "</td><td>" + json.version + '</td><td class="d-xxl-flex justify-content-xxl-center"><button onclick="remove("' + s + '", this)" class="btn btn-danger" style="margin-left: 5px;" type="submit"><i class="fa fa-trash" style="font-size: 15px;"></i></button></td></tr>')
          $("#ListRtrs").append(row);
          document.getElementById("Hostname").value = "";
          document.getElementById("Shortname").value = "";
          waitingDialog.hide();
          alertify.success("Router " + s + " has been successfulfy added");
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

function remove(name, td) {
  var dataToSend = {
    "shortname": name
  };
  alertify.confirm("Are you sure you want to remove the router? All data will be lost.", function (e) {
    if (e) {
      // send data
      waitingDialog.show();
      $(function () {
        $.ajax({
          type: 'POST',
          url: "/delrouter",
          data: JSON.stringify(dataToSend),
          contentType: "application/json",
          dataType: "json",
          success: function (json) {
            if (json.status == "OK") {
              $(td).closest("tr").remove();
              waitingDialog.hide();
              alertify.success("Router " + name + " has been successfulfy removed")
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
  }).setHeader('JSTO...');
}

function importCSV() {
  alertify.alert("JSTO...", "Feature not supported yet.");
}