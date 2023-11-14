function saveCred() {
    var u = document.getElementById("Username").value.trim();
    var p = document.getElementById("Password").value;
    var u2 = document.getElementById("Username2").value.trim();
    var p2 = document.getElementById("Password2").value;
    var t = document.getElementById("Usetls").checked;
    var s = document.getElementById("Skipverify").checked;
    var c = document.getElementById("Clienttls").checked;
    
    var tls = "no"
    if (t) {
      tls = "yes"
    }
    var skip = "no"
    if (s) {
      skip = "yes"
    }
    var client = "no"
    if (c) {
      client = "yes"
    }

    var dataToSend = {"netuser": u, "netpwd": p, "gnmiuser": u2, "gnmipwd": p2, "usetls": tls, "skipverify": skip, "clienttls": client};
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
      
                alertify.success("Crendentials have been successfulfy updated");
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
