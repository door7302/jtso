function addAsso() {
    alertify.confirm("Are you sure you want to empty the DB? All data will be lost.", function (e) { 
        if (e) {
          $(function() {
            $.ajax({
                type: 'POST',
                url: "/influxmgt",
                data: JSON.stringify({"action": "emptydb", "data": ""}),
                contentType: "application/json",
                dataType: "json",
                success : function(result) {
                  if (result["status"] == "ok") {
                      alertify.success('The JTS DB has been successfully empty')
                  }
                  else {
                    alertify.alert("JTSO...", json.msg);
                  }
                },    
                error : function(xhr, ajaxOptions, thrownError) {
                    alertify.alert("JSTO...", "Unexpected error...");
                }
            });
        });   
      }
    }).setHeader('JSTO...'); 
}