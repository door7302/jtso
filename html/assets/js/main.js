function emptydb() {
  alertify.confirm("Are you sure you want to empty the DB? All data will be lost.", function (e) {
    if (e) {
      $(function () {
        $.ajax({
          type: 'POST',
          url: "/influxmgt",
          data: JSON.stringify({
            "action": "emptydb",
            "data": ""
          }),
          contentType: "application/json",
          dataType: "json",
          success: function (json) {
            if (json["status"] == "OK") {
              alertify.success('The JTS DB has been successfully empty')
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

function changeRP() {
alertify
  .prompt(
    'Change the Retention Policy duration',
    'Please enter a duration (e.g. 90d, 2m, 150h):',
    '30d',
    function (evt, value) {
      // OK clicked

      // validate duration
      if (!/^[0-9]+(d|h|m|s)$/.test(value)) {
        alertify.error('Invalid duration format (use 90d, 2m, 150h...)');
        return false; // keep dialog open
      }

      $.ajax({
        type: 'POST',
        url: '/influxmgt',
        data: JSON.stringify({
          action: 'changeduration',
          data: value
        }),
        contentType: 'application/json',
        dataType: 'json',
        success: function (json) {
          if (json.status === 'OK') {
            alertify.success(
              'The JTS Retention Policy Duration has been successfully changed to (' + value + ')'
            );
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
}


function enableDebug(element) {

  const elementId = element.id;

  if (element.style.backgroundColor == 'red') {

    alertify.confirm("Are you sure you want to disable Debug mode for the " + elementId + " Telegraf Instance.", function (e) {
      if (e) {
        $(function () {
          $.ajax({
            type: 'POST',
            url: "/updatedebug",
            data: JSON.stringify({
              "instance": elementId
            }),
            contentType: "application/json",
            dataType: "json",
            success: function (json) {
              if (json["status"] == "OK") {
                alertify.success('The debug mode has been successfully disabled on the ' + elementId + ' Telegraf Instance.')
                element.style.backgroundColor = 'grey';
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

    alertify.confirm("Are you sure you want to enable Debug mode for the " + elementId + " Telegraf Instance?<br/><br/><b>Note:</b> The instance will be automatically reloaded. Enabling debug mode may produce a significant volume of logs.</b> You should be able to see debug logs into the file /var/log/debugtelegraf_" + elementId.toLowerCase() + ".log", function (e) {
      if (e) {
        $(function () {
          $.ajax({
            type: 'POST',
            url: "/updatedebug",
            data: JSON.stringify({
              "instance": elementId
            }),
            contentType: "application/json",
            dataType: "json",
            success: function (json) {
              if (json["status"] == "OK") {
                alertify.success('The debug mode has been successfully enabled on the ' + elementId + ' Telegraf Instance</br></br>You can now monitor file /var/log/debugtelegraf_' + elementId.toLowerCase() + '.log')
                element.style.backgroundColor = 'red';
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