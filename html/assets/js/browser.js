let eventSource;
const browseButton = document.getElementById("browse");
const modal = document.getElementById("modalcore")

browseButton.addEventListener("click", function () {
  
  var p = document.getElementById("pathName").value.trim();
  var m = document.getElementById("merge").checked;
  var r = document.getElementById("router").value.trim();

  var dataToSend = {"shortname": r, "xpath": p, "merge": m};
  fetch("/searchxpath", {
      method: "POST",
      headers: {
          "Content-Type": "application/json",
      },
      body: JSON.stringify(dataToSend),
  })
      .then(response => response.json())
      .then(data => {
          alertify.success("XPATH search started...");
          browseButton.disabled = true;
          $('#logs').modal('show');
          // Start the EventSource for streaming
          eventSource = new EventSource("/stream");

          eventSource.onmessage = function(event) {
              const data = JSON.parse(event.data);
              modal.innerHTML += data.msg + '<br>';
              modal.scrollTop = modal.scrollHeight;
              if (data.status == "END") {
                alertify.alert("JSTO...", "Streaming terminé");
                eventSource.close();
                browseButton.disabled = false;
              }
          };

          eventSource.onerror = function(event) {
              alertify.alert("JSTO...", "Unexpected error: " + event);
              browseButton.disabled = false;
              $('#logs').modal('hide');
              eventSource.close();
          };
      })
      .catch(error => {
        alertify.alert("JSTO...", "Unexpected error: " + error);
        browseButton.disabled = false;
        $('#logs').modal('hide');
      });
});
