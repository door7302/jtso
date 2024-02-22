let eventSource;
const browseButton = document.getElementById("browse");
const modal = document.getElementById("modalcore")
modal.style.scrollBehavior = 'smooth';

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
          modal.innerHTML = '';
          $('#logs').modal('show');
          eventSource.onmessage = function(event) {
              const data = JSON.parse(event.data);
              appendContent(data.msg);
              scrollToBottom()
              if (data.status == "END") {
                alertify.alert("JSTO...", "Streaming terminé");
                eventSource.close();
                browseButton.disabled = false;
                $('#logs').modal('hide');
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

  // Function to append new content
  function appendContent(text) {
    var newElement = document.createElement('div');
    newElement.innerHTML = text;
    modal.appendChild(newElement);
  }

  // Function to scroll to the bottom with smooth scrolling
  function scrollToBottom() {
    modal.scrollTop = modal.scrollHeight;
  }
