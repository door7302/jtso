
const btnAnalyze = document.getElementById('analyze');
const btnLoad = document.getElementById('load');
const btnSave = document.getElementById('save');
const btnSaveAs = document.getElementById('saveas');
const btnStart = document.getElementById('start');
const btnStop = document.getElementById('stop');
const btnClear = document.getElementById('clear');
const btnReset = document.getElementById('reset');
const selectConfig = document.getElementById('ondemand-config');
const monitorState = document.getElementById('monitor-state');

// BUTTON CLICK HANDLERS

btnAnalyse.onclick = function () {
    const modal = new 
    bootstrap.Modal(document.getElementById('monitor'));
    modal.show();
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

}

function changeRP() {
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
        tr.appendChild(pathTd);

        /* FIELDS */
        const fieldsTd = document.createElement("td");
        const fieldsWrap = document.createElement("div");
        fieldsWrap.className = "d-flex flex-wrap gap-1";

        entry.fields.forEach(field => {
            const badge = document.createElement("span");
            badge.className = "badge bg-warning text-dark";

            badge.innerHTML = field.name;

            if (field.processor !== 0 && field.processor !== "0") {
                badge.innerHTML +=
                    ' <i class="fa fa-info-circle ms-1" title="Processor enabled"></i>';
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
            badge.textContent = tag;
            tagsWrap.appendChild(badge);
        });

        tagsTd.appendChild(tagsWrap);
        tr.appendChild(tagsTd);

        /* ACTIONS */
        const actionsTd = document.createElement("td");
        actionsTd.className = "text-center align-middle";

        actionsTd.innerHTML = `
            <i class="fa fa-pencil-alt text-primary me-3 action-edit"
               role="button"
               title="Edit"
               data-index="${index}"></i>

            <i class="fa fa-trash text-danger action-delete"
               role="button"
               title="Delete"
               data-index="${index}"></i>
        `;

        tr.appendChild(actionsTd);
        tbody.appendChild(tr);
    });
}



/* Example usage */
const exampleJson = {
    "name": "profilex",
    "routers": ["rtr1", "rtr2"],
    "entries": [
        {
            "path": "/node1/node2",
            "fields": [
                { "name": "field1", "processor": 1 },
                { "name": "field2", "processor": "0" }
            ],
            "tags": ["tag1", "tag2"]
        }
    ]
};

renderResultTable(exampleJson);


const toAdd = {
    path: "",
    fields: [],
    tags: []
};

/* PATH */
document.getElementById("pathName").addEventListener("input", e => {
    toAdd.path = e.target.value;
});

/* ADD FIELD */
function addField() {
    const name = fieldName.value.trim();
    if (!name) return;

    const processor =
        convertCheck.checked || rateCheck.checked ? 1 : 0;

    toAdd.fields.push({ name, processor });

    fieldName.value = "";
    convertCheck.checked = false;
    rateCheck.checked = false;

    bootstrap.Modal.getInstance(fieldModal).hide();
    renderPreview();
}

/* ADD TAG */
function addTag() {
    const name = tagName.value.trim();
    if (!name) return;

    toAdd.tags.push(name);

    tagName.value = "";
    bootstrap.Modal.getInstance(tagModal).hide();
    renderPreview();
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

/* RENDER */
function renderPreview() {
    const fieldsDiv = document.getElementById("fieldsPreview");
    const tagsDiv = document.getElementById("tagsPreview");

    fieldsDiv.innerHTML = "";
    tagsDiv.innerHTML = "";

    toAdd.fields.forEach((f, i) => {
        fieldsDiv.innerHTML += `
            <span class="badge bg-warning text-dark">
                ${f.name}
                ${f.processor != 0 ? '<i class="fa fa-info-circle ms-1"></i>' : ''}
                <i class="fa fa-times ms-1 text-danger"
                   role="button"
                   onclick="removeField(${i})"></i>
            </span>
        `;
    });

    toAdd.tags.forEach((t, i) => {
        tagsDiv.innerHTML += `
            <span class="badge bg-primary">
                ${t}
                <i class="fa fa-times ms-1 text-light"
                   role="button"
                   onclick="removeTag(${i})"></i>
            </span>
        `;
    });

    console.log("toAdd =", JSON.stringify(toAdd, null, 2));
}
