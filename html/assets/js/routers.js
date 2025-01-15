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
          { orderable: false, targets: 5 } // Disable sorting on the "Actions" column
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
          const table = $("#ListRtrs").DataTable();
          table.row.add([
              s,
              h,
              json.family,
              json.model,
              json.version,
              `
                  <button onclick="reset('${h}', '${s}', this)" class="btn btn-success" style="margin-left: 5px;" type="button">
                      <i class="fa fa-sync" style="font-size: 15px;"></i>
                  </button>
                  <button onclick="remove('${s}', this)" class="btn btn-danger" style="margin-left: 5px;" type="submit">
                      <i class="fa fa-trash" style="font-size: 15px;"></i>
                  </button>
              `
          ]).draw();
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

function reset(hname,sname,td){
  var dataToSend = {
    "hostname": hname,
    "shortname": sname
  };
  alertify.confirm("Do you want to refresh the router " + sname +"'s entry", function (e) {
    if (e) {
      // send data
      waitingDialog.show();
      $(function () {
        $.ajax({
          type: 'POST',
          url: "/resetrouter",
          data: JSON.stringify(dataToSend),
          contentType: "application/json",
          dataType: "json",
          success: function (json) {
            if (json.status == "OK") {
              const row = $(td).closest("tr");
              row.find("td").eq(2).text(json.family); // Update the third column (Family)
              row.find("td").eq(3).text(json.model); // Update the fourth column (Model)
              row.find("td").eq(4).text(json.version); // Update the fifth column (Version)
              waitingDialog.hide();
              alertify.success("Router " + sname + "'s entry has been successfulfy updated")

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

function showInfo() {
  alertify.alert("JSTO...", "CSV file must include these following fields with the ';' separator:</br></br>[shortName];[HostName]</br>");  
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
          const response = await fetch('/uploadrtrcsv', {
              method: 'POST',
              body: formData
          });

          const result = await response.json();
          
          waitingDialog.hide();

          alertify.confirm(result.msg, function (e) {
            if (e) {
              //  nothing to do
            }
          }).setHeader('JSTO...');

          window.location.reload(); 
    
      } catch (error) {
          alertify.alert("JSTO...", "An error occurred while uploading the file")
      }
  });
}