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
      { orderable: false, targets: 4 }
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
  if (!fileInput.files || fileInput.files.length === 0) {
    alertify.alert("JTT...", "Please select a CSV test file");
    return;
  }

  var file = fileInput.files[0];
  var reader = new FileReader();
  reader.onload = function (e) {
    var content = e.target.result;
    // Split into lines and filter out empty lines
    var lines = content.split(/\r?\n/).filter(function (l) { return l.trim() !== ''; });
    if (lines.length === 0) {
      alertify.alert("JTT...", "CSV file is empty");
      return;
    }

    var dataToSend = {
      name: testName,
      csv_lines: lines
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
          // Add new rows to table for each job
          var table = $('#jttJobsTable').DataTable();
          for (var j = 0; j < json.jobs.length; j++) {
            var job = json.jobs[j];
            var badge = stateBadge[job.status] || stateBadge["QUEUED"];
            var actions;
            if (job.status === "COMPLETED" || job.status === "FAILED" || job.status === "CANCELED") {
              actions = buildFinishedActions(job.job_id, job.name);
            } else {
              actions = buildActiveActions(job.job_id, job.name);
            }
            table.row.add([
              job.date,
              job.name,
              job.job_id,
              badge,
              actions
            ]).draw(false);
          }

          // Reset form
          $('#jttTestName').val('');
          fileInput.value = '';
          waitingDialog.hide();
          alertify.success("Test '" + testName + "' has been successfully submitted (" + json.jobs.length + " job(s))");
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
            row.find("td").eq(3).html(stateBadge["CANCELED"]);
            row.find("td").eq(4).html(buildFinishedActions(jobId, name));
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
        var currentState = row.find("td").eq(3).text().trim();
        var newState = json.state;
        if (newState !== currentState) {
          row.find("td").eq(3).html(stateBadge[newState]);
          // Update actions if state moved to a finished state
          if (newState === "COMPLETED" || newState === "FAILED" || newState === "CANCELED") {
            row.find("td").eq(4).html(buildFinishedActions(jobId, name));
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
