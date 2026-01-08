
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