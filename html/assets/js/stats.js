const modal = document.getElementById("modalcore")
modal.style.scrollBehavior = 'smooth';

$(document).ready(function () {
    const $gaugeContainer = $('#container-gauges');
    const refreshInterval = 30000; // 5 seconds

    // Function to fetch stats and update gauges
    function fetchStats() {
        $.ajax({
            url: '/containerstats',
            method: 'GET',
            timeout: 10000,
            success: function (response) {
                if (response.status === 'OK') {

                    const data = response.data;

                    // Clear existing gauges
                    $gaugeContainer.empty();

                    // Create gauges for each container
                    Object.keys(data).forEach(container => {
                        const cpu = (data[container].cpu || 0).toFixed(2); // Two decimals for CPU
                        const mem = (data[container].mem || 0).toFixed(2); // Two decimals for Memory

                        // Create a container for the gauges
                        const gaugeCard = $(`
                            <div class="col-md-4 mb-4">
                                <div class="card shadow-sm">
                                    <div class="card-body text-center">
                                        <h5 class="card-title">${container}</h5>
                                        <div id="gauge-cpu-${container}" class="gauge mb-3"></div>
                                      
                                        <div id="gauge-mem-${container}" class="gauge mt-3"></div>
                                         <button style="margin:10px;" class="btn btn-success" onclick="getLogs('${container}')">
                                            <i class="fa fa-history"></i> Last logs
                                        </button>
                                       
                                    </div>
                                </div>
                            </div>
                        `);

                        $gaugeContainer.append(gaugeCard);

                        // Initialize gauges
                        new JustGage({
                            id: `gauge-cpu-${container}`,
                            value: parseFloat(cpu),
                            min: 0,
                            max: 100,
                            title: "CPU",
                            levelColors: ["#28a745", "#ffc107", "#dc3545"],
                            label: `${cpu}%`,
                            decimals: true
                        });

                        new JustGage({
                            id: `gauge-mem-${container}`,
                            value: parseFloat(mem),
                            min: 0,
                            max: 100,
                            title: "Memory",
                            levelColors: ["#007bff", "#17a2b8", "#6f42c1"],
                            label: `${mem}%`,
                            decimals: true
                        });
                    });
                } else {
                    alertify.error('Failed to fetch stats: Invalid status');
                }
            },
            error: function () {
                alertify.error('Error fetching stats from /stats');
            }
        });
    }


    // Fetch stats immediately and set up periodic refresh
    fetchStats();
    setInterval(fetchStats, refreshInterval);

});

// Function to append new content
function appendContent(text) {
    var newElement = document.createElement('div');
    newElement.textContent = text;
    modal.appendChild(newElement);
}

// Function to scroll to the bottom with smooth scrolling
function scrollToBottom() {
    modal.scrollTop = modal.scrollHeight;
  }

function getLogs(c) {
    waitingDialog.show();
    $(function () {
        $.ajax({
            method: 'GET',
            url: "/containerlogs?name=" + encodeURIComponent(c),
            success: function (json) {
                if (json.status == "OK") {
                    modal.innerHTML = '';
                    for (let i = 0; i < json.data.length; i++) {
                        appendContent(json.data[i])
                    }
                    scrollToBottom()
                    $('#logs').modal('show');
                    waitingDialog.hide();
                    alertify.success("Logs of " + c + " container has been successfulfy retrieved")
                } else {
                    waitingDialog.hide();
                    alertify.alert("JSTO...", json.msg);
                }
            },
            error: function (xhr, ajaxOptions, thrownError) {
                waitingDialog.hide();
                alertify.alert("JSTO...", "Unexpected error");
            }
        });
    });
}

function closeModal() {
    const modal = document.getElementById('config');
    $('#logs').modal('hide');
    modal.style.display = 'none';  // Hide the modal
    const backdrop = document.querySelector('.modal-backdrop');
    if (backdrop) backdrop.remove(); // Remove the backdrop element
  }