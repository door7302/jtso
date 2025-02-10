let eventSource;
const browseButton = document.getElementById("browse");
const exportButton = document.getElementById("export");
const modal = document.getElementById("modalcore")
const tick = document.getElementById("tick")

modal.style.scrollBehavior = 'smooth';

$(document).ready(function () {

  $('#searching').on('input', function () {
    var searchString = $(this).val();
    $('#result').jstree(true).search(searchString);
  });

  // Collapse button click event
  $('#collapse').on('click', function () {
    $('#result').jstree('close_all');
  });

  // Expand button click event
  $('#expand').on('click', function () {
    $('#result').jstree('open_all');
  });

  $("#result").jstree({
    "core": {
      "data": []
    },
    "themes": {
      "theme": "default",
      "dots": true,
      "icons": true
    },
    "plugins": ["state", "sort", "search"]
  });
});


browseButton.addEventListener("click", function () {

  var p = document.getElementById("pathName").value.trim();
  var m = document.getElementById("merge").checked;
  var r = document.getElementById("router").value.trim();


  var dataToSend = {
    "shortname": r,
    "xpath": p,
    "merge": m
  };
  exportButton.disabled = true;
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
      eventSource.onmessage = function (event) {
        const data = JSON.parse(event.data);
        if (data.status == "OK") {
          appendContent(data.msg);
          scrollToBottom()
        }
        if (data.status == "XPATH") {
          tick.setAttribute('data-value', data.msg);
        }
        if (data.status == "END") {
          appendContent(data.msg);
          scrollToBottom()
          eventSource.close();
          browseButton.disabled = false;
          exportButton.disabled = false;
          $('#result').jstree(true).settings.core.data = JSON.parse(data.payload);
          $('#result').jstree(true).refresh();
          //$('#logs').modal('hide');
          alertify.success('Here the results!')
        }
        if (data.status == "ERROR") {

          eventSource.close();
          browseButton.disabled = false;
          $('#result').jstree(true).settings.core.data = [];
          $('#result').jstree(true).refresh();
          $('#logs').modal('hide');
          alertify.alert("JSTO...", data.msg);
        }

      };
      eventSource.onerror = function (event) {
        browseButton.disabled = false;
        $('#result').jstree(true).settings.core.data = [];
        $('#result').jstree(true).refresh();
        $('#logs').modal('hide');
        eventSource.close();
        alertify.alert("JSTO...", "Unexpected error: " + JSON.stringify(event));
      };
    })
    .catch(error => {
      browseButton.disabled = false;
      $('#result').jstree(true).settings.core.data = JSON.parse([]);
      $('#result').jstree(true).refresh();
      $('#logs').modal('hide');
      eventSource.close();
      alertify.alert("JSTO...", "Unexpected error: " + JSON.stringify(error));
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

function closeModal() {
  $('#logs').modal('hide');
}


function exportXpath() {

  const fileUrl = "assets/rawfiles/xpaths-result.txt"; 
  const suggestedFileName = "xpaths-result.txt"; 

  // Create a link element
  const link = document.createElement('a');
  link.href = fileUrl; 
  link.download = suggestedFileName; 

  document.body.appendChild(link);

  link.click();

  document.body.removeChild(link);
}