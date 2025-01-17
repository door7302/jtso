$(document).ready(function () {
    const $gaugeContainer = $('#container-gauges');
    const refreshInterval = 30000; // 5 seconds

    // Function to fetch stats and update gauges
    function fetchStats() {
        $.ajax({
            url: '/containerstats',
            method: 'GET',
            success: function (response) {
                if (response.status === 'OK') {
                    const data = response.Data;

                    // Clear existing gauges
                    $gaugeContainer.empty();

                    // Create gauges for each container
                    Object.keys(data).forEach(container => {
                        const cpu = data[container].cpu;
                        const mem = data[container].mem;

                        // Create a container for the gauges
                        const gaugeCard = $(`
                            <div class="col-md-6 mb-4">
                                <div class="card shadow-sm">
                                    <div class="card-body text-center">
                                        <h5 class="card-title">${container}</h5>
                                        <div id="gauge-cpu-${container}" class="gauge mb-3"></div>
                                        <p class="mb-0 text-muted">CPU Utilization</p>
                                        <div id="gauge-mem-${container}" class="gauge mt-3"></div>
                                        <p class="mb-0 text-muted">Memory Utilization</p>
                                    </div>
                                </div>
                            </div>
                        `);

                        $gaugeContainer.append(gaugeCard);

                        // Initialize gauges
                        new JustGage({
                            id: `gauge-cpu-${container}`,
                            value: cpu,
                            min: 0,
                            max: 100,
                            title: "CPU",
                            levelColors: ["#28a745", "#ffc107", "#dc3545"],
                        });

                        new JustGage({
                            id: `gauge-mem-${container}`,
                            value: mem,
                            min: 0,
                            max: 100,
                            title: "Memory",
                            levelColors: ["#007bff", "#17a2b8", "#6f42c1"],
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

    // Dark mode toggle
    $('#darkModeSwitch').change(function () {
        const theme = $(this).is(':checked') ? 'dark' : 'light';
        $('html').attr('data-bs-theme', theme);
    });
});