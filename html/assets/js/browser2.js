let eventSource;
const browseButton = document.getElementById("browse");
const exportButton = document.getElementById("export");
const modal = document.getElementById("modalcore");
const tick = document.getElementById("tick");

modal.style.scrollBehavior = 'smooth';

document.querySelector("meta[http-equiv='Content-Security-Policy']")
  ?.setAttribute("content", "upgrade-insecure-requests");

$(document).ready(function () {

  // Initialize Fancytree
  $("#result").fancytree({
    extensions: ["filter"],
    quicksearch: true,
    filter: {
      autoExpand: false,
      mode: "dimm", 
    },
    source: [],
    strings: {
        noData: ""
    },
  });

  // Search input
    $('#searching').on('input', function () {
    const searchString = $(this).val();
    const tree = $("#result").fancytree("getTree");

    if (searchString) {
        tree.filterNodes(searchString, { autoExpand: true });
    } else {
        tree.clearFilter(); // restore normal view
    }
    });

  // Collapse all
  $('#collapse').on('click', function () {
    const tree = $("#result").fancytree("getTree");
    tree.visit(function(node){
      node.setExpanded(false);
    });
  });

  // Expand all
  $('#expand').on('click', function () {
    const tree = $("#result").fancytree("getTree");
    tree.visit(function(node){
      node.setExpanded(true);
    });
  });

});

browseButton.addEventListener("click", function () {
  
  const p = document.getElementById("pathName").value.trim();
  const m = document.getElementById("merge").checked;
  const r = document.getElementById("router").value.trim();
  browseButton.disabled = true;
  modal.innerHTML = '';
  tick.setAttribute('data-value', 0);

  const dataToSend = { "shortname": r, "xpath": p, "merge": m };
  exportButton.disabled = true;

  fetch("/searchxpath", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(dataToSend)
  })
  .then(response => response.json())
  .then(data => {
    

    eventSource = new EventSource("/stream");
    $('#logs').modal('show');

    eventSource.onmessage = function(event) {
      const data = JSON.parse(event.data);

      if (data.status === "OK") {
        appendContent(data.msg);
        scrollToBottom();
      }
      if (data.status === "XPATH") {
        tick.setAttribute('data-value', data.msg);
      }
      if (data.status === "END") {
        appendContent(data.msg);
        scrollToBottom();
        eventSource.close();
        browseButton.disabled = false;
        exportButton.disabled = false;

        // Update Fancytree data
        const tree = $("#result").fancytree("getTree");
        tree.reload(JSON.parse(data.payload));

        alertify.success('Here the results!');
      }
      if (data.status === "ERROR") {
        eventSource.close();
        browseButton.disabled = false;
        const tree = $("#result").fancytree("getTree");
        tree.reload([]);
        $('#logs').modal('hide');
        alertify.alert("JSTO...", data.msg);
      }
    };

    eventSource.onerror = function(event) {
      browseButton.disabled = false;
      const tree = $("#result").fancytree("getTree");
      tree.reload([]);
      $('#logs').modal('hide');
      eventSource.close();
      alertify.alert("JSTO...", "Unexpected error: " + JSON.stringify(event));
    };

  })
  .catch(error => {
    browseButton.disabled = false;
    const tree = $("#result").fancytree("getTree");
    tree.reload([]);
    $('#logs').modal('hide');
    if (eventSource) eventSource.close();
    alertify.alert("JSTO...", "Unexpected error: " + JSON.stringify(error));
  });
});

// Function to append new content
function appendContent(text) {
  const newElement = document.createElement('div');
  newElement.innerHTML = text;
  modal.appendChild(newElement);
}

// Scroll smoothly to bottom
function scrollToBottom() {
  modal.scrollTop = modal.scrollHeight;
}

// Close modal
document
  .querySelector('#logs .close')
  .addEventListener('click', function () {
    $('#logs').modal('hide');
  });

// Export function
function exportXpath() {
  const fileUrl = "rawfiles/xpaths-result.txt"; 
  window.open(fileUrl, "_blank");
}