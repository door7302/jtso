function updateDoc() {
    var p = document.getElementById("profiles").value.trim();
    var img = document.getElementById("profileImg");
    var desc = document.getElementById("profileDesc");
    var tele = document.getElementById("profileTele");
    var graf = document.getElementById("profileGraf");
    var kapa = document.getElementById("profileKapa");


    var dataToSend = {"profile": p};
    // send data
    $(function() {
        $.ajax({
            type: 'POST',
            url: "/updatedoc",
            data: JSON.stringify(dataToSend),
            contentType: "application/json",
            dataType: "json",
            success : function(json) {
              if (json.status == "OK") {
                img.setAttribute('src', json.img);
                desc.innerHTML= json.desc;
                tele.innerHTML= json.tele;
                graf.innerHTML= json.graf;
                kapa.innerHTML= json.kapa;

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
