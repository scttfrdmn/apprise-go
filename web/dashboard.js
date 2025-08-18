// Apprise-Go Dashboard JavaScript

// Global state
let serverConfig = null;
let servicesData = null;

// Initialize dashboard
document.addEventListener('DOMContentLoaded', function() {
    initializeTabs();
    loadOverviewData();
    setupEventListeners();
});

// Tab management
function initializeTabs() {
    const tabs = document.querySelectorAll('.tab');
    const tabContents = document.querySelectorAll('.tab-content');

    tabs.forEach(tab => {
        tab.addEventListener('click', function() {
            const targetTab = this.getAttribute('data-tab');
            
            // Update active tab
            tabs.forEach(t => t.classList.remove('active'));
            this.classList.add('active');
            
            // Update active content
            tabContents.forEach(content => content.classList.remove('active'));
            document.getElementById(targetTab).classList.add('active');
            
            // Load content for specific tabs
            switch(targetTab) {
                case 'services':
                    loadServices();
                    break;
                case 'scheduler':
                    loadScheduler();
                    break;
                case 'config':
                    loadConfiguration();
                    break;
            }
        });
    });
}

// Event listeners
function setupEventListeners() {
    // Notification form
    document.getElementById('notification-form').addEventListener('submit', handleNotificationSubmit);
    
    // Service form
    document.getElementById('service-form').addEventListener('submit', handleServiceSubmit);
    
    // Job form
    document.getElementById('job-form').addEventListener('submit', handleJobSubmit);
}

// API helper functions
async function apiRequest(endpoint, options = {}) {
    try {
        const response = await fetch(endpoint, {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            },
            ...options
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        return await response.json();
    } catch (error) {
        console.error('API Request failed:', error);
        throw error;
    }
}

// Load overview data
async function loadOverviewData() {
    try {
        // Load health status
        const health = await apiRequest('/health');
        document.getElementById('server-status').textContent = '✅ Healthy';
        
        // Load version info
        const version = await apiRequest('/version');
        document.getElementById('api-version').textContent = version.data.version || 'Unknown';
        
        // Load services count
        const services = await apiRequest('/api/v1/services');
        document.getElementById('total-services').textContent = services.data.total || '0';
        
        // Check scheduler status
        if (health.data.scheduler) {
            document.getElementById('scheduler-status').textContent = '✅ Enabled';
        } else {
            document.getElementById('scheduler-status').textContent = '⚠️ Disabled';
        }
        
        // Load system info
        loadSystemInfo(version.data);
        
    } catch (error) {
        console.error('Failed to load overview data:', error);
        showAlert('error', 'Failed to load server information: ' + error.message);
    }
}

// Load system information
function loadSystemInfo(versionData) {
    const systemInfo = document.getElementById('system-info');
    systemInfo.innerHTML = `
        <div class="grid">
            <div>
                <strong>Version:</strong> ${versionData.version}<br>
                <strong>Upstream:</strong> ${versionData.upstream_version}<br>
                <strong>Port Revision:</strong> ${versionData.port_version}
            </div>
            <div>
                <strong>Go Version:</strong> ${versionData.go_version}<br>
                <strong>Platform:</strong> ${versionData.platform}<br>
                <strong>Architecture:</strong> ${versionData.architecture}
            </div>
        </div>
    `;
}

// Load services
async function loadServices() {
    const loadingEl = document.getElementById('services-loading');
    const gridEl = document.getElementById('services-grid');
    
    try {
        loadingEl.classList.remove('hidden');
        gridEl.classList.add('hidden');
        
        const response = await apiRequest('/api/v1/services');
        servicesData = response.data;
        
        renderServices(servicesData.services);
        
        loadingEl.classList.add('hidden');
        gridEl.classList.remove('hidden');
        
    } catch (error) {
        console.error('Failed to load services:', error);
        showAlert('error', 'Failed to load services: ' + error.message);
        loadingEl.classList.add('hidden');
    }
}

// Render services
function renderServices(services) {
    const grid = document.getElementById('services-grid');
    
    if (!services || services.length === 0) {
        grid.innerHTML = '<div class="alert alert-info">No services configured</div>';
        return;
    }
    
    grid.innerHTML = services.map(service => `
        <div class="service-card">
            <div class="service-name">${service.name || service.id}</div>
            <div class="service-info">
                ID: ${service.id}<br>
                Attachments: ${service.supports_attachments ? '✅' : '❌'}<br>
                Max Body: ${service.max_body_length > 0 ? service.max_body_length + ' chars' : 'Unlimited'}
            </div>
            <button class="btn btn-sm btn-primary" onclick="testServiceById('${service.id}')">Test</button>
        </div>
    `).join('');
}

// Load scheduler
async function loadScheduler() {
    const loadingEl = document.getElementById('scheduler-loading');
    const contentEl = document.getElementById('scheduler-content');
    const unavailableEl = document.getElementById('scheduler-unavailable');
    
    try {
        loadingEl.classList.remove('hidden');
        contentEl.classList.add('hidden');
        unavailableEl.classList.add('hidden');
        
        const response = await apiRequest('/api/v1/scheduler/jobs');
        
        renderJobs(response.data.jobs || []);
        
        loadingEl.classList.add('hidden');
        contentEl.classList.remove('hidden');
        
    } catch (error) {
        console.error('Failed to load scheduler:', error);
        
        if (error.message.includes('503')) {
            // Scheduler not available
            loadingEl.classList.add('hidden');
            unavailableEl.classList.remove('hidden');
        } else {
            showAlert('error', 'Failed to load scheduler: ' + error.message);
            loadingEl.classList.add('hidden');
        }
    }
}

// Render jobs
function renderJobs(jobs) {
    const jobsList = document.getElementById('jobs-list');
    
    if (!jobs || jobs.length === 0) {
        jobsList.innerHTML = '<div class="alert alert-info">No scheduled jobs found</div>';
        return;
    }
    
    jobsList.innerHTML = jobs.map(job => `
        <div class="card">
            <div style="display: flex; justify-content: between; align-items: center;">
                <div style="flex: 1;">
                    <h3>${job.name}</h3>
                    <p><strong>Schedule:</strong> ${job.cron_expression}</p>
                    <p><strong>Message:</strong> ${job.body}</p>
                    <span class="status-badge ${job.enabled ? 'status-healthy' : 'status-warning'}">
                        ${job.enabled ? 'Enabled' : 'Disabled'}
                    </span>
                </div>
                <div>
                    <button class="btn btn-sm btn-secondary" onclick="toggleJob(${job.id}, ${!job.enabled})">
                        ${job.enabled ? 'Disable' : 'Enable'}
                    </button>
                    <button class="btn btn-sm btn-danger" onclick="deleteJob(${job.id})">Delete</button>
                </div>
            </div>
        </div>
    `).join('');
}

// Load configuration
async function loadConfiguration() {
    const loadingEl = document.getElementById('config-loading');
    const contentEl = document.getElementById('config-content');
    
    try {
        loadingEl.classList.remove('hidden');
        contentEl.classList.add('hidden');
        
        const response = await apiRequest('/api/v1/config');
        serverConfig = response.data;
        
        renderConfiguration(serverConfig);
        
        loadingEl.classList.add('hidden');
        contentEl.classList.remove('hidden');
        
    } catch (error) {
        console.error('Failed to load configuration:', error);
        showAlert('error', 'Failed to load configuration: ' + error.message);
        loadingEl.classList.add('hidden');
    }
}

// Render configuration
function renderConfiguration(config) {
    const content = document.getElementById('config-content');
    
    content.innerHTML = `
        <div class="grid">
            <div>
                <h3>General</h3>
                <p><strong>Version:</strong> ${config.version}</p>
                <p><strong>Services:</strong> ${config.supported_services.length} supported</p>
            </div>
            <div>
                <h3>Features</h3>
                ${Object.entries(config.features).map(([key, value]) => 
                    `<p><strong>${key.replace(/_/g, ' ')}:</strong> ${value ? '✅' : '❌'}</p>`
                ).join('')}
            </div>
            <div>
                <h3>Limits</h3>
                ${Object.entries(config.limits).map(([key, value]) => 
                    `<p><strong>${key.replace(/_/g, ' ')}:</strong> ${value}</p>`
                ).join('')}
            </div>
            ${config.scheduler ? `
                <div>
                    <h3>Scheduler</h3>
                    <p><strong>Enabled:</strong> ${config.scheduler.enabled ? '✅' : '❌'}</p>
                    <p><strong>Database:</strong> ${config.scheduler.database_path || 'N/A'}</p>
                    <p><strong>Queue Size:</strong> ${config.scheduler.queue_size}</p>
                    <p><strong>Max Retries:</strong> ${config.scheduler.max_retries}</p>
                </div>
            ` : ''}
        </div>
    `;
}

// Form handlers
async function handleNotificationSubmit(e) {
    e.preventDefault();
    
    const title = document.getElementById('notification-title').value.trim();
    const body = document.getElementById('notification-body').value.trim();
    const type = document.getElementById('notification-type').value;
    const urls = document.getElementById('notification-urls').value.trim().split('\n').filter(url => url.trim());
    
    if (!body || urls.length === 0) {
        showAlert('error', 'Please provide both a message and at least one service URL');
        return;
    }
    
    const button = e.target.querySelector('button[type="submit"]');
    const btnText = button.querySelector('.btn-text');
    const loading = button.querySelector('.loading');
    
    try {
        btnText.textContent = 'Sending...';
        loading.classList.remove('hidden');
        button.disabled = true;
        
        const response = await apiRequest('/api/v1/notify', {
            method: 'POST',
            body: JSON.stringify({
                title: title || undefined,
                body: body,
                type: type,
                urls: urls
            })
        });
        
        if (response.success) {
            showAlert('success', `Notification sent successfully! ${response.data.successful}/${response.data.total} services succeeded.`);
            document.getElementById('notification-form').reset();
        } else {
            showAlert('error', 'Failed to send notification: ' + response.message);
        }
        
    } catch (error) {
        console.error('Failed to send notification:', error);
        showAlert('error', 'Failed to send notification: ' + error.message);
    } finally {
        btnText.textContent = 'Send Notification';
        loading.classList.add('hidden');
        button.disabled = false;
    }
}

async function handleServiceSubmit(e) {
    e.preventDefault();
    
    const url = document.getElementById('service-url').value.trim();
    
    if (!url) {
        showAlert('error', 'Please provide a service URL');
        return;
    }
    
    try {
        const response = await apiRequest('/api/v1/services', {
            method: 'POST',
            body: JSON.stringify({ url: url })
        });
        
        if (response.success) {
            showAlert('success', 'Service added successfully!');
            document.getElementById('service-form').reset();
            loadServices(); // Refresh services list
        } else {
            showAlert('error', 'Failed to add service: ' + response.message);
        }
        
    } catch (error) {
        console.error('Failed to add service:', error);
        showAlert('error', 'Failed to add service: ' + error.message);
    }
}

async function handleJobSubmit(e) {
    e.preventDefault();
    
    const name = document.getElementById('job-name').value.trim();
    const cronExpr = document.getElementById('job-cron').value.trim();
    const title = document.getElementById('job-title').value.trim();
    const body = document.getElementById('job-body').value.trim();
    const services = document.getElementById('job-services').value.trim().split('\n').filter(url => url.trim());
    
    if (!name || !cronExpr || !body || services.length === 0) {
        showAlert('error', 'Please fill in all required fields');
        return;
    }
    
    try {
        const response = await apiRequest('/api/v1/scheduler/jobs', {
            method: 'POST',
            body: JSON.stringify({
                name: name,
                cron_expr: cronExpr,
                title: title || undefined,
                body: body,
                services: services,
                enabled: true
            })
        });
        
        if (response.success) {
            showAlert('success', 'Scheduled job created successfully!');
            document.getElementById('job-form').reset();
            hideCreateJobForm();
            loadScheduler(); // Refresh jobs list
        } else {
            showAlert('error', 'Failed to create job: ' + response.message);
        }
        
    } catch (error) {
        console.error('Failed to create job:', error);
        showAlert('error', 'Failed to create job: ' + error.message);
    }
}

// Utility functions
function showAlert(type, message) {
    // Create alert element
    const alert = document.createElement('div');
    alert.className = `alert alert-${type}`;
    alert.innerHTML = message;
    
    // Insert at the top of container
    const container = document.querySelector('.container');
    container.insertBefore(alert, container.firstChild);
    
    // Auto remove after 5 seconds
    setTimeout(() => {
        if (alert.parentNode) {
            alert.parentNode.removeChild(alert);
        }
    }, 5000);
}

function showCreateJobForm() {
    document.getElementById('create-job-form').classList.remove('hidden');
}

function hideCreateJobForm() {
    document.getElementById('create-job-form').classList.add('hidden');
}

async function toggleJob(jobId, enable) {
    try {
        const action = enable ? 'enable' : 'disable';
        const response = await apiRequest(`/api/v1/scheduler/jobs/${jobId}/${action}`, {
            method: 'POST'
        });
        
        if (response.success) {
            showAlert('success', `Job ${action}d successfully!`);
            loadScheduler(); // Refresh jobs list
        } else {
            showAlert('error', `Failed to ${action} job: ` + response.message);
        }
    } catch (error) {
        console.error(`Failed to toggle job:`, error);
        showAlert('error', `Failed to toggle job: ` + error.message);
    }
}

async function deleteJob(jobId) {
    if (!confirm('Are you sure you want to delete this job?')) {
        return;
    }
    
    try {
        const response = await apiRequest(`/api/v1/scheduler/jobs/${jobId}`, {
            method: 'DELETE'
        });
        
        if (response.success) {
            showAlert('success', 'Job deleted successfully!');
            loadScheduler(); // Refresh jobs list
        } else {
            showAlert('error', 'Failed to delete job: ' + response.message);
        }
    } catch (error) {
        console.error('Failed to delete job:', error);
        showAlert('error', 'Failed to delete job: ' + error.message);
    }
}

async function testService() {
    const url = document.getElementById('service-url').value.trim();
    
    if (!url) {
        showAlert('error', 'Please provide a service URL to test');
        return;
    }
    
    // Extract service ID from URL for testing
    const serviceId = url.split('://')[0];
    
    try {
        const response = await apiRequest(`/api/v1/services/${serviceId}/test`, {
            method: 'POST',
            body: JSON.stringify({
                url: url,
                title: 'Apprise-Go Test',
                message: 'This is a test notification from the Apprise-Go dashboard'
            })
        });
        
        if (response.success) {
            showAlert('success', 'Service test successful!');
        } else {
            showAlert('error', 'Service test failed: ' + response.message);
        }
    } catch (error) {
        console.error('Service test failed:', error);
        showAlert('error', 'Service test failed: ' + error.message);
    }
}

async function testServiceById(serviceId) {
    try {
        const response = await apiRequest(`/api/v1/services/${serviceId}/test`, {
            method: 'POST',
            body: JSON.stringify({
                url: `${serviceId}://test`,
                title: 'Apprise-Go Test',
                message: 'This is a test notification from the Apprise-Go dashboard'
            })
        });
        
        if (response.success) {
            showAlert('success', `${serviceId} service test successful!`);
        } else {
            showAlert('error', `${serviceId} service test failed: ` + response.message);
        }
    } catch (error) {
        console.error('Service test failed:', error);
        showAlert('error', `${serviceId} service test failed: ` + error.message);
    }
}