function saveSettings() {
  var u = document.getElementById("Username").value.trim();
  var p = document.getElementById("Password").value;
  var u2 = document.getElementById("Username2").value.trim();
  var p2 = document.getElementById("Password2").value;
  var t = document.getElementById("Usetls").checked;
  var s = document.getElementById("Skipverify").checked;
  var c = document.getElementById("Clienttls").checked;
  var kEnabled = document.getElementById("UseKafka").checked;
  var kBrokers = document.getElementById("KafkaBrokers").value.trim();
  var kTopic = document.getElementById("KafkaTopic").value.trim();
  var kVersion = document.getElementById("KafkaVersion").value.trim();
  var kFormat = document.getElementById("KafkaFormat").value.trim().toLowerCase();
  var kCompression = document.getElementById("KafkaCompression").value.trim().toLowerCase();
  var kMessageSize = document.getElementById("KafkaMessageSize").value.trim();  

  var SupportedFormats = ["json", "influx"];
  var SupportedCompressions = ["none", "gzip", "snappy", "lz4", "zstd"];
  var DictKafkaCodec = {
    "none": 0,
    "gzip": 1,
    "snappy": 2,
    "lz4": 3,
    "zstd": 4
  }
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
  var enabled = 0;
  if (kEnabled) {
    enabled = 1;
  }

  if (kEnabled && (kFormat == "" || SupportedFormats.indexOf(kFormat) == -1)) {
    alertify.alert("JSTO...", "Invalid Kafka format");
    return;
  }
  if (kEnabled && (kCompression == "" || SupportedCompressions.indexOf(kCompression) == -1)) {
    alertify.alert("JSTO...", "Invalid Kafka compression");
    return;
  } 
  
  if (kMessageSize == "" || isNaN(kMessageSize) || parseInt(kMessageSize) <= 0) {
    alertify.alert("JSTO...", "Invalid Kafka message size");
    return;
  }

  if (kEnabled && kBrokers == "") {
    alertify.alert("JSTO...", "Kafka brokers cannot be empty if Kafka is enabled");
    return;
  }

  if (kEnabled && kTopic == "") {
    alertify.alert("JSTO...", "Kafka topic cannot be empty if Kafka is enabled");
    return;
  } 

  if (kEnabled && kVersion == "") {
    alertify.alert("JSTO...", "Kafka version cannot be empty if Kafka is enabled");
    return;
  } 

  if (u == "" || p == "" || u2 == "" || p2 == "") {
    alertify.alert("JSTO...", "Username and password fields cannot be empty");
    return;
  }

  var dataToSend = {
    "netuser": u,
    "netpwd": p,
    "gnmiuser": u2,
    "gnmipwd": p2,
    "usetls": tls,
    "skipverify": skip,
    "clienttls": client,
    "kafkaenabled": enabled,
    "kafkabrokers": kBrokers,
    "kafkatopic": kTopic,
    "kafkaversion": kVersion,
    "kafkaformat": kFormat,
    "kafkacompression": DictKafkaCodec[kCompression],
    "kafkamessagesize": kMessageSize
  };
  // send data
  $(function () {
    $.ajax({
      type: 'POST',
      url: "/updatesettings",
      data: JSON.stringify(dataToSend),
      contentType: "application/json",
      dataType: "json",
      success: function (json) {
        if (json.status == "OK") {

          alertify.success("Settings have been successfully updated");
        } else {
          alertify.alert("JSTO...", json.msg);
        }
      },
      error: function (xhr, ajaxOptions, thrownError) {
        alertify.alert("JSTO...", "Unexpected error");
      }
    });
  });
}