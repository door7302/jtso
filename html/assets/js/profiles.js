function addAsso() {
  var r = document.getElementById("router").value.trim();
  var all_selected = document.getElementById('profiles').options
  var selected = [];
  var raw_selected = ""
  for (var option of all_selected)
  {
      if (option.selected) {
          selected.push(option.value);
          raw_selected = raw_selected + option.value + " "
      }
  }
 
  
  var dataToSend = {"shortname": r, "profiles": selected};
  // send data
  $(function() {
      $.ajax({
          type: 'POST',
          url: "/addprofile",
          data: JSON.stringify(dataToSend),
          contentType: "application/json",
          dataType: "json",
          success : function(json) {
            if (json.status == "OK") {
              var row = $("<tr><td>"+r+"</td><td>"+raw_selected+'</td><td class="d-xxl-flex justify-content-xxl-center"><button onclick="removeAsso("'+r+'", this)" class="btn btn-danger" style="margin-left: 5px;" type="submit"><i class="fa fa-trash" style="font-size: 15px;"></i></button></td></tr>')
              $("#ListRtrs").append(row);
              alertify.success("Profile(s) have been successfulfy added to router "+r)
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

function removeAsso(name, td) { 
var dataToSend = {"shortname": name};
// send data
$(function() {
  $.ajax({
      type: 'POST',
      url: "/delprofile",
      data: JSON.stringify(dataToSend),
      contentType: "application/json",
      dataType: "json",
      success : function(json) {
        if (json.status == "OK") {
          $(td).closest("tr").remove();
          alertify.success("Router "+name+" has been successfulfy removed")
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