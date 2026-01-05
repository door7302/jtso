let initialData = null;

const rootTitle = document.getElementById("rootTitle");
const summaryEl = document.getElementById("summary");
const cardsContainer = document.getElementById("cardsContainer");
const originFilter = document.getElementById("originFilter");
const statusEl = document.getElementById("status");
const toolbar = document.getElementById("toolbar");

function applyFilters() {
  if (!initialData) return;
  const origin = originFilter.value;

  const filtered = initialData.listOfPaths.filter(p => {
    if (origin && p.origin !== origin) return false;

    return true;
  });

  renderCards(filtered);
}

function renderCards(paths) {
  cardsContainer.innerHTML = "";
  if (!paths.length) {
    cardsContainer.innerHTML =
      "<div style='font-size:13px;color:#aaa;'>No paths match filters.</div>";
    return;
  }

  paths.forEach((p) => {
    const card = document.createElement("div");
    card.className = "card";

    const header = document.createElement("div");
    header.className = "card-header";

    const pathEl = document.createElement("div");
    pathEl.className = "card-path";
    pathEl.textContent = p.name;

    const meta = document.createElement("div");
    meta.className = "card-meta";

    const intervalBadge = document.createElement("div");
    intervalBadge.className = "badge";
    intervalBadge.innerHTML = `<span class="badge-label">Interval:</span><span>${p.interval} sec(s)</span>`;
    meta.appendChild(intervalBadge);

    const originBadge = document.createElement("div");
    originBadge.className =
      "badge " +
      (p.origin === "native" ? "badge-origin-native" : "badge-origin-openconfig");
    originBadge.innerHTML = `<span class="badge-label">Origin:</span><span>${p.origin}</span>`;
    meta.appendChild(originBadge);

    if (p.aliases && p.aliases.length) {
      const aliasesEl = document.createElement("div");
      aliasesEl.className = "aliases";
      p.aliases.forEach(a => {
        const pill = document.createElement("span");
        pill.className = "alias-pill badge-alias";
        pill.textContent = "Alias: " + a;
        aliasesEl.appendChild(pill);
      });
      meta.appendChild(aliasesEl);
    }

    header.appendChild(pathEl);
    header.appendChild(meta);
    card.appendChild(header);

    const fields = p.listOfFields || [];
    const fieldsContainer = document.createElement("div");
    fieldsContainer.className = "fields-container";

    if (fields.length) {
      const toggle = document.createElement("button");
      toggle.type = "button";
      toggle.className = "fields-toggle";
      toggle.innerHTML = `<span>Fields (${fields.length})</span><span>▼</span>`;

      const list = document.createElement("div");
      list.className = "fields-list";

      fields.forEach(f => {
        const item = document.createElement("div");
        item.className = "field-item";
        item.title = f;
        item.textContent = f;
        list.appendChild(item);
      });

      toggle.addEventListener("click", () => {
        const isOpen = list.classList.toggle("open");
        toggle.querySelector("span:last-child").textContent = isOpen ? "▲" : "▼";
      });

      fieldsContainer.appendChild(toggle);
      fieldsContainer.appendChild(list);
    } else {
      const empty = document.createElement("div");
      empty.className = "pill-empty";
      if (p.aliases && p.aliases.length) {
        empty.textContent = "Check Alias instead.";
      } else {
        empty.textContent = "No fields for this path.";
      }
      fieldsContainer.appendChild(empty);
    }

    card.appendChild(fieldsContainer);

    const footer = document.createElement("div");
    footer.className = "card-footer";
    footer.textContent = `${fields.length} field(s)`;
    card.appendChild(footer);

    cardsContainer.appendChild(card);
  });
}

originFilter.addEventListener("change", applyFilters);

function resetRender() {
  rootTitle.textContent = "";
  summaryEl.textContent = "";
  statusEl.textContent = "Click on a 'Show Sensors' button";
  toolbar.style.display = "none";

  // Remove all current cards
  cardsContainer.innerHTML = "";
}

function updateDoc() {
  var p = document.getElementById("profiles").value.trim();
  var desc = document.getElementById("profileDesc");
  var tele = document.getElementById("profileTele");
  var graf = document.getElementById("profileGraf");
  var kapa = document.getElementById("profileKapa");

  if (p == "default") {
    desc.innerHTML = "N/A";
    tele.innerHTML = "N/A";
    graf.innerHTML = "N/A";
    kapa.innerHTML = "N/A";
  } else {
    var dataToSend = { "profile": p };
    waitingDialog.show();
    resetRender();

    // send data
    $(function () {
      $.ajax({
        type: 'POST',
        url: "/updatedoc",
        data: JSON.stringify(dataToSend),
        contentType: "application/json",
        dataType: "json",
        success: function (json) {
          if (json.status == "OK") {
            //img.setAttribute('src', "img/" + json.img);
            desc.innerHTML = json.desc.trim();
            tele.innerHTML = json.tele.trim();
            graf.innerHTML = json.graf.trim();
            kapa.innerHTML = json.kapa.trim();
            waitingDialog.hide();
          } else {
            alertify.alert("JSTO...", json.msg);
            waitingDialog.hide();
          }
        },
        error: function (xhr, ajaxOptions, thrownError) {
          alertify.alert("JSTO...", "Unexpected error");
          waitingDialog.hide();
        }
      });
    });
  }
}

async function loadConfig(fileName) {
  try {
    const response = await fetch(fileName);
    if (!response.ok) {
      throw new Error(`Failed to load ${fileName}: ${response.statusText}`);
    }

    const jsonContent = await response.json();

    // Pretty-print JSON
    const formattedJson = JSON.stringify(jsonContent, null, 2);

    // Add syntax highlighting
    const highlightedJson = Prism.highlight(formattedJson, Prism.languages.json, 'json');

    // Update modal content with highlighted JSON
    document.getElementById('modalcore').innerHTML = `<pre><code class="language-json">${highlightedJson}</code></pre>`;

    // Show the modal
    const modal = new bootstrap.Modal(document.getElementById('config'));
    modal.show();
  } catch (error) {
    alertify.alert("JSTO...", "Error loading config: " + error);
    document.getElementById('modalcore').textContent = 'Error loading configuration.';
  }
}

function showSensor(family, profile, config) {
  try {
    var dataToSend = {
      "family": family,
      "profile": profile,
      "config": config
    };

    waitingDialog.show();

    // send data
    $(function () {
      $.ajax({
        type: 'POST',
        url: "/gettree",
        data: JSON.stringify(dataToSend),
        contentType: "application/json",
        dataType: "json",
        success: function (json) {
          if (json.status == "OK") {
            initialData = json.tree;

            rootTitle.textContent = initialData.rootName;
            const totalPaths = initialData.listOfPaths.length;
            const totalFields = initialData.listOfPaths.reduce(
              (acc, p) => acc + (p.listOfFields ? p.listOfFields.length : 0),
              0
            );
            summaryEl.textContent = `${totalPaths} paths, ${totalFields} fields`;

            statusEl.textContent = "";
            toolbar.style.display = "flex";

            renderCards(initialData.listOfPaths);
            waitingDialog.hide();
          } else {
            alertify.alert("JSTO...", json.msg);
            waitingDialog.hide();
            statusEl.className = "error";
            statusEl.textContent = "Failed to load sensors.";
          }
        },
        error: function (xhr, ajaxOptions, thrownError) {
          alertify.alert("JSTO...", "Unexpected error");
          waitingDialog.hide();
          statusEl.className = "error";
          statusEl.textContent = "Failed to load sensors.";
        }
      });
    });
  } catch (error) {
    alertify.alert("JSTO...", "Error loading sensors: " + error);
    statusEl.className = "error";
    statusEl.textContent = "Failed to load data.";
  }
}

function resetIntervals() {
  var p = document.getElementById("profiles").value.trim();
  if (p == "default") {
    alertify.alert("JSTO...", "Please select a profile.");
  } else {
    alertify.confirm("Do you want to reset the streaming interval(s) for the profile " + p + " to their default values?", function (e) {
      if (e) {
        $(function () {
          $.ajax({
            type: 'POST',
            url: "/intervalmgmt",
            data: JSON.stringify({
              "action": "reset",
              "data": p
            }),
            contentType: "application/json",
            dataType: "json",
            success: function (json) {
              if (json["status"] == "OK") {
                alertify.success('The streamming intervals for profile " + p + " have been successfully reset')
              } else {
                alertify.alert("JTSO...", json.msg);
              }
            },
            error: function (xhr, ajaxOptions, thrownError) {
              alertify.alert("JSTO...", "Unexpected error...");
            }
          });
        });
      }
    }).setHeader('JSTO...');
  }
}

function modifyIntervals() {
  var p = document.getElementById("profiles").value.trim();
  if (p == "default") {
    alertify.alert("JSTO...", "Please select a profile.");
  } else {
    $(function () {
      $.ajax({
        type: 'POST',
        url: "/intervalmgmt",
        data: JSON.stringify({
          "action": "getinterval",
          "data": p
        }),
        contentType: "application/json",
        dataType: "json",
        success: function (json) {
          if (json["status"] == "OK") {
            openIntervalModal(json.intervals)
          } else {
            alertify.alert("JTSO...", json.msg);
          }
        },
        error: function (xhr, ajaxOptions, thrownError) {
          alertify.alert("JSTO...", "Unexpected error...");
        }
      });
    });
  }
}

function openIntervalModal(data) {
  const tbody = $("#intervalTableBody");
  tbody.empty();

  data.forEach(item => {
    const configured =
      item["configured-interval"] > 0
        ? item["configured-interval"]
        : "";

    const platforms = (item.assigned && item.assigned.length)
      ? `<div class="platform-badges">
           ${item.assigned.map(p =>
             `<span class="badge badge-primary">${p}</span>`
           ).join("")}
         </div>`
      : `<span class="text-muted">—</span>`;

    const row = `
      <tr>
        <td>
          <span class="path-text" title="${item.path}">
            ${item.path}
          </span>
        </td>
        <td>${platforms}</td>
        <td>${item["default-interval"]}</td>
        <td>
          <input
            type="number"
            class="form-control interval-input"
            data-path="${item.path}"
            value="${configured}"
            min="1"
            placeholder="leave empty"
          />
        </td>
      </tr>
    `;

    tbody.append(row);
  });

  $("#intervalModal").modal("show");
}

$("#applyIntervals").on("click", function () {
  const result = [];

  $(".interval-input").each(function () {
    const value = $(this).val();
    const path = $(this).data("path");

    if (value !== "") {
      result.push({
        path: path,
        "configured-interval": parseInt(value, 10)
      });
    }
  });

  alert(JSON.stringify(result, null, 2));
  $("#intervalModal").modal("hide");
});

document.querySelector('#config .close').addEventListener('click', function () {
  const modal = document.getElementById('config');
  modal.classList.remove('show'); // Remove the `show` class
  modal.style.display = 'none';   // Hide the modal
  document.body.classList.remove('modal-open'); // Remove modal-open from body
  const backdrop = document.querySelector('.modal-backdrop');
  if (backdrop) backdrop.remove(); // Remove backdrop
});

// Enable Bootstrap tooltips
$('[data-toggle="tooltip"]').tooltip({
  container: 'body'
});