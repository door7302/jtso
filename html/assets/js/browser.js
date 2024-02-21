function Browse() {
    var p = document.getElementById("pathName").value.trim();
    var m = document.getElementById("merge").checked;
    var r = document.getElementById("router").value.trim();
 

    var dataToSend = {"shortname": r, "xpath": p, "merge": m};
   // waitingDialog.show();
    // send data
    

    $(function() {
        $.ajax({
            type: 'POST',
            url: "/searchxpath",
            data: JSON.stringify(dataToSend),
            contentType: "application/json",
            dataType: "json",
            success : function(json) {
              if (json.status == "OK") {
               // waitingDialog.hide();
                alertify.success("Xpath search endeed");
                const eventSource = new EventSource("/stream");

                eventSource.onmessage = function(event) {
                const data = JSON.parse(event.data);
                // Update the DOM with the received data
                const messageElement = document.getElementById("message");
                messageElement.innerHTML = `Message: ${data.msg}`;
                };
                eventSource.onerror = function(event) {
                  console.error("EventSource failed:", event);
                  eventSource.close();
              };
              }
              else {
               // waitingDialog.hide();
                alertify.alert("JSTO...", json.msg);
              }             
            },    
            error : function(xhr, ajaxOptions, thrownError) {  
                //waitingDialog.hide();      
                alertify.alert("JSTO...", "Unexpected error");
            }
        });
    });
}

