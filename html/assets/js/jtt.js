$(document).ready(function () {
  $('#jttJobsTable').DataTable({
    paging: false,
    searching: true,
    ordering: true,
    info: false,
    responsive: true,
    language: {
      search: "Filter:",
    },
    columnDefs: [
      { orderable: false, targets: 3 }
    ]
  });

  $('#routerlist').multiselect({
    includeSelectAllOption: true,
    enableFiltering: true,
    filterPlaceholder: 'Search...',
    buttonWidth: '100%'
  });
});

// Badge mapping
var stateBadge = {
  "QUEUED": '<span class="badge bg-secondary">QUEUED</span>',
  "IN-PROGRESS": '<span class="badge bg-info text-dark">IN-PROGRESS</span>',
  "COMPLETED": '<span class="badge bg-success">COMPLETED</span>',
  "FAILED": '<span class="badge bg-danger">FAILED</span>',
  "CANCELED": '<span class="badge bg-dark">CANCELED</span>'
};

// Action buttons builders
function buildActiveActions(jobId, name) {
  return '<button class="btn btn-sm btn-outline-info" title="Update Status" onclick="updateJob(\'' + jobId + '\',\'' + name.replace(/'/g, "\\'") + '\', this)">' +
    '<i class="fas fa-sync-alt"></i></button>' +
    '<button class="btn btn-sm btn-outline-warning ms-1" title="Cancel Job" onclick="cancelJob(\'' + jobId + '\',\'' + name.replace(/'/g, "\\'") + '\', this)">' +
    '<i class="fas fa-ban"></i></button>';
}

function buildFinishedActions(jobId, name) {
  return '<button class="btn btn-sm btn-outline-primary" title="Show Result" onclick="showDetail(\'' + jobId + '\',\'' + name.replace(/'/g, "\\'") + '\')">' +
    '<i class="fas fa-eye"></i></button>' +
    '<button class="btn btn-sm btn-outline-danger ms-1" title="Delete Job" onclick="removeJob(\'' + jobId + '\',\'' + name.replace(/'/g, "\\'") + '\', this)">' +
    '<i class="fas fa-trash"></i></button>';
}

// 1/ Launch Test
$('#launchTest').on('click', function () {
  var testName = $('#jttTestName').val().trim();
  var selected = $('#routerlist').val();
  var fileInput = document.getElementById('jttTestFile');

  // Validations
  if (testName === "") {
    alertify.alert("JTT...", "Test name cannot be empty");
    return;
  }
  if (testName.length > 64) {
    alertify.alert("JTT...", "Test name cannot exceed 64 characters");
    return;
  }
  if (!selected || selected.length === 0) {
    alertify.alert("JTT...", "Please select at least one router");
    return;
  }
  if (!fileInput.files || fileInput.files.length === 0) {
    alertify.alert("JTT...", "Please select a JSON test file");
    return;
  }

  var file = fileInput.files[0];
  var reader = new FileReader();
  reader.onload = function (e) {
    var content = e.target.result;
    var jsonData;
    try {
      jsonData = JSON.parse(content);
    } catch (err) {
      alertify.alert("JTT...", "Invalid JSON file: " + err.message);
      return;
    }

    // Build routers array
    var routers = [];
    for (var i = 0; i < selected.length; i++) {
      var parts = selected[i].split('#');
      routers.push({
        shortname: parts[0],
        hostname: parts[1],
        model: parts[2],
        family: parts[3]
      });
    }

    var dataToSend = {
      name: testName,
      routers: routers,
      test_data: jsonData
    };

    waitingDialog.show();
    $.ajax({
      type: 'POST',
      url: "/jttlaunch",
      data: JSON.stringify(dataToSend),
      contentType: "application/json",
      dataType: "json",
      success: function (json) {
        if (json.status == "OK") {
          // Add new row to table
          var table = $('#jttJobsTable').DataTable();
          table.row.add([
            testName,
            json.job_id,
            stateBadge["QUEUED"],
            buildActiveActions(json.job_id, testName)
          ]).draw(false);

          // Reset form
          $('#jttTestName').val('');
          $('#routerlist').multiselect('deselectAll', false);
          $('#routerlist').multiselect('updateButtonText');
          fileInput.value = '';
          waitingDialog.hide();
          alertify.success("Test '" + testName + "' has been successfully submitted");
        } else {
          waitingDialog.hide();
          alertify.alert("JTT...", json.msg);
        }
      },
      error: function () {
        waitingDialog.hide();
        alertify.alert("JTT...", "Unexpected error");
      }
    });
  };
  reader.readAsText(file);
});

// 2/ Cancel Job
function cancelJob(jobId, name, td) {
  alertify.confirm("Are you sure you want to cancel the job '" + name + "'?", function (e) {
    if (e) {
      var dataToSend = {
        job_id: jobId,
        name: name
      };
      waitingDialog.show();
      $.ajax({
        type: 'POST',
        url: "/jttcancel",
        data: JSON.stringify(dataToSend),
        contentType: "application/json",
        dataType: "json",
        success: function (json) {
          if (json.status == "OK") {
            var row = $(td).closest("tr");
            row.find("td").eq(2).html(stateBadge["CANCELED"]);
            row.find("td").eq(3).html(buildFinishedActions(jobId, name));
            waitingDialog.hide();
            alertify.success("Job '" + name + "' has been canceled");
          } else {
            waitingDialog.hide();
            alertify.alert("JTT...", json.msg);
          }
        },
        error: function () {
          waitingDialog.hide();
          alertify.alert("JTT...", "Unexpected error");
        }
      });
    }
  }).setHeader('JTT...');
}

// 3/ Update Job Status
function updateJob(jobId, name, td) {
  var dataToSend = {
    job_id: jobId,
    name: name
  };
  waitingDialog.show();
  $.ajax({
    type: 'POST',
    url: "/jttupdate",
    data: JSON.stringify(dataToSend),
    contentType: "application/json",
    dataType: "json",
    success: function (json) {
      if (json.status == "OK") {
        var row = $(td).closest("tr");
        var currentState = row.find("td").eq(2).text().trim();
        var newState = json.state;
        if (newState !== currentState) {
          row.find("td").eq(2).html(stateBadge[newState]);
          // Update actions if state moved to a finished state
          if (newState === "COMPLETED" || newState === "FAILED" || newState === "CANCELED") {
            row.find("td").eq(3).html(buildFinishedActions(jobId, name));
          }
        }
        waitingDialog.hide();
        alertify.success("Job '" + name + "' state: " + newState);
      } else {
        waitingDialog.hide();
        alertify.alert("JTT...", json.msg);
      }
    },
    error: function () {
      waitingDialog.hide();
      alertify.alert("JTT...", "Unexpected error");
    }
  });
}

// 4/ Remove Job
function removeJob(jobId, name, td) {
  alertify.confirm("Are you sure you want to delete the job '" + name + "'?", function (e) {
    if (e) {
      var dataToSend = {
        job_id: jobId,
        name: name
      };
      waitingDialog.show();
      $.ajax({
        type: 'POST',
        url: "/jttdelete",
        data: JSON.stringify(dataToSend),
        contentType: "application/json",
        dataType: "json",
        success: function (json) {
          if (json.status == "OK") {
            var table = $('#jttJobsTable').DataTable();
            table.row($(td).closest("tr")).remove().draw(false);
            waitingDialog.hide();
            alertify.success("Job '" + name + "' has been deleted");
          } else {
            waitingDialog.hide();
            alertify.alert("JTT...", json.msg);
          }
        },
        error: function () {
          waitingDialog.hide();
          alertify.alert("JTT...", "Unexpected error");
        }
      });
    }
  }).setHeader('JTT...');
}

// 5/ Show Detail
function showDetail(jobId, name) {
  var dataToSend = {
    job_id: jobId,
    name: name
  };
  waitingDialog.show();
  $.ajax({
    type: 'POST',
    url: "/jttdetail",
    data: JSON.stringify(dataToSend),
    contentType: "application/json",
    dataType: "json",
    success: function (json) {
      if (json.status == "OK") {
        $('#jttDetailModalLabel').text("Job Details - " + name);
        $('#jttDetailModalBody').html('<pre>' + JSON.stringify(json.data, null, 2) + '</pre>');
        waitingDialog.hide();
        var modal = new bootstrap.Modal(document.getElementById('jttDetailModal'));
        modal.show();
      } else {
        waitingDialog.hide();
        alertify.alert("JTT...", json.msg);
      }
    },
    error: function () {
      waitingDialog.hide();
      alertify.alert("JTT...", "Unexpected error");
    }
  });
}
