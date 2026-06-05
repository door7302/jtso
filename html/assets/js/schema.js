
const DownloadButton = document.getElementById("download");

DownloadButton.addEventListener("click", function () {
  
  const r = document.getElementById("router").value.trim();
  // split the value to get shortname, hostname, model, version
  const [shortname, hostname, model, version, family] = r.split("#");
  DownloadButton.disabled = true;
  alertify.confirm("Make sure you have enable netconf-monitoring option on the router to allow schema downloading", function (e) {
      if (e) {
        // Show progress area
        const progressDiv = document.getElementById("download-progress");
        if (progressDiv) {
          progressDiv.style.display = "block";
          progressDiv.innerHTML = '<div class="alert alert-info"><strong>Starting...</strong></div>';
        }

        const params = new URLSearchParams({
          hostname: hostname,
          shortname: shortname,
          model: model,
          version: version
        });

        const evtSource = new EventSource("/downloadyang?" + params.toString());

        evtSource.addEventListener("progress", function(event) {
          if (progressDiv) {
            progressDiv.innerHTML = '<div class="alert alert-info"><i class="bi bi-arrow-repeat spin"></i> <strong>' + event.data + '</strong></div>';
          }
        });

        evtSource.addEventListener("folder", function(event) {
          // Add the new folder to the schema folder dropdown
          const select = document.getElementById("schemaFolder");
          if (select) {
            const exists = Array.from(select.options).some(opt => opt.value === event.data);
            if (!exists) {
              const option = new Option(event.data, event.data, true, true);
              select.add(option);
            } else {
              select.value = event.data;
            }
          }
        });

        evtSource.addEventListener("done", function(event) {
          evtSource.close();
          if (progressDiv) {
            progressDiv.innerHTML = '<div class="alert alert-success"><i class="bi bi-check-circle"></i> <strong>' + event.data + '</strong></div>';
          }
          DownloadButton.disabled = false;
        });

        evtSource.addEventListener("error", function(event) {
          evtSource.close();
          if (progressDiv) {
            progressDiv.innerHTML = '<div class="alert alert-danger"><i class="bi bi-x-circle"></i> <strong>' + (event.data || "Connection error") + '</strong></div>';
          }
          DownloadButton.disabled = false;
        });

        evtSource.onerror = function() {
          evtSource.close();
          if (progressDiv) {
            progressDiv.innerHTML = '<div class="alert alert-danger"><i class="bi bi-x-circle"></i> <strong>Connection lost</strong></div>';
          }
          DownloadButton.disabled = false;
        };
      } else {
        DownloadButton.disabled = false;
      }
    }).setHeader('JSTO...');
});

// Schema folder: load available schemas on button click
const loadSchemasBtn = document.getElementById("loadSchemas");
if (loadSchemasBtn) {
  loadSchemasBtn.addEventListener("click", loadSchemas);
}

function loadSchemas() {
  const folder = document.getElementById("schemaFolder").value.trim();
  const navigator = document.getElementById("navigator");
  if (!folder || !navigator) return;

  navigator.innerHTML = '<div class="text-center"><i class="bi bi-arrow-repeat spin"></i> Loading...</div>';

  $.ajax({
    type: 'GET',
    url: "/listschemas?folder=" + encodeURIComponent(folder),
    dataType: "json",
    success: function(json) {
      if (json["status"] === "OK") {
        const schemas = json["schemas"] || [];
        if (schemas.length === 0) {
          navigator.innerHTML = '<div class="alert alert-info">No schemas found in this folder.</div>';
          return;
        }
        // Store for filtering
        loadedSchemas = schemas;
        loadedFolder = folder;
        renderSchemaGrid("");
      } else {
        navigator.innerHTML = '<div class="alert alert-danger">' + json["msg"] + '</div>';
      }
    },
    error: function() {
      navigator.innerHTML = '<div class="alert alert-danger">Failed to load schemas.</div>';
    }
  });
}

let loadedSchemas = [];
let loadedFolder = "";

function renderSchemaGrid(filter) {
  const navigator = document.getElementById("navigator");
  if (!navigator) return;
  const f = filter.toLowerCase();

  // Only create the filter input once, then just update the grid
  let filterInput = document.getElementById("schemaGridFilter");
  let gridContainer = document.getElementById("schemaGridContent");

  if (!filterInput) {
    navigator.innerHTML = '<input type="text" class="form-control form-control-sm mb-3" id="schemaGridFilter" placeholder="Filter schemas...">' +
      '<div id="schemaGridContent"></div>' +
      '<div id="schemaGridCount" class="text-muted small mt-2"></div>';
    filterInput = document.getElementById("schemaGridFilter");
    gridContainer = document.getElementById("schemaGridContent");
  }

  filterInput.value = filter;

  let html = '<div class="row g-2">';
  let count = 0;
  for (let i = 0; i < loadedSchemas.length; i++) {
    if (f && !loadedSchemas[i].toLowerCase().includes(f)) continue;
    html += '<div class="col-md-4">';
    html += '<a href="#" class="card schema-item text-decoration-none h-100" data-schema="' + loadedSchemas[i] + '" data-folder="' + loadedFolder + '">';
    html += '<div class="card-body d-flex align-items-center py-2 px-3">';
    html += '<i class="bi bi-file-earmark-code me-2 text-success"></i>';
    html += '<span class="small">' + loadedSchemas[i] + '</span>';
    html += '</div></a></div>';
    count++;
  }
  html += '</div>';
  gridContainer.innerHTML = html;
  document.getElementById("schemaGridCount").textContent = count + ' / ' + loadedSchemas.length + ' schemas';
}

// Live filter on schema grid
$(document).on("input", "#schemaGridFilter", function() {
  renderSchemaGrid($(this).val());
});

// --- Schema detail modal ---
let schemaData = [];
let currentSchemaName = "";

// Delegate click on schema items
$(document).on("click", ".schema-item", function(e) {
  e.preventDefault();
  const schema = $(this).data("schema");
  const folder = $(this).data("folder");
  currentSchemaName = schema;
  openSchemaModal(folder, schema);
});

function openSchemaModal(folder, schema) {
  $("#schemaModalLabel").text(schema);
  $("#schemaTableBody").html('<tr><td colspan="3" class="text-center"><i class="bi bi-arrow-repeat spin"></i> Loading...</td></tr>');
  $("#schemaCount").text("");

  // Reset filters
  $("#filterXpath").val("");
  $("#filterDesc").val("");
  $("#filterType").val("");

  // Show modal
  const modal = new bootstrap.Modal(document.getElementById("schemaModal"));
  modal.show();

  $.ajax({
    type: 'GET',
    url: "/getschema?folder=" + encodeURIComponent(folder) + "&schema=" + encodeURIComponent(schema),
    dataType: "json",
    success: function(json) {
      if (Array.isArray(json)) {
        schemaData = json.sort(function(a, b) {
          return (a.xpath || "").localeCompare(b.xpath || "");
        });
        renderSchemaTable();
      } else {
        $("#schemaTableBody").html('<tr><td colspan="3" class="text-danger">' + (json["msg"] || "Error loading schema") + '</td></tr>');
      }
    },
    error: function() {
      $("#schemaTableBody").html('<tr><td colspan="3" class="text-danger">Failed to load schema data.</td></tr>');
    }
  });
}

function renderSchemaTable() {
  const fXpath = $("#filterXpath").val().toLowerCase();
  const fDesc = $("#filterDesc").val().toLowerCase();
  const fType = $("#filterType").val().toLowerCase();
  const showDesc = $("#showDesc").is(":checked");
  const showType = $("#showType").is(":checked");

  // Toggle column visibility
  $(".col-desc").toggle(showDesc);
  $(".col-type").toggle(showType);

  let html = "";
  let count = 0;

  for (let i = 0; i < schemaData.length; i++) {
    const item = schemaData[i];
    const xpath = item.xpath || "";
    const desc = item.xdesc || "";
    const type = item.xtype || "";

    // Apply filters
    if (fXpath && !xpath.toLowerCase().includes(fXpath)) continue;
    if (fDesc && !desc.toLowerCase().includes(fDesc)) continue;
    if (fType && !type.toLowerCase().includes(fType)) continue;

    html += '<tr>';
    html += '<td style="word-break:break-all;">' + escapeHtml(xpath) + '</td>';
    if (showDesc) html += '<td class="col-desc">' + escapeHtml(desc) + '</td>';
    if (showType) html += '<td class="col-type"><code>' + escapeHtml(type) + '</code></td>';
    html += '</tr>';
    count++;
  }

  if (count === 0) {
    html = '<tr><td colspan="3" class="text-muted">No matching entries.</td></tr>';
  }

  $("#schemaTableBody").html(html);
  $("#schemaCount").text(count + " / " + schemaData.length + " entries");
}

function escapeHtml(text) {
  const div = document.createElement('div');
  div.appendChild(document.createTextNode(text));
  return div.innerHTML;
}

// Filter inputs
$(document).on("input", "#filterXpath, #filterDesc, #filterType", function() {
  renderSchemaTable();
});

// Toggle checkboxes
$(document).on("change", "#showDesc, #showType", function() {
  renderSchemaTable();
});

// Export CSV
$(document).on("click", "#exportCsv", function() {
  const showDesc = $("#showDesc").is(":checked");
  const showType = $("#showType").is(":checked");
  const fXpath = $("#filterXpath").val().toLowerCase();
  const fDesc = $("#filterDesc").val().toLowerCase();
  const fType = $("#filterType").val().toLowerCase();

  let csv = "xpath";
  if (showDesc) csv += ",description";
  if (showType) csv += ",type";
  csv += "\n";

  for (let i = 0; i < schemaData.length; i++) {
    const item = schemaData[i];
    const xpath = item.xpath || "";
    const desc = item.xdesc || "";
    const type = item.xtype || "";

    if (fXpath && !xpath.toLowerCase().includes(fXpath)) continue;
    if (fDesc && !desc.toLowerCase().includes(fDesc)) continue;
    if (fType && !type.toLowerCase().includes(fType)) continue;

    csv += '"' + xpath.replace(/"/g, '""') + '"';
    if (showDesc) csv += ',"' + desc.replace(/"/g, '""') + '"';
    if (showType) csv += ',"' + type.replace(/"/g, '""') + '"';
    csv += "\n";
  }

  const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
  const link = document.createElement("a");
  link.href = URL.createObjectURL(blob);
  link.download = currentSchemaName + ".csv";
  link.click();
});