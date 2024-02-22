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

          browseButton.disabled = true;

          // Start the EventSource for streaming
          eventSource = new EventSource("/stream");

          eventSource.onmessage = function(event) {
              const data = JSON.parse(event.data);
              modal.innerHTML += data.msg + '<br>';
          };

          eventSource.onerror = function(event) {
              console.error("EventSource failed:", event);
              eventSource.close();
          };
      })
      .catch(error => console.error("Error starting streaming:", error));
});
