function saveCred() {
    var u = document.getElementById("Username").value;
    var p = document.getElementById("Password").value;
    var u2 = document.getElementById("Username2").value;
    var p2 = document.getElementById("Password2").value;
    var t = document.getElementById("Usetls").checked;
    
    var tls = "no"
    if (t) {
      tls = "yes"
    }

    var dataToSend = {"netuser": u, "netpwd": p, "gnmiuser": u2, "gnmipwd": p2, "usetls": tls};
    // send data
    $(function() {
        $.ajax({
            type: 'POST',
            url: "/updatecred",
            data: JSON.stringify(dataToSend),
            contentType: "application/json",
            dataType: "json",
            success : function(json) {
              if (json.status == "OK") {
      
                alertify.success("Crendentials have been successfulfy added");
              }
              else {
                alertify.alert("JSTO...", json.msg);
              }             
            },    
            error : function(xhr, ajaxOptions, thrownError) {        
                alertify.alert("JSTO...", "Unexpected error");
            }
        });
    });
}
