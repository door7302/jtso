
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