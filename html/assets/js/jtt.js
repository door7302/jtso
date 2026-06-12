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

  // Add Refresh All button next to the filter field
  var refreshBtn = '<button id="refreshAllJobs" class="btn btn-sm btn-outline-info ms-2" title="Refresh All Active Jobs">' +
    '<i class="fas fa-sync-alt"></i> Refresh All</button>';
  $('#jttJobsTable_filter').append(refreshBtn);

  $('#refreshAllJobs').on('click', function () {
    refreshAllActiveJobs();
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

// Refresh All Active Jobs
function refreshAllActiveJobs() {
  var table = $('#jttJobsTable').DataTable();
  var activeRows = [];

  table.rows().every(function () {
    var data = this.data();
    var stateText = $(data[3]).text().trim();
    if (stateText === "QUEUED" || stateText === "IN-PROGRESS") {
      activeRows.push({ row: this, jobId: data[2], name: data[1] });
    }
  });

  if (activeRows.length === 0) {
    alertify.message("No active jobs to refresh");
    return;
  }

  waitingDialog.show();
  var pending = activeRows.length;
  var updated = 0;

  for (var i = 0; i < activeRows.length; i++) {
    (function (entry) {
      $.ajax({
        type: 'POST',
        url: "/jttupdate",
        data: JSON.stringify({ job_id: entry.jobId, name: entry.name }),
        contentType: "application/json",
        dataType: "json",
        success: function (json) {
          if (json.status == "OK") {
            var newState = json.state;
            var rowNode = $(entry.row.node());
            var currentState = rowNode.find("td").eq(3).text().trim();
            if (newState !== currentState) {
              rowNode.find("td").eq(3).html(stateBadge[newState]);
              if (newState === "COMPLETED" || newState === "FAILED" || newState === "CANCELED") {
                rowNode.find("td").eq(4).html(buildFinishedActions(entry.jobId, entry.name));
              }
              updated++;
            }
          }
          pending--;
          if (pending === 0) {
            waitingDialog.hide();
            alertify.success("Refreshed " + activeRows.length + " job(s), " + updated + " updated");
          }
        },
        error: function () {
          pending--;
          if (pending === 0) {
            waitingDialog.hide();
            alertify.success("Refreshed " + activeRows.length + " job(s), " + updated + " updated");
          }
        }
      });
    })(activeRows[i]);
  }
}

// 0/ File Validation
$('#validateFile').on('click', function () {
  var fileInput = document.getElementById('jttTestFile');

  if (!fileInput.files || fileInput.files.length === 0) {
    alertify.alert("JTT...", "Please select a CSV test file");
    return;
  }

  var file = fileInput.files[0];
  var reader = new FileReader();
  reader.onload = function (e) {
    var content = e.target.result;
    var lines = content.split(/\r?\n/).filter(function (l) { return l.trim() !== ''; });
    if (lines.length === 0) {
      alertify.alert("JTT...", "CSV file is empty");
      return;
    }

    var csvErrors = validateCSV(lines);
    if (csvErrors.length > 0) {
      // Populate the errors modal table
      var tbody = $('#csvErrorsTableBody');
      tbody.empty();
      for (var i = 0; i < csvErrors.length; i++) {
        tbody.append('<tr><td>' + csvErrors[i].line + '</td><td>' + csvErrors[i].message + '</td></tr>');
      }
      var modal = new bootstrap.Modal(document.getElementById('csvErrorsModal'));
      modal.show();
      // Keep Launch button disabled
      $('#launchTest').prop('disabled', true);
    } else {
      alertify.success("File is valid");
      $('#launchTest').prop('disabled', false);
    }
  };
  reader.readAsText(file);
});

// Disable Launch button when file input changes (force re-validation)
$('#jttTestFile').on('change', function () {
  $('#launchTest').prop('disabled', true);
});

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

  var selectedRouters = $('#routerlist').val();
  if (!selectedRouters || selectedRouters.length === 0) {
    alertify.alert("JTT...", "Please select at least one router");
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

    var routers = [];
    for (var r = 0; r < selectedRouters.length; r++) {
      var parts = selectedRouters[r].split("#");
      routers.push({
        shortname: parts[0] || "",
        hostname: parts[1] || "",
        model: parts[2] || "",
        family: parts[3] || "",
        version: parts[4] || ""
      });
    }

    var dataToSend = {
      name: testName,
      csv_lines: lines,
      routers: routers
    };

    waitingDialog.show();
    $.ajax({
      type: 'POST',
      url: "/jttlaunch",
      data: JSON.stringify(dataToSend),
      contentType: "application/json",
      dataType: "json",
      success: function (json) {
        var table = $('#jttJobsTable').DataTable();
        var jobErrors = [];

        if (json.status == "OK" || json.status == "NOK") {
          // Process all jobs
          if (json.jobs && json.jobs.length > 0) {
            for (var j = 0; j < json.jobs.length; j++) {
              var job = json.jobs[j];

              // Collect errors
              if (job.error && job.error !== "") {
                jobErrors.push({ name: job.name, job_id: job.job_id, error: job.error });
              }

              // Add to table only if status is not WATCHDOG
              if (job.status !== "WATCHDOG") {
                // Remove existing row with same job_id if present
                table.rows().every(function () {
                  var data = this.data();
                  if (data[2] === job.job_id) {
                    this.remove();
                  }
                });

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
            }
          }

          // Reset form
          $('#jttTestName').val('');
          fileInput.value = '';
          waitingDialog.hide();

          // Show errors modal or success
          if (jobErrors.length > 0) {
            var tbody = $('#jobErrorsTableBody');
            tbody.empty();
            for (var e = 0; e < jobErrors.length; e++) {
              tbody.append('<tr><td>' + jobErrors[e].name + '</td><td>' + jobErrors[e].job_id + '</td><td>' + jobErrors[e].error + '</td></tr>');
            }
            var modal = new bootstrap.Modal(document.getElementById('jobErrorsModal'));
            modal.show();
          } else if (json.status == "OK") {
            alertify.success("Test '" + testName + "' has been successfully submitted (" + (json.jobs ? json.jobs.length : 0) + " job(s))");
          }
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
        $('#jttDetailModalBody').html(buildDetailView(json.data));
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

function buildDetailView(data) {
  var html = '';

  // Global summary card
  var statusBadge = stateBadge[data.status] || '<span class="badge bg-secondary">' + data.status + '</span>';
  html += '<div class="card mb-3">';
  html += '<div class="card-header"><strong><i class="fas fa-info-circle me-2"></i>Summary</strong></div>';
  html += '<div class="card-body">';
  html += '<table class="table table-sm table-bordered mb-0">';
  html += '<tbody>';
  html += '<tr><th style="width:180px;">Job ID</th><td><code>' + data.job_id + '</code></td></tr>';
  html += '<tr><th>Status</th><td>' + statusBadge + '</td></tr>';
  html += '<tr><th>Device</th><td>' + data.device_name + '</td></tr>';
  html += '<tr><th>Model</th><td>' + (data.model || 'N/A') + '</td></tr>';
  html += '<tr><th>Test Type</th><td>' + data.test_type + '</td></tr>';
  html += '<tr><th>Completed At</th><td>' + (data.completed_at || 'N/A') + '</td></tr>';
  if (data.error && data.error !== "") {
    html += '<tr><th>Error</th><td><span class="text-danger">' + data.error + '</span></td></tr>';
  }
  html += '</tbody></table>';
  html += '</div></div>';

  // Per-subscription results
  if (data.listOfPaths && data.listOfPaths.length > 0) {
    // Compute global stats
    var totalLeaves = 0, passedLeaves = 0, failedLeaves = 0;
    for (var p = 0; p < data.listOfPaths.length; p++) {
      var path = data.listOfPaths[p];
      if (path.leaves) {
        for (var l = 0; l < path.leaves.length; l++) {
          totalLeaves++;
          if (path.leaves[l].test_status === "PASSED") passedLeaves++;
          else failedLeaves++;
        }
      }
    }

    html += '<div class="card mb-3">';
    html += '<div class="card-header"><strong><i class="fas fa-chart-bar me-2"></i>Test Results Overview</strong>';
    html += '<span class="float-end">';
    html += '<span class="badge bg-success me-1">' + passedLeaves + ' Passed</span>';
    html += '<span class="badge bg-danger me-1">' + failedLeaves + ' Failed</span>';
    html += '<span class="badge bg-secondary">' + totalLeaves + ' Total</span>';
    html += '</span></div>';
    html += '<div class="card-body p-0">';

    // Progress bar
    var pctPass = totalLeaves > 0 ? Math.round((passedLeaves / totalLeaves) * 100) : 0;
    html += '<div class="progress" style="height:6px;border-radius:0;">';
    html += '<div class="progress-bar bg-success" style="width:' + pctPass + '%"></div>';
    html += '<div class="progress-bar bg-danger" style="width:' + (100 - pctPass) + '%"></div>';
    html += '</div>';
    html += '</div></div>';

    // Per subscription accordion
    html += '<div class="accordion" id="subsAccordion">';
    for (var p = 0; p < data.listOfPaths.length; p++) {
      var path = data.listOfPaths[p];
      var subId = 'sub_' + p;
      var subPassed = 0, subFailed = 0;
      if (path.leaves) {
        for (var l = 0; l < path.leaves.length; l++) {
          if (path.leaves[l].test_status === "PASSED") subPassed++;
          else subFailed++;
        }
      }
      var subTotal = (path.leaves ? path.leaves.length : 0);
      var subBadgeClass = subFailed === 0 ? 'bg-success' : 'bg-danger';

      html += '<div class="accordion-item">';
      html += '<h2 class="accordion-header" id="heading_' + subId + '">';
      html += '<button class="accordion-button collapsed" type="button" data-bs-toggle="collapse" data-bs-target="#collapse_' + subId + '">';
      html += '<i class="fas fa-stream me-2"></i><code>' + path.subscription + '</code>';
      html += '<span class="ms-3 badge bg-light text-dark">Interval: ' + path.interval + 's</span>';
      html += '<span class="ms-2 badge bg-light text-dark">Origin: ' + (path.origin || 'N/A') + '</span>';
      if (path.category) {
        html += '<span class="ms-2 badge bg-light text-dark">Category: ' + path.category + '</span>';
      }
      html += '<span class="ms-auto me-2 badge ' + subBadgeClass + '">' + subPassed + '/' + subTotal + ' passed</span>';
      html += '</button></h2>';

      html += '<div id="collapse_' + subId + '" class="accordion-collapse collapse" data-bs-parent="#subsAccordion">';
      html += '<div class="accordion-body p-2">';

      // Leaves table
      if (path.leaves && path.leaves.length > 0) {
        html += '<table class="table table-sm table-striped table-bordered mb-0">';
        html += '<thead class="table-light"><tr>';
        html += '<th style="width:40px;"></th>';
        html += '<th>Leaf Path</th>';
        html += '<th style="width:120px;">Description</th>';
        html += '<th style="width:110px;">Counter Type</th>';
        html += '<th style="width:80px;">Test Type</th>';
        html += '<th style="width:80px;">Status</th>';
        html += '</tr></thead><tbody>';

        for (var l = 0; l < path.leaves.length; l++) {
          var leaf = path.leaves[l];
          var leafStatus = leaf.test_status === "PASSED"
            ? '<span class="badge bg-success">PASSED</span>'
            : '<span class="badge bg-danger">FAILED</span>';
          var leafIcon = leaf.test_status === "PASSED"
            ? '<i class="fas fa-check-circle text-success"></i>'
            : '<i class="fas fa-times-circle text-danger"></i>';
          var leafId = subId + '_leaf_' + l;

          html += '<tr data-bs-toggle="collapse" data-bs-target="#' + leafId + '" style="cursor:pointer;" class="' + (leaf.test_status !== "PASSED" ? "table-danger" : "") + '">';
          html += '<td class="text-center">' + leafIcon + '</td>';
          html += '<td><code style="font-size:0.8em;">' + leaf.gnmi_leaf + '</code></td>';
          html += '<td>' + (leaf.description || '') + '</td>';
          html += '<td><span class="badge bg-light text-dark">' + (leaf.counter_type || '') + '</span></td>';
          html += '<td>' + leaf.test_type + '</td>';
          html += '<td>' + leafStatus + '</td>';
          html += '</tr>';

          // Expandable detail row
          html += '<tr class="collapse" id="' + leafId + '"><td colspan="6" class="p-0">';
          html += '<div class="p-3 bg-light">';

          // Leaf meta info
          html += '<div class="row mb-2">';
          html += '<div class="col-md-4"><small class="text-muted">Specific Thresholds:</small> ' + (leaf.specific_thresholds ? 'Yes' : 'No') + '</div>';
          html += '<div class="col-md-4"><small class="text-muted">Value Ratio:</small> ' + leaf.value_ratio + '</div>';
          html += '<div class="col-md-4"><small class="text-muted">False Positive:</small> ' + leaf.false_positive + '</div>';
          html += '</div>';

          if (leaf.netconf_rpc) {
            html += '<div class="mb-2"><small class="text-muted">Netconf RPC:</small> <code style="font-size:0.8em;">' + leaf.netconf_rpc + '</code></div>';
          }
          if (leaf.netconf_leaf) {
            html += '<div class="mb-2"><small class="text-muted">Netconf Leaf:</small> <code style="font-size:0.8em;">' + leaf.netconf_leaf + '</code></div>';
          }

          // Test detail steps
          if (leaf.test_detail && leaf.test_detail.length > 0) {
            html += '<div class="mt-2"><small class="text-muted fw-bold">Test Steps:</small>';
            html += '<ol class="mb-0 mt-1" style="font-size:0.85em;">';
            for (var d = 0; d < leaf.test_detail.length; d++) {
              var stepClass = leaf.test_detail[d].toLowerCase().indexOf('error') !== -1 || leaf.test_detail[d].toLowerCase().indexOf('fail') !== -1 || leaf.test_detail[d].toLowerCase().indexOf('not found') !== -1
                ? 'text-danger' : 'text-dark';
              html += '<li class="' + stepClass + '">' + leaf.test_detail[d] + '</li>';
            }
            html += '</ol></div>';
          }

          html += '</div>';
          html += '</td></tr>';
        }

        html += '</tbody></table>';
      } else {
        html += '<p class="text-muted mb-0">No leaf results for this subscription.</p>';
      }

      html += '</div></div></div>';
    }
    html += '</div>';
  } else {
    html += '<div class="alert alert-info">No subscription results available for this job.</div>';
  }

  return html;
}

// Here is a header of CSV FILEs
// TEST_TYPE;PARENT_PATH;LEAF_PATH;COUNTER_TYPE;DESCRIPTION;CATEGORY;ORIGIN;INTERVAL_RATE;PARENT_NETCONF_RPC;LEAF_NETCONF_PATH;OVERRIDE_THRESHOLD;VALUE_CHECK_RATIO;FALSE_POSITIVE_ALLOWED;SUPPORTED_FAMILIES

// CSV Sanity Check function
// Returns an array of error objects {line: number, message: string}
// If array is empty, the CSV is valid
function validateCSV(lines) {
  var errors = [];
  var SUPPORTED_FAMILIES = ["MX", "PTX", "ACX", "EX", "QFX", "SRX", "CRPD", "CPTX", "VMX", "VSRX", "VJUNOS", "VEVO"];

  if (lines.length === 0) {
    errors.push({ line: 0, message: "CSV file is empty" });
    return errors;
  }

  // Check header (line 1)
  var header = lines[0].trim();
  if (!header.toUpperCase().startsWith("TEST_TYPE")) {
    errors.push({ line: 1, message: "First line must be the header starting with 'TEST_TYPE'" });
  }
  if (header.indexOf(";") === -1) {
    errors.push({ line: 1, message: "Header does not use ';' as separator" });
  }

  // Parse data lines (skip header)
  for (var i = 1; i < lines.length; i++) {
    var lineNum = i + 1;
    var line = lines[i];

    // Check separator
    if (line.indexOf(";") === -1) {
      errors.push({ line: lineNum, message: "Line does not use ';' as separator" });
      continue;
    }

    var cols = line.split(";");

    // Check column count
    if (cols.length !== 14) {
      errors.push({ line: lineNum, message: "Expected 14 columns, found " + cols.length });
      continue;
    }

    // Column 0: TEST_TYPE
    var testType = cols[0].trim();
    var testTypeInt = parseInt(testType, 10);
    if (isNaN(testTypeInt)) {
      errors.push({ line: lineNum, message: "Column 0 (TEST_TYPE): must be an integer" });
    } else if (testTypeInt > 3) {
      errors.push({ line: lineNum, message: "Column 0 (TEST_TYPE): Unknown Test Type '" + testTypeInt + "' (must be 0-3)" });
    }
    // Note: if testType == 0, it will be treated as 1 (no error, just a remark)

    // Column 1: PARENT_PATH
    var parentPath = cols[1].trim();
    var pathErrors1 = validatePath(parentPath, false);
    for (var e = 0; e < pathErrors1.length; e++) {
      errors.push({ line: lineNum, message: "Column 1 (PARENT_PATH): " + pathErrors1[e] });
    }

    // Column 2: LEAF_PATH
    var leafPath = cols[2].trim();
    var pathErrors2 = validatePath(leafPath, true);
    for (var e = 0; e < pathErrors2.length; e++) {
      errors.push({ line: lineNum, message: "Column 2 (LEAF_PATH): " + pathErrors2[e] });
    }

    // Column 3: COUNTER_TYPE
    var counterType = cols[3].trim().toUpperCase();
    if (counterType !== "INCREMENTAL" && counterType !== "RELATIVE" && counterType !== "CONSTANT") {
      errors.push({ line: lineNum, message: "Column 3 (COUNTER_TYPE): must be INCREMENTAL, RELATIVE, or CONSTANT (got '" + cols[3].trim() + "')" });
    }

    // Column 4: DESCRIPTION - no check
    // Column 5: CATEGORY - no check

    // Column 6: ORIGIN
    var origin = cols[6].trim().toUpperCase();
    if (origin === "") {
      errors.push({ line: lineNum, message: "Column 6 (ORIGIN): cannot be empty" });
    } else if (origin !== "OPENCONFIG" && origin !== "NATIVE") {
      errors.push({ line: lineNum, message: "Column 6 (ORIGIN): must be OPENCONFIG or NATIVE (got '" + cols[6].trim() + "')" });
    }

    // Column 7: INTERVAL_RATE
    var interval = cols[7].trim();
    var intervalInt = parseInt(interval, 10);
    if (interval === "" || isNaN(intervalInt)) {
      errors.push({ line: lineNum, message: "Column 7 (INTERVAL_RATE): must be an integer" });
    } else if (intervalInt < 0 || intervalInt > 3600) {
      errors.push({ line: lineNum, message: "Column 7 (INTERVAL_RATE): must be between 0 and 3600 (got " + intervalInt + ")" });
    }

    // Column 8: PARENT_NETCONF_RPC (only checked if TEST_TYPE == 3)
    if (testTypeInt === 3) {
      var rpc = cols[8].trim();
      if (rpc !== "") {
        var rpcErrors = validateXmlRpc(rpc);
        for (var e = 0; e < rpcErrors.length; e++) {
          errors.push({ line: lineNum, message: "Column 8 (PARENT_NETCONF_RPC): " + rpcErrors[e] });
        }
      }
    }

    // Column 9: LEAF_NETCONF_PATH (only checked if TEST_TYPE == 3)
    if (testTypeInt === 3) {
      var netconfLeaf = cols[9].trim();
      if (netconfLeaf !== "") {
        var pathErrors9 = validatePath(netconfLeaf, true);
        for (var e = 0; e < pathErrors9.length; e++) {
          errors.push({ line: lineNum, message: "Column 9 (LEAF_NETCONF_PATH): " + pathErrors9[e] });
        }
      }
    }

    // Column 10: OVERRIDE_THRESHOLD (yes/no or empty)
    var override = cols[10].trim().toUpperCase();
    if (override !== "" && override !== "YES" && override !== "NO") {
      errors.push({ line: lineNum, message: "Column 10 (OVERRIDE_THRESHOLD): must be YES, NO, or empty (got '" + cols[10].trim() + "')" });
    }

    // Column 11 & 12: only checked if OVERRIDE_THRESHOLD == YES
    if (override === "YES") {
      // Column 11: VALUE_CHECK_RATIO (0-100)
      var ratio = cols[11].trim();
      var ratioInt = parseInt(ratio, 10);
      if (ratio === "" || isNaN(ratioInt)) {
        errors.push({ line: lineNum, message: "Column 11 (VALUE_CHECK_RATIO): must be an integer when OVERRIDE_THRESHOLD is YES" });
      } else if (ratioInt < 0 || ratioInt > 100) {
        errors.push({ line: lineNum, message: "Column 11 (VALUE_CHECK_RATIO): must be between 0 and 100 (got " + ratioInt + ")" });
      }

      // Column 12: FALSE_POSITIVE_ALLOWED (0-100)
      var fp = cols[12].trim();
      var fpInt = parseInt(fp, 10);
      if (fp === "" || isNaN(fpInt)) {
        errors.push({ line: lineNum, message: "Column 12 (FALSE_POSITIVE_ALLOWED): must be an integer when OVERRIDE_THRESHOLD is YES" });
      } else if (fpInt < 0 || fpInt > 100) {
        errors.push({ line: lineNum, message: "Column 12 (FALSE_POSITIVE_ALLOWED): must be between 0 and 100 (got " + fpInt + ")" });
      }
    }

    // Column 13: SUPPORTED_FAMILIES
    var familiesStr = cols[13].trim();
    if (familiesStr === "") {
      errors.push({ line: lineNum, message: "Column 13 (SUPPORTED_FAMILIES): cannot be empty" });
    } else {
      var familyList = familiesStr.split("|");
      for (var f = 0; f < familyList.length; f++) {
        var fam = familyList[f].trim().toUpperCase();
        if (fam === "") {
          errors.push({ line: lineNum, message: "Column 13 (SUPPORTED_FAMILIES): empty family entry found" });
        } else if (SUPPORTED_FAMILIES.indexOf(fam) === -1) {
          errors.push({ line: lineNum, message: "Column 13 (SUPPORTED_FAMILIES): unsupported family '" + familyList[f].trim() + "'" });
        }
      }
    }
  }

  return errors;
}

// Validate a path with optional attributes
// isLeaf: if true, the last node must NOT have attributes
function validatePath(path, isLeaf) {
  var errors = [];

  if (path === "") {
    errors.push("path is empty");
    return errors;
  }

  // Split path into nodes (ignore leading slash)
  var rawPath = path;
  if (rawPath.startsWith("/")) {
    rawPath = rawPath.substring(1);
  }

  // Split by / but respect brackets
  var nodes = splitPathNodes(rawPath);

  for (var n = 0; n < nodes.length; n++) {
    var node = nodes[n];
    var isLastNode = (n === nodes.length - 1);

    // Extract node name and attributes
    var bracketStart = node.indexOf("[");
    var nodeName = bracketStart === -1 ? node : node.substring(0, bracketStart);

    if (nodeName === "") {
      errors.push("empty node name found in path");
      continue;
    }

    // Check attributes if present
    if (bracketStart !== -1) {
      if (isLeaf && isLastNode) {
        errors.push("last node of leaf path must not have attributes (found in '" + node + "')");
        continue;
      }

      var attrPart = node.substring(bracketStart);
      var attrErrors = validateAttributes(attrPart);
      for (var a = 0; a < attrErrors.length; a++) {
        errors.push(attrErrors[a] + " in node '" + node + "'");
      }
    }
  }

  return errors;
}

// Split a path string into nodes respecting brackets
function splitPathNodes(path) {
  var nodes = [];
  var current = "";
  var inBracket = 0;

  for (var i = 0; i < path.length; i++) {
    var ch = path[i];
    if (ch === "[") {
      inBracket++;
      current += ch;
    } else if (ch === "]") {
      inBracket--;
      current += ch;
    } else if (ch === "/" && inBracket === 0) {
      if (current !== "") {
        nodes.push(current);
      }
      current = "";
    } else {
      current += ch;
    }
  }
  if (current !== "") {
    nodes.push(current);
  }
  return nodes;
}

// Validate attribute string like [att1=value1][att2=value2]
function validateAttributes(attrStr) {
  var errors = [];
  var regex = /\[([^\]]*)\]/g;
  var match;
  var found = false;

  while ((match = regex.exec(attrStr)) !== null) {
    found = true;
    var content = match[1];
    var eqIndex = content.indexOf("=");
    if (eqIndex === -1) {
      errors.push("attribute missing '=' sign: [" + content + "]");
      continue;
    }
    var attrName = content.substring(0, eqIndex);
    var attrValue = content.substring(eqIndex + 1);

    if (attrName.trim() === "") {
      errors.push("attribute name is empty: [" + content + "]");
    }
    if (attrValue.trim() === "") {
      errors.push("attribute value is empty: [" + content + "]");
    }
  }

  // Check for leftover characters outside brackets
  var stripped = attrStr.replace(/\[[^\]]*\]/g, "");
  if (stripped.trim() !== "") {
    errors.push("malformed attribute syntax (extra characters outside brackets)");
  }

  if (!found) {
    errors.push("malformed attribute syntax (no valid [attr=value] found)");
  }

  return errors;
}

// Validate XML RPC string
// Supported formats: <node1><node2>value</node2></node1>, <node1/>, <node1><node2/></node1>
function validateXmlRpc(rpc) {
  var errors = [];

  // Basic check: must start with < and end with >
  if (!rpc.startsWith("<") || !rpc.endsWith(">")) {
    errors.push("must be valid XML (should start with '<' and end with '>')");
    return errors;
  }

  // Try to parse as XML using DOMParser
  var parser = new DOMParser();
  var doc = parser.parseFromString(rpc, "application/xml");
  var parseError = doc.getElementsByTagName("parsererror");
  if (parseError.length > 0) {
    errors.push("invalid XML syntax");
  }

  return errors;
}
