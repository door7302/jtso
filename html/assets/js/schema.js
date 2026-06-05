
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
              const option = new Option(event.data, event.data);
              select.add(option);
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
        let html = '<div class="list-group">';
        for (let i = 0; i < schemas.length; i++) {
          html += '<a href="#" class="list-group-item list-group-item-action schema-item" data-schema="' + schemas[i] + '" data-folder="' + folder + '">' + schemas[i] + '</a>';
        }
        html += '</div>';
        navigator.innerHTML = html;
      } else {
        navigator.innerHTML = '<div class="alert alert-danger">' + json["msg"] + '</div>';
      }
    },
    error: function() {
      navigator.innerHTML = '<div class="alert alert-danger">Failed to load schemas.</div>';
    }
  });
}