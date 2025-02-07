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

    alertify.confirm("Are you sure you want to enable Debug mode for the " + elementId + " Telegraf Instance?<br/><br/><b>Note:</b> The instance will be automatically reloaded. Enabling debug mode may produce a significant volume of logs.</b> You should be able to see debug logs into the file /var/tmp/debugtelegraf_" + elementId.toLowerCase() + ".log", function (e) {
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
                alertify.success('The debug mode has been successfully enabled on the ' + elementId + ' Telegraf Instance</br></br>You can now monitor file /var/log/debugtelegraf_' + elementId.toLowerCase() +'.log')
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