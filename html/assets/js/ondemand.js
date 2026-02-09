/* Example usage - what is an ondemand profile? 

{
  "name": "profilex",
  "routers": [
    "rtr1",
    "rtr2"
  ],
  "entries": [
    {
      "path": "/node1/node2",
      "interval": 10,
      "aliases": [],
      "fields": [
        {
          "name": "field1",
          "monitor": true,
          "rate": true,
          "convert": false,
          "inherit_tags": [
              "tag1"
            ]
          }
        },
        {
          "name": "field2",
          "monitor": true,
          "rate": false,
          "convert": false,
          "inherit_tags": [
              "tag2"
            ]
          }
        }
      ]
    }
  ]
}; 

*/

const allTagsTable = document.getElementById("alltags-table");
const fieldsTable = document.getElementById("fields-table");
const btnAnalyze = document.getElementById('analyze');
const btnAddEntry = document.getElementById('addentry');
const btnResetEntry = document.getElementById('resetentry');
const btnLoad = document.getElementById('load');
const btnExport = document.getElementById('export');
const btnRouter = document.getElementById('updatertr')
const rtrList = document.getElementById('routerlist')
const btnSave = document.getElementById('save');
const btnSaveAs = document.getElementById('saveas');
const btnStart = document.getElementById('startstop');
const bntStartText = document.getElementById('btn-text');
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
const configName = document.getElementById("configName");
const r = document.getElementById('router');
const collectingIcon = document.getElementById("collecting-icon");

var toAdd = {
    path: "",
    interval: 0,
    aliases: [],
    fields: [],
};

var tmpGnmi = {
    aliases: [],
    fields: [],
};

var tmpMap = {};

var profileSaved = true;

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
    allTagsTable.innerHTML = "";
    fieldsTable.innerHTML = "";
    aliasesList.innerHTML = "";
    aliasesInfo.classList.add("d-none");
    $('#monitor').modal('show');
}

btnResetEntry.onclick = function () {
    alertify.confirm("Are you sure you want to clear the current path search?", function (e) {
        if (e) {
            resetEntry()
        }
    }).setHeader('JSTO...');
}

btnRouter.onclick = function () {

    // check if collector runs
    if (window.dynamicData.run) {
        alertify.alert("JSTO...", "Please Stop Collector before updating the on-demand router list!");
        return;
    }

    var all_selected = rtrList.options

    window.dynamicData.currentProfile.routers.length = 0

    for (var option of all_selected) {
        if (option.selected) {
            window.dynamicData.currentProfile.routers.push(option.value);
        }
    }
    alertify.success('Router list successfully updated!')

}

btnAddEntry.onclick = function () {
    alertify.confirm("Are you sure you want to append this path in the monitoring list of the current On-demand profile?", function (e) {
        if (e) {

            // Can't add entry if collection is running
            if (window.dynamicData.run) {
                alertify.alert("JSTO...", "Please Stop Collector before modifying the on-demand profile!");
                return;
            }

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
                if (toAdd.interval < 10) {
                    alertify.alert("JSTO...", "Inverval must be greater than 10 secs!");
                    return;
                }
                // A change occured
                changeProfileState(false);

                // append the entry
                window.dynamicData.currentProfile.entries.push({
                    path: toAdd.path,
                    interval: toAdd.interval,
                    aliases: toAdd.aliases,
                    fields: toAdd.fields,
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
    allTagsTable.innerHTML = "";
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

    if (!allTagsTable || !fieldsTable) {
        console.warn("Tables not found in DOM");
        return;
    }

    // ===== Clear existing rows =====
    allTagsTable.innerHTML = "";
    fieldsTable.innerHTML = "";
    aliasesList.innerHTML = "";
    aliasesInfo.classList.add("d-none");
    tmpMap = {};

    // // ===== Populate Fields table =====
    var uniqueTags = []
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

        // Get tags 
        const tagTable = [];

        field.inherit_tags.forEach(tag => {
            if (!uniqueTags.includes(tag)) {
                uniqueTags.push(tag);
            }
            tagTable.push(tag);
        });
        tmpMap[field.name] = tagTable;
    });

    // ===== Populate Tags table =====
    uniqueTags.forEach(tag => {
        const tr = document.createElement("tr");

        tr.innerHTML = `
            <td>${tag}</td>
            </td>
        `;

        allTagsTable.appendChild(tr);
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
                "data": "",
                "profile": {}
            }),
            contentType: "application/json",
            dataType: "json",
            success: function (json) {
                if (json.status == "OK") {
                    btnGnmi.disabled = false;
                    waitingDialog.hide();
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
    const config = selectConfig.value;
    if (config != "default") {
        // manage execption
        if (window.dynamicData.run) {
            alertify.alert("JSTO...", "Please stop the on-demand monitoring before loading a new config!");
            return;
        }
        alertify.confirm("Are you sure you want to load the " + config + " config file? Current config will be lost if not saved!", function (e) {
            if (e) {
                waitingDialog.show();
                $(function () {
                    $.ajax({
                        type: 'POST',
                        url: "/ondemandmgt",
                        data: JSON.stringify({
                            "action": "load",
                            "path": "",
                            "router": "",
                            "data": config,
                            "profile": {}
                        }),
                        contentType: "application/json",
                        dataType: "json",
                        success: function (json) {
                            if (json["status"] == "OK") {
                                waitingDialog.hide();

                                renderResultTable(json.profile);
                                window.dynamicData.currentProfile = json.profile;
                                changeProfileState(true);
                                alertify.success('File ' + config + ' has been loaded.');
                            } else {
                                waitingDialog.hide();
                                alertify.alert("JTSO...", json.msg);
                            }
                        },
                        error: function (xhr, ajaxOptions, thrownError) {
                            waitingDialog.hide();
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
    const config = selectConfig.value;
    if (config != "default") {
        // manage execption
        if (window.dynamicData.run) {
            alertify.alert("JSTO...", "Please stop the on-demand monitoring before saving config!");
            return;
        }

        if (window.dynamicData.currentProfile.entries.length == 0) {
            alertify.alert("JSTO...", "Please add at least one path into the monitor list!");
            return;
        }

        alertify.confirm("Are you sure you want to override the " + config + " config file?", function (e) {
            if (e) {
                window.dynamicData.currentProfile.name = config;
                waitingDialog.show();
                $(function () {
                    $.ajax({
                        type: 'POST',
                        url: "/ondemandmgt",
                        data: JSON.stringify({
                            "action": "save",
                            "path": "",
                            "router": "",
                            "data": config,
                            "profile": window.dynamicData.currentProfile
                        }),
                        contentType: "application/json",
                        dataType: "json",
                        success: function (json) {
                            if (json["status"] == "OK") {
                                waitingDialog.hide();
                                changeProfileState(true);
                                alertify.success('File ' + config + ' has been saved.')
                            } else {
                                waitingDialog.hide();
                                alertify.alert("JTSO...", json.msg);
                            }
                        },
                        error: function (xhr, ajaxOptions, thrownError) {
                            waitingDialog.hide();
                            alertify.alert("JSTO...", "Unexpected error...");
                        }
                    });
                });
            }
        }).setHeader('JSTO...');
    } else {
        save()
    }
};

btnSaveAs.onclick = function () {
    save()
};

function save() {
    // manage execption
    if (window.dynamicData.run) {
        alertify.alert("JSTO...", "Please stop the on-demand monitoring before saving config!");
        return;
    }

    if (window.dynamicData.currentProfile.entries.length == 0) {
        alertify.alert("JSTO...", "Please add at least one path into the monitor list!");
        return;
    }

    alertify
        .prompt(
            'Please enter a name for your config: ',
            'filename',
            function (evt, value) {
                // OK click
                window.dynamicData.currentProfile.name = value;
                waitingDialog.show();
                $.ajax({
                    type: 'POST',
                    url: '/ondemandmgt',
                    data: JSON.stringify({
                        "action": "save",
                        "path": "",
                        "router": "",
                        "data": value,
                        "profile": window.dynamicData.currentProfile
                    }),
                    contentType: 'application/json',
                    dataType: 'json',
                    success: function (json) {
                        if (json.status === 'OK') {
                            waitingDialog.hide();
                            changeProfileState(true);
                            alertify.success('File ' + value + ' has been saved.');
                        } else {
                            waitingDialog.hide();
                            alertify.alert('JSTO...', json.msg);
                        }
                    },
                    error: function () {
                        waitingDialog.hide();
                        alertify.alert('JSTO...', 'Unexpected error...');
                    }
                });
            },
            function () {
                // Cancel clicked
                waitingDialog.hide();
                alertify.message('Operation cancelled');
            }
        ).setHeader('JSTO...');
}


btnExport.onclick = function () {
    const config = selectConfig.value;
    if (config != "default") {

        alertify.confirm("Do you want to dowload the " + config + " json file?", function (e) {
            if (e) {
                waitingDialog.show();
                $(function () {
                    $.ajax({
                        type: 'POST',
                        url: "/ondemandmgt",
                        data: JSON.stringify({
                            "action": "export",
                            "path": "",
                            "router": "",
                            "data": config,
                            "profile": {}
                        }),
                        contentType: 'application/json',
                        dataType: 'json',
                        success: function (json) {
                            waitingDialog.hide();

                            if (json.Status !== "OK") {
                                alertify.alert("Error", json.Msg || "Unknown error");
                                return;
                            }

                            // Convert the Data field to string
                            const jsonString = JSON.stringify(json.Data, null, 2);

                            const blob = new Blob([jsonString], { type: "application/json" });
                            const url = window.URL.createObjectURL(blob);

                            const a = document.createElement("a");
                            a.href = url;
                            a.download = config + ".json";  // file name
                            a.click();

                            window.URL.revokeObjectURL(url);
                        },
                        error: function () {
                            waitingDialog.hide();
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

btnStart.onclick = function () {
    if (window.dynamicData.run) {
        // Here we should stop
        alertify.confirm("Are you sure you want to stop data collection?", function (e) {
            if (e) {
                $(function () {
                    $.ajax({
                        type: 'POST',
                        url: "/ondemandmgt",
                        data: JSON.stringify({
                            "action": "stop",
                            "path": "",
                            "router": "",
                            "data": window.dynamicData.currentProfile.name,
                            "profile": {}
                        }),
                        contentType: "application/json",
                        dataType: "json",
                        success: function (json) {
                            if (json["status"] == "OK") {
                                window.dynamicData.run = false;
                                changeBtnState(window.dynamicData.run);
                                alertify.success('On-demand data-collection has been stopped')
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
        // Here we should start

        // check if at lease there is one path in monitoring list 
        if (window.dynamicData.currentProfile.entries.length == 0) {
            alertify.alert("JSTO...", "Please add at least one path into the monitor list before starting data collection!");
            return;
        }

        // check if config as been saved
        if (profileSaved == false) {
            alertify.alert("JSTO...", "Please save your config before starting data collection!");
            return;
        }

        // check if there are selected router
        if (window.dynamicData.currentProfile.routers.length == 0) {
            alertify.alert("JSTO...", "Please select at least one router before starting data collection!");
            return;
        }

        alertify.confirm("Are you sure you want to apply the current on-demand configuration and start data collection?", function (e) {
            if (e) {
                $(function () {
                    $.ajax({
                        type: 'POST',
                        url: "/ondemandmgt",
                        data: JSON.stringify({
                            "action": "start",
                            "path": "",
                            "router": "",
                            "data": "",
                            "profile": window.dynamicData.currentProfile
                        }),
                        contentType: "application/json",
                        dataType: "json",
                        success: function (json) {
                            if (json["status"] == "OK") {
                                window.dynamicData.run = true;
                                changeBtnState(window.dynamicData.run);
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
    }
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
                        "path": "",
                        "router": "",
                        "data": "",
                        "profile": {}
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

            window.dynamicData.currentProfile = {
                name: "no-name",
                routers: [],
                entries: []
            }
            // A change occured
            window.dynamicData.run = false;
            changeProfileState(true);
            resetEntry()

            renderResultTable(window.dynamicData.currentProfile);
            alertify.success('On-demand tool has been reset.');

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

        var uniqueTags = [];
        entry.fields.forEach(field => {
            const badge = document.createElement("span");
            badge.className = "badge bg-warning text-dark";

            badge.innerHTML = field.name;

            if (field.rate || field.converter) {
                badge.innerHTML +=
                    ' <i class="fa fa-info-circle ms-1" data-bs-toggle="tooltip" title="Processor enabled"></i>';
            }

            fieldsWrap.appendChild(badge);

            // Get tags 
            field.inherit_tags.forEach(tag => {
                if (!uniqueTags.includes(tag)) {
                    uniqueTags.push(tag);
                }
            });
        });

        fieldsTd.appendChild(fieldsWrap);
        tr.appendChild(fieldsTd);

        /* TAGS */
        const tagsTd = document.createElement("td");
        const tagsWrap = document.createElement("div");
        tagsWrap.className = "d-flex flex-wrap gap-1";

        uniqueTags.forEach(tag => {
            const badge = document.createElement("span");
            badge.className = "badge bg-primary";
            badge.textContent = tag;
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
    };

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
                    convert: !!convertCb?.checked,
                    inherit_tags: tmpMap[name]
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
                rate: field.rate,
                inherit_tags: field.inherit_tags
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

    // Retrieve tags
    var tags = []
    var tagN = tagName.value.trim();
    if (tagN) {
        tags = tagN.split(";").map(item => normalizePath(item.trim()));
    }

    name = normalizeFieldPath(name);
    toAdd.fields.push({ "name": name, "monitor": true, "convert": convertCheck.checked, "rate": rateCheck.checked, "inherit_tags": tags });

    fieldName.value = "";
    tagName.value = "";
    convertCheck.checked = false;
    rateCheck.checked = false;

    bootstrap.Modal.getInstance(fieldModal).hide();
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

function removeAlias(idx) {
    toAdd.aliases.splice(idx, 1);
    renderPreview();
}

/* RENDER */
function renderPreview() {

    fieldsDiv.innerHTML = "";
    tagsDiv.innerHTML = "";
    aliasesDiv.innerHTML = "";

    var uniqueTags = [];
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
        // Get tags 
        f.inherit_tags.forEach(tag => {
            if (!uniqueTags.includes(tag)) {
                uniqueTags.push(tag);
            }
        });
    });


    uniqueTags.forEach((t, i) => {
        tagsDiv.innerHTML += `
            <span class="badge bg-primary">
                ${t}
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


function changeBtnState(collectstate) {
    if (collectstate) {
        btnSave.disabled = true;
        btnSaveAs.disabled = true;
        btnLoad.disabled = true;
        btnRouter.disable = true;
        btnAddEntry.disabled = true;
        btnReset.disabled = true;
        bntStartText.textContent = "Stop Collector"
        collectingIcon.style.display = "inline-block";
        collectingIcon.classList.add("blink");

    } else {
        btnSave.disabled = false;
        btnSaveAs.disabled = false;
        btnLoad.disabled = false;
        btnRouter.disable = false;
        btnAddEntry.disabled = false;
        btnReset.disabled = false;
        bntStartText.textContent = "Start Collector";
        collectingIcon.style.display = "none";
        collectingIcon.classList.remove("blink");
    }
}

function resetEntry() {
    toAdd = {
        path: "",
        interval: 0,
        aliases: [],
        fields: [],
    };
    renderPreview();
    tmpGnmi = {
        aliases: [],
        fields: [],
    };
    tmpMap = {};
    pathInput.value = "";
    pathInterval.value = ""
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
        const deleteBtn = e.target.closest(".action-delete");
        if (!deleteBtn) return; // exit if not delete button
        // Can't add entry if collection is running
        if (window.dynamicData.run) {
            alertify.alert("JSTO...", "Please Stop Collector before resetting the on-demand profile!");
            return;
        }

        alertify.confirm("Are you sure you want to remove this path from the monitoring list?", function (f) {
            if (f) {

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

                // A change occured
                changeProfileState(false);

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

function changeProfileState(action) {
    if (action) {
        profileSaved = true;
        configName.innerHTML = `
        <label><b>Current config:&nbsp;</b>${window.dynamicData.currentProfile.name}</label>
        `;
    } else {
        profileSaved = false;
        configName.innerHTML = `
        <label><b>Current config:&nbsp;</b>${window.dynamicData.currentProfile.name}</label>
        <i class="fa fa-save text-danger ms-2"
            style="font-size: 1rem;"
            title="Unsaved changes"></i>
        `;

    }
}

function initApp() {
    changeBtnState(window.dynamicData.run);
    renderResultTable(window.dynamicData.currentProfile);
}
