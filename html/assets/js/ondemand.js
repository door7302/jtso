/* Example usage - what is an ondemand profile? 

{
    "name": "profilex",
    "routers": ["rtr1", "rtr2"],
    "entries": [
        {
            "path": "/node1/node2",
            "interval": 10,
            "aliases": [], 
            "fields": [
                { "name": "field1", "monitor": true, "rate": true, "convert": false },
                { "name": "field2", "monitor": true, "rate": false, "convert": false }
            ],
            "tags": ["tag1", "tag2"]
        }
    ]
}; 

*/

const groupbyTable = document.getElementById("groupby-table");
const fieldsTable = document.getElementById("fields-table");
const btnAnalyze = document.getElementById('analyze');
const btnAddEntry = document.getElementById('addentry');
const btnResetEntry = document.getElementById('resetentry');
const btnLoad = document.getElementById('load');
const btnSave = document.getElementById('save');
const btnSaveAs = document.getElementById('saveas');
const btnStart = document.getElementById('start');
const btnStop = document.getElementById('stop');
const btnClear = document.getElementById('clear');
const btnReset = document.getElementById('reset');
const btnGnmi = document.getElementById("gnmiconce");
const btnApply = document.getElementById('applyMonitor');
const btnCancel = document.getElementById('cancelMonitor');
const selectConfig = document.getElementById('ondemand-config');
const monitorState = document.getElementById('monitor-state');
const pathInput = document.getElementById("pathName");
const pathInterval = document.getElementById("interval");
const fieldsDiv = document.getElementById("fieldsPreview");
const tagsDiv = document.getElementById("tagsPreview");
const aliasesDiv = document.getElementById("aliasesPreview");
const aliasesList = document.getElementById("aliases-list");
const aliasesInfo = document.getElementById("aliases-info");
const r = document.getElementById('router');

var toAdd = {
    path: "",
    interval: 60,
    aliases: [],
    fields: [],
    tags: []
};

var tmpGnmi = {
    aliases: [],
    fields: [],
    tags: []
};

/* UPdate current path */
pathInput.addEventListener("input", (e) => {
    toAdd.path = e.target.value;
});

/* UPdate current path interval */
pathInterval.addEventListener("input", (e) => {
     toAdd.interval = parseInt(e.target.value, 10); 
});


// BUTTON CLICK HANDLERS
btnAnalyze.onclick = function () {
    groupbyTable.innerHTML = "";
    fieldsTable.innerHTML = "";
    aliasesList.innerHTML = "";
    aliasesInfo.classList.add("d-none");
    $('#monitor').modal('show');
}

function resetEntry() {
    toAdd = {
        path: "",
        interval: 60,
        aliases: [],
        fields: [],
        tags: []
    };
    renderPreview();
    var tmpGnmi = {
        aliases: [],
        fields: [],
        tags: []
    };
    pathInput.value = "";
    pathInterval.value = ""
}

btnResetEntry.onclick = function () {
    alertify.confirm("Are you sure you want to clear the current path search?", function (e) {
        if (e) {
            resetEntry()
        }
    }).setHeader('JSTO...');
}

btnAddEntry.onclick = function () {
    alertify.confirm("Are you sure you want to append this path in the monitoring list of the current On-demand profile?", function (e) {
        if (e) {
            // check path format
            if (toAdd.path == "") {
                alertify.alert("JSTO...", "Path could not be empty!");
                return;
            }

            var check = sanityCheckXPath(toAdd.path)
            if (check.valid == false) {
                alertify.alert("JSTO...", check.reason);
                return;
            }

            // check unicity 
            var exists = false;
            for (let i = 0; i < window.dynamicData.currentProfile.entries.length; i++) {
                if (window.dynamicData.currentProfile.entries[i].path == toAdd.path) {
                    exists = true;
                    break;
                }
            }

            if (!exists) {
                // do some additionnal checks 
                if (toAdd.fields.length == 0) {
                    alertify.alert("JSTO...", "You should add at least one field to monitor!");
                    return;
                }

                // do some additionnal checks 
                if (toAdd.interval < 2) {
                    alertify.alert("JSTO...", "Inverval must be greater than 2 secs!");
                    return;
                }

                // append the entry
                window.dynamicData.currentProfile.entries.push({
                    path: toAdd.path,
                    interval: toAdd.interval,
                    aliases: toAdd.aliases,
                    fields: toAdd.fields,
                    tags: toAdd.tags
                });
                renderResultTable(window.dynamicData.currentProfile)
                resetEntry()
                alertify.success('Sensor path added well...')
            } else {
                alertify.alert("JSTO...", "This path is already present in the monitoring list...");
            }
        }
    }).setHeader('JSTO...');
}

btnAnalyze.onclick = function () {
    groupbyTable.innerHTML = "";
    fieldsTable.innerHTML = "";
    aliasesList.innerHTML = "";
    aliasesInfo.classList.add("d-none");
    $('#monitor').modal('show');
}

btnCancel.onclick = function () {
    $('#monitor').modal('hide');
}

btnApply.onclick = function () {
    tmpGnmi = buildTmpGnmi()
    processGnmiData(tmpGnmi)
    $('#monitor').modal('hide');
}

function sanityCheckXPath(xpath) {
    if (typeof xpath !== "string") {
        return { valid: false, reason: "XPath must be a string" };
    }

    const trimmed = xpath.trim();

    // Rule 1 & 2: not empty and not "/"
    if (!trimmed || trimmed === "/") {
        return { valid: false, reason: "XPath is empty or root only" };
    }

    // Rule 3: must start with "/"
    if (!trimmed.startsWith("/")) {
        return { valid: false, reason: "XPath must start with '/'" };
    }

    // Rule 4: must not end with "/"
    if (trimmed.endsWith("/")) {
        return { valid: false, reason: "XPath must not end with '/'" };
    }

    // Remove leading "/" and split nodes
    const nodes = trimmed.slice(1).split("/");

    // Rule 5: must have at least 2 nodes
    if (nodes.length < 2) {
        return { valid: false, reason: "XPath must contain at least 2 nodes" };
    }

    // Rule 6: validate each node (with or without attributes)
    const nodeRegex = /^[a-zA-Z_][a-zA-Z0-9_-]*(\[[^\]]+\])?$/;

    for (const node of nodes) {
        if (!nodeRegex.test(node)) {
            return {
                valid: false,
                reason: `Invalid node syntax: '${node}'`
            };
        }
    }

    return { valid: true };
}

function provisionMonitorTables(data) {

    if (!groupbyTable || !fieldsTable) {
        console.warn("Tables not found in DOM");
        return;
    }

    // ===== Clear existing rows =====
    groupbyTable.innerHTML = "";
    fieldsTable.innerHTML = "";
    aliasesList.innerHTML = "";
    aliasesInfo.classList.add("d-none");

    // ===== Populate GroupBy Tags table =====
    data.tags.forEach(tag => {
        const tr = document.createElement("tr");

        tr.innerHTML = `
            <td>${tag.name}</td>
            <td class="text-center">
                <input type="checkbox" ${tag.groupby ? "checked" : ""}>
            </td>
        `;

        groupbyTable.appendChild(tr);
    });

    // ===== Populate Fields table =====
    data.fields.forEach(field => {
        const tr = document.createElement("tr");

        tr.innerHTML = `
        <td>${field.name}</td>
        <td class="text-center">
            <input type="checkbox" data-role="monitor" ${field.monitor ? "checked" : ""}>
        </td>
        <td class="text-center">
            <input type="checkbox" data-role="rate" ${field.rate ? "checked" : ""}>
        </td>
        <td class="text-center">
            <input type="checkbox" data-role="convert" ${field.convert ? "checked" : ""}>
        </td>
    `;
        fieldsTable.appendChild(tr);
    });

    // ===== Populate Aliases =====
    data.aliases.forEach((alias, i) => {
        aliasesList.insertAdjacentHTML("beforeend", `
        <div class="form-check">
            <input class="form-check-input" type="checkbox" id="alias${i}" value="${alias}">
            <label class="form-check-label" for="alias${i}">
                ${alias}
            </label>
        </div>
    `);
    });

    if (Array.isArray(data.aliases) && data.aliases.length > 0) {
        aliasesInfo.classList.remove("d-none");
    }

}

btnGnmi.onclick = function () {
    var check = sanityCheckXPath(toAdd.path)
    if (check.valid == false) {
        alertify.alert("JSTO...", check.reason);
        $('#monitor').modal('hide');
        return
    }
    var selectedRouter = r.value.trim();
    btnGnmi.disabled = true;
    waitingDialog.show();
    // send data
    $(function () {
        $.ajax({
            type: 'POST',
            url: "/ondemandmgt",
            data: JSON.stringify({
                "action": "gnmionce",
                "path": toAdd.path,
                "router": selectedRouter,
                "data": ""
            }),
            contentType: "application/json",
            dataType: "json",
            success: function (json) {
                if (json.status == "OK") {
                    btnGnmi.disabled = false;
                    waitingDialog.hide();
                    /* received payload 
                    Data = {
                        aliases: [],
                        tags: [
                            { name: "host", groupby: true },
                            { name: "interface", groupby: false }
                        ],
                        fields: [
                            { name: "in-octets", monitor: true, rate: true, convert: false },
                            { name: "out-octets", monitor: false, rate: false, convert: false }
                        ]
                    }; */
                    // save the reply from JTSO 
                    provisionMonitorTables(json.data);
                } else {
                    btnGnmi.disabled = false;
                    waitingDialog.hide();
                    alertify.alert("JSTO...", json.msg);
                }
            },
            error: function (xhr, ajaxOptions, thrownError) {
                btnGnmi.disabled = false;
                waitingDialog.hide();
                alertify.alert("JSTO...", "Unexpected error");
            }
        });
    });
}

btnLoad.onclick = function () {
    const selectedConfig = selectConfig.value;
    if (selectedConfig != "default") {
        alertify.confirm("Are you sure you want to load the " + selectedConfig + " config file?", function (e) {
            if (e) {
                $(function () {
                    $.ajax({
                        type: 'POST',
                        url: "/ondemandmgt",
                        data: JSON.stringify({
                            "action": "load",
                            "data": selectConfig
                        }),
                        contentType: "application/json",
                        dataType: "json",
                        success: function (json) {
                            if (json["status"] == "OK") {
                                alertify.success('File ' + selectedConfig + ' has been loaded.')
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
    } else {
        alertify.alert("JTSO...", "Please select an config file in the list");
    }
};

btnSave.onclick = function () {
    const selectedConfig = selectConfig.value;
    if (selectedConfig != "default") {
        alertify.confirm("Are you sure you want to override the " + selectedConfig + " config file?", function (e) {
            if (e) {
                $(function () {
                    $.ajax({
                        type: 'POST',
                        url: "/ondemandmgt",
                        data: JSON.stringify({
                            "action": "save",
                            "data": selectConfig
                        }),
                        contentType: "application/json",
                        dataType: "json",
                        success: function (json) {
                            if (json["status"] == "OK") {
                                alertify.success('File ' + selectedConfig + ' has been saved.')
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
    } else {

    }
};

btnSaveAs.onclick = function () {
    alertify
        .prompt(
            'Please enter a name for your config: ',
            function (evt, value) {
                // OK click
                $.ajax({
                    type: 'POST',
                    url: '/ondemandmgt',
                    data: JSON.stringify({
                        action: 'saveas',
                        data: value
                    }),
                    contentType: 'application/json',
                    dataType: 'json',
                    success: function (json) {
                        if (json.status === 'OK') {
                            alertify.success('File ' + value + ' has been saved.')
                        } else {
                            alertify.alert('JSTO...', json.msg);
                        }
                    },
                    error: function () {
                        alertify.alert('JSTO...', 'Unexpected error...');
                    }
                });
            },
            function () {
                // Cancel clicked
                alertify.message('Operation cancelled');
            }
        ).setHeader('JSTO...');
};

btnStart.onclick = function () {
    alertify.confirm("Are you sure you want to apply the current on-demand configuration and start data collection?", function (e) {
        if (e) {
            $(function () {
                $.ajax({
                    type: 'POST',
                    url: "/ondemandmgt",
                    data: JSON.stringify({
                        "action": "start",
                        "data": ""
                    }),
                    contentType: "application/json",
                    dataType: "json",
                    success: function (json) {
                        if (json["status"] == "OK") {
                            alertify.success('On-demand configuration has been applied and data-collection started')
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
};

btnStop.onclick = function () {
    alertify.confirm("Are you sure you want to stop the current on-demand data collection?", function (e) {
        if (e) {
            $(function () {
                $.ajax({
                    type: 'POST',
                    url: "/ondemandmgt",
                    data: JSON.stringify({
                        "action": "stop",
                        "data": ""
                    }),
                    contentType: "application/json",
                    dataType: "json",
                    success: function (json) {
                        if (json["status"] == "OK") {
                            alertify.success('On-demand collector has been stopped')
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
};

btnClear.onclick = function () {
    alertify.confirm("Are you sure you want clear all past on-demand data from the DB?", function (e) {
        if (e) {
            $(function () {
                $.ajax({
                    type: 'POST',
                    url: "/ondemandmgt",
                    data: JSON.stringify({
                        "action": "clear",
                        "data": ""
                    }),
                    contentType: "application/json",
                    dataType: "json",
                    success: function (json) {
                        if (json["status"] == "OK") {
                            alertify.success('Past On-demand data has been cleared.')
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
};

btnReset.onclick = function () {
    alertify.confirm("Are you sure you want to clear the on-demand config and stop data collection? Make sure you have saved your on-demand config before resetting. Unsaved config will be lost.", function (e) {
        if (e) {
            $(function () {
                $.ajax({
                    type: 'POST',
                    url: "/ondemandmgt",
                    data: JSON.stringify({
                        "action": "reset",
                        "data": ""
                    }),
                    contentType: "application/json",
                    dataType: "json",
                    success: function (json) {
                        if (json["status"] == "OK") {
                            window.dynamicData.currentProfile = {
                                name: "no-name",
                                routers: [],
                                entries: []
                            }
                            alertify.success('On-demand tool has been reset.')
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
};

function renderResultTable(data) {
    const tbody = document.getElementById("result-table-body");
    tbody.innerHTML = "";

    data.entries.forEach((entry, index) => {
        const tr = document.createElement("tr");
        

        /* PATH */
        const pathTd = document.createElement("td");
        pathTd.classList.add("align-middle", "text-break");
        pathTd.textContent = entry.path;
        const badge = document.createElement("span");
        badge.classList.add("badge", "bg-info", "ms-2");
        badge.innerHTML = entry.interval + " secs";
        pathTd.appendChild(badge);

        if (entry.aliases.length !== 0) {
            const badge = document.createElement("span");
            badge.classList.add("badge", "bg-secondary", "ms-2");

            const tooltipText = entry.aliases.join("\n");

            badge.innerHTML = `
        Alias
        <i class="fa fa-info-circle ms-1"
           data-bs-toggle="tooltip"
           title="${tooltipText}"></i>
        `;

            pathTd.appendChild(badge);
        }

        tr.appendChild(pathTd);

        /* FIELDS */
        const fieldsTd = document.createElement("td");
        const fieldsWrap = document.createElement("div");
        fieldsWrap.className = "d-flex flex-wrap gap-1";

        entry.fields.forEach(field => {
            const badge = document.createElement("span");
            badge.className = "badge bg-warning text-dark";

            badge.innerHTML = field.name;

            if (field.rate || field.converter) {
                badge.innerHTML +=
                    ' <i class="fa fa-info-circle ms-1" data-bs-toggle="tooltip" title="Processor enabled"></i>';
            }

            fieldsWrap.appendChild(badge);
        });

        fieldsTd.appendChild(fieldsWrap);
        tr.appendChild(fieldsTd);

        /* TAGS */
        const tagsTd = document.createElement("td");
        const tagsWrap = document.createElement("div");
        tagsWrap.className = "d-flex flex-wrap gap-1";

        entry.tags.forEach(tag => {
            const badge = document.createElement("span");
            badge.className = "badge bg-primary";
            badge.textContent = tag.name;
            tagsWrap.appendChild(badge);
        });

        tagsTd.appendChild(tagsWrap);
        tr.appendChild(tagsTd);

        /* ACTIONS */
        const actionsTd = document.createElement("td");
        actionsTd.className = "text-center align-middle";

        /* Don't support edit as of now 
               <i class="fa fa-pencil-alt text-primary me-3 action-edit"
               role="button"
               title="Edit"
               data-index="${index}"></i>
        */
        actionsTd.innerHTML = `
            <i class="fa fa-trash text-danger action-delete"
               role="button"
               title="Delete"
               data-index="${index}"></i>
        `;

        tr.appendChild(actionsTd);
        tbody.appendChild(tr);
    });

    // update tooltip
    var tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'))
    var tooltipList = tooltipTriggerList.map(function (tooltipTriggerEl) {
        return new bootstrap.Tooltip(tooltipTriggerEl)
    });
}

function buildTmpGnmi() {
    const tmpGnmi = {
        aliases: [],
        fields: [],
        tags: []
    };

    /* ==========================
       TAGS (groupby-table)
       ========================== */
    document
        .querySelectorAll("#groupby-table tr")
        .forEach(row => {
            const tagName = row.cells[0]?.textContent.trim();
            const checkbox = row.querySelector("input[type='checkbox']");

            if (checkbox && checkbox.checked && tagName) {
                tmpGnmi.tags.push({
                    name: tagName,
                    groupby: true
                });
            }
        });

    /* ==========================
       ALIASES (aliases-list)
       ========================== */
    document
        .querySelectorAll("#aliases-list input[type='checkbox']:checked")
        .forEach(cb => {
            tmpGnmi.aliases.push(cb.value);
        });

    /* ==========================
       FIELDS (fields-table)
       ========================== */
    document
        .querySelectorAll("#fields-table tr")
        .forEach(row => {
            const name = row.cells[0]?.textContent.trim();

            const monitorCb = row.querySelector("input[data-role='monitor']");
            const rateCb = row.querySelector("input[data-role='rate']");
            const convertCb = row.querySelector("input[data-role='convert']");

            if (monitorCb && monitorCb.checked && name) {
                tmpGnmi.fields.push({
                    name: name,
                    monitor: true,
                    rate: !!rateCb?.checked,
                    convert: !!convertCb?.checked
                });
            }
        });

    return tmpGnmi;
}

function processGnmiData(tmpGnmi) {
    /* ==========================
       FIELDS
       ========================== */
    tmpGnmi.fields.forEach(field => {
        const processor = (field.rate === true || field.convert === true) ? 1 : 0;

        const existing = toAdd.fields.find(f => f.name === field.name);

        if (existing) {
            // update existing entry
            existing.monitor = field.monitor;
            existing.convert = field.convert;
            existing.rate = field.rate;
        } else {
            // add new entry
            toAdd.fields.push({
                name: field.name,
                monitor: field.monitor,
                convert: field.convert,
                rate: field.rate
            });
        }
    });

    /* ==========================
       TAGS
       ========================== */
    tmpGnmi.tags.forEach(tag => {
        if (!toAdd.tags.includes(tag)) {
            toAdd.tags.push({
                name: tag.name,
                groupby: tag.groupby
            });
        }
    });

    /* ==========================
   Aliases
   ========================== */
    tmpGnmi.aliases.forEach(alias => {
        if (!toAdd.aliases.includes(alias)) {
            toAdd.aliases.push(alias);
        }
    });

    renderPreview();
}

/* ADD FIELD */
function addField() {
    var name = fieldName.value.trim();
    if (!name) return;

    name = normalizeFieldPath(name);
    toAdd.fields.push({ "name": name, "monitor": true, "convert": convertCheck.checked, "rate": rateCheck.checked });

    fieldName.value = "";
    convertCheck.checked = false;
    rateCheck.checked = false;

    bootstrap.Modal.getInstance(fieldModal).hide();
    renderPreview();
}

/* ADD TAG */
function addTag() {
    var name = tagName.value.trim();
    if (!name) return;

    name = normalizePath(name);
    toAdd.tags.push({ "name": name, "groupby": true });

    tagName.value = "";
    bootstrap.Modal.getInstance(tagModal).hide();
    renderPreview();
}

/* ADD Alias */
function addAlias() {
    var name = aliasName.value.trim();
    if (!name) return;

    name = normalizePath(name);
    toAdd.aliases.push(name);

    aliasName.value = "";
    bootstrap.Modal.getInstance(aliasModal).hide();
    renderPreview();
}

function normalizePath(name) {
    if (name.startsWith("./")) {
        return "/" + name.slice(2);
    }
    if (name.startsWith(".") || name.startsWith("/")) {
        return name;
    }
    return "/" + name;
}

function normalizeFieldPath(name) {
    if (name.startsWith(".") || name.startsWith("/")) {
        return name;
    }
    return "/" + name;
}

/* REMOVE */
function removeField(idx) {
    toAdd.fields.splice(idx, 1);
    renderPreview();
}

function removeTag(idx) {
    toAdd.tags.splice(idx, 1);
    renderPreview();
}

function removeAlias(idx) {
    toAdd.aliases.splice(idx, 1);
    renderPreview();
}

/* RENDER */
function renderPreview() {

    fieldsDiv.innerHTML = "";
    tagsDiv.innerHTML = "";
    aliasesDiv.innerHTML = "";

    toAdd.fields.forEach((f, i) => {
        fieldsDiv.innerHTML += `
            <span class="badge bg-warning text-dark">
                ${f.name}
                ${f.rate != 0 || f.convert ? '<i class="fa fa-info-circle ms-1" data-bs-toggle="tooltip" title="Processor enabled"></i>' : ''}
                <i class="fa fa-times ms-1 text-danger"
                   role="button"
                   onclick="removeField(${i})"></i>
            </span>
        `;
    });

    toAdd.tags.forEach((t, i) => {
        tagsDiv.innerHTML += `
            <span class="badge bg-primary">
                ${t.name}
                <i class="fa fa-times ms-1 text-light"
                   role="button"
                   onclick="removeTag(${i})"></i>
            </span>
        `;
    });

    toAdd.aliases.forEach((t, i) => {
        aliasesDiv.innerHTML += `
            <span class="badge bg-secondary">
                ${t}
                <i class="fa fa-times ms-1 text-light"
                   role="button"
                   onclick="removeAlias(${i})"></i>
            </span>
        `;
    });

    // update tooltip
    var tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'))
    var tooltipList = tooltipTriggerList.map(function (tooltipTriggerEl) {
        return new bootstrap.Tooltip(tooltipTriggerEl)
    });
}

// Close modal
document
    .querySelector('#monitor .close')
    .addEventListener('click', function () {
        $('#monitor').modal('hide');
    });

document
    .getElementById("result-table-body")
    .addEventListener("click", function (e) {

        alertify.confirm("Are you sure you want to remove this path from the monitoring list?", function (f) {
            if (f) {
                const deleteBtn = e.target.closest(".action-delete");
                if (!deleteBtn) return;

                const row = deleteBtn.closest("tr");
                if (!row) return;

                /* ==========================
                   Extract PATH (first column)
                   ========================== */
                const pathCell = row.cells[0];

                // Get only the text node (exclude alias badge)
                const pathName = pathCell.childNodes[0].textContent.trim();


                /* ==========================
                   Remove entry from window.dynamicData.currentProfile
                   ========================== */
                window.dynamicData.currentProfile.entries = window.dynamicData.currentProfile.entries.filter(
                    entry => entry.path !== pathName
                );

                /* ==========================
                   Re-render table
                   ========================== */
                renderResultTable(window.dynamicData.currentProfile);
            }
        }).setHeader('JSTO...');
    });

document.addEventListener("DOMContentLoaded", function () {
    initApp();
});

function initApp() {
    renderResultTable(window.dynamicData.currentProfile)
}
