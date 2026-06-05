
const DownloadButton = document.getElementById("download");

DownloadButton.addEventListener("click", function () {
  
  const r = document.getElementById("router").value.trim();
  // split the value to get shortname, hostname, model, version
  const [shortname, hostname, model, version, family] = r.split("#");
  DownloadButton.disabled = true;
  alertify.confirm("Make sure you have enable netconf-monitoring option on the router to allow schema downloading", function (e) {
      if (e) {
        waitingDialog.show();
        $(function () {
          $.ajax({
            type: 'POST',
            url: "/downloadyang",
            data: JSON.stringify({
              "shortname": shortname,
              "hostname": hostname,
              "model": model,
              "version": version,
              "family": family
            }),
            contentType: "application/json",
            dataType: "json",
            success: function (json) {
              if (json["status"] == "OK") {
                alertify.alert("JSTO...", json["Msg"]);
              } else {
                alertify.alert("JSTO...", json["Msg"]);
              }
              waitingDialog.hide();
            },
            error: function (xhr, ajaxOptions, thrownError) {
              alertify.alert("JSTO...", "Unexpected error...");
              waitingDialog.hide();
            }
          });
        });
      }
    }).setHeader('JSTO...');
    DownloadButton.disabled = false;
    
});