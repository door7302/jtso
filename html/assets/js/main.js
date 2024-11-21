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
          success: function (result) {
            if (result["status"] == "ok") {
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
  var dataToSend = {
    "debug": false
  };
  const elementId = element.id;

  if (element.style.backgroundColor == 'red') {
    dataToSend = false;
    alertify.confirm("Are you sure you want to disable Debug mode for the " + elementId + " Telegraf Instance.", function (e) {
      if (e) {
      
        element.style.backgroundColor = 'grey';
       
      }
    }).setHeader('JSTO...');
  } else {
    dataToSend = true;
    alertify.confirm("Are you sure you want to enable Debug mode for the " + elementId + " Telegraf Instance?<br/><br/><b>Note:</b> The instance will be automatically reloaded. Enabling debug mode may produce a significant volume of logs.", function (e) {
      if (e) {
        
          element.style.backgroundColor = 'red';

      }
    }).setHeader('JSTO...');
  }
  
 




}