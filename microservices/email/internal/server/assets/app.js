// Template Manager Application
document.addEventListener('DOMContentLoaded', function() {
    // Initialize the template manager
    initTemplateManager();
    
    // Set up WebSocket connection for live updates
    setupWebSocket();
});

// Global variables
let templateEditor = null;
let sampleDataEditor = null;
let currentTemplate = null;
let currentSampleData = null;
let autoRefreshEnabled = true;
let socket = null;

// Initialize the template manager
function initTemplateManager() {
    // Setup the Monaco editor
    setupEditors();
    
    // Load templates
    loadTemplates();

    // Set up event listeners
    document.getElementById('save-button').addEventListener('click', saveTemplate);
    document.getElementById('save-sample-button').addEventListener('click', saveSampleData);
    document.getElementById('refresh-preview').addEventListener('click', refreshPreview);
    document.getElementById('theme-selector').addEventListener('change', changeTheme);
}

// Set up WebSocket connection for live updates
function setupWebSocket() {
    // Create WebSocket connection
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    socket = new WebSocket(wsUrl);
    
    // Connection opened
    socket.addEventListener('open', (event) => {
        console.log('Connected to WebSocket server');
    });
    
    // Listen for messages
    socket.addEventListener('message', (event) => {
        try {
            const data = JSON.parse(event.data);
            
            if (data.type === 'template_updated') {
                // If the current template is updated by another client
                if (data.templateName === currentTemplate) {
                    showToast('info', `Template "${data.templateName}" was updated by another user`);
                    
                    // Reload the template content
                    fetch(`/api/templates/${currentTemplate}`)
                        .then(response => response.text())
                        .then(content => {
                            // Only update if not currently editing
                            if (document.getElementById('save-button').disabled) {
                                templateEditor.setValue(content);
                                refreshPreview();
                            }
                        })
                        .catch(error => {
                            console.error('Failed to reload template:', error);
                        });
                }
            } else if (data.type === 'sample_updated') {
                // If the current sample data is updated by another client
                if (data.sampleName === currentTemplate) {
                    showToast('info', `Sample data for "${data.sampleName}" was updated by another user`);
                    
                    // Reload the sample data
                    if (document.getElementById('samples-section').classList.contains('active')) {
                        loadSampleData(currentTemplate, true);
                    } else {
                        // Just refresh the preview with new data
                        refreshPreview();
                    }
                }
            }
        } catch (error) {
            console.error('Error processing WebSocket message:', error);
        }
    });
    
    // Connection closed
    socket.addEventListener('close', (event) => {
        console.log('Disconnected from WebSocket server');
        
        // Try to reconnect after 3 seconds
        setTimeout(setupWebSocket, 3000);
    });
    
    // Connection error
    socket.addEventListener('error', (event) => {
        console.error('WebSocket error:', event);
    });
}

// Tab switching
function showTab(tabId) {
    // Hide all tab contents
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.add('hidden');
        content.classList.remove('active');
    });
    
    // Show the selected tab content
    document.getElementById(tabId + '-section').classList.remove('hidden');
    document.getElementById(tabId + '-section').classList.add('active');
    
    // Update tab buttons
    document.querySelectorAll('.tab-button').forEach(button => {
        button.classList.remove('active', 'border-zinc-500', 'text-zinc-200');
        button.classList.add('border-transparent', 'text-zinc-500');
    });
    
    document.getElementById(tabId + '-tab').classList.add('active', 'border-zinc-500', 'text-zinc-200');
    document.getElementById(tabId + '-tab').classList.remove('border-transparent', 'text-gray-500');

    // Load sample data when switching to the sample data tab
    if (tabId === 'samples' && currentTemplate) {
        loadSampleData(currentTemplate);
    }
}

// Setup Monaco editors
function setupEditors() {
    require.config({ paths: { 'vs': 'https://unpkg.com/monaco-editor@0.45.0/min/vs' }});
    require(['vs/editor/editor.main'], function() {
        // Define custom themes
        defineMonacoThemes();
        
        // Setup HTML template editor
        templateEditor = monaco.editor.create(document.getElementById('editor'), {
            value: '',
            language: 'html',
            theme: 'brilliance-black',
            automaticLayout: true,
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            lineNumbers: 'on',
            glyphMargin: false,
            folding: true,
            lineDecorationsWidth: 10,
            lineNumbersMinChars: 3
        });
        
        templateEditor.onDidChangeModelContent(debounce(function() {
            // Enable save button when content changes
            document.getElementById('save-button').disabled = false;
            
            // Auto-refresh preview with changes
            if (autoRefreshEnabled) {
                refreshPreview();
            }
        }, 500));
        
        // Setup JSON sample data editor
        sampleDataEditor = monaco.editor.create(document.getElementById('sample-editor'), {
            value: '',
            language: 'json',
            theme: 'brilliance-black',
            automaticLayout: true,
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            formatOnPaste: true,
            formatOnType: true
        });
        
        sampleDataEditor.onDidChangeModelContent(debounce(function() {
            // Enable save button when content changes
            document.getElementById('save-sample-button').disabled = false;
        }, 500));
        
        // Update the theme selector to show the current theme
        document.getElementById('theme-selector').value = 'brilliance-black';
    });
}

// Change editor theme
function changeTheme(event) {
    const theme = event.target.value;
    monaco.editor.setTheme(theme);
    
    // Save the user's theme preference in localStorage
    localStorage.setItem('editorTheme', theme);
}

// Define custom Monaco themes
function defineMonacoThemes() {
    // Load Brilliance Black theme
    fetch('https://cdn.jsdelivr.net/npm/monaco-themes@0.4.4/themes/Brilliance%20Black.json')
        .then(data => data.json())
        .then(data => {
            monaco.editor.defineTheme('brilliance-black', data);
            monaco.editor.setTheme('brilliance-black');
        });
    
    // Load other themes for the selector
    const themes = [
        { id: 'dracula', url: 'https://cdn.jsdelivr.net/npm/monaco-themes@0.4.4/themes/Dracula.json' },
        { id: 'monokai', url: 'https://cdn.jsdelivr.net/npm/monaco-themes@0.4.4/themes/Monokai.json' },
        { id: 'github', url: 'https://cdn.jsdelivr.net/npm/monaco-themes@0.4.4/themes/GitHub.json' },
        { id: 'solarized-dark', url: 'https://cdn.jsdelivr.net/npm/monaco-themes@0.4.4/themes/Solarized-dark.json' },
        { id: 'nord', url: 'https://cdn.jsdelivr.net/npm/monaco-themes@0.4.4/themes/Nord.json' }
    ];
    
    themes.forEach(theme => {
        fetch(theme.url)
            .then(data => data.json())
            .then(data => {
                monaco.editor.defineTheme(theme.id, data);
            });
    });
    
    // Load saved theme preference
    const savedTheme = localStorage.getItem('editorTheme');
    if (savedTheme) {
        // We'll apply this once the editors are created
        setTimeout(() => {
            monaco.editor.setTheme(savedTheme);
            document.getElementById('theme-selector').value = savedTheme;
        }, 100);
    }
}

// Load templates from the server
function loadTemplates() {
    fetch('/api/templates')
        .then(response => response.json())
        .then(templates => {
            const templateList = document.getElementById('template-list');
            templateList.innerHTML = '';
            
            templates.forEach((template, index) => {
                const li = document.createElement('li');
                const button = document.createElement('button');
                button.textContent = template;
                button.className = 'w-full text-left px-3 py-2 text-sm font-medium rounded hover:bg-zinc-800 hover:text-zinc-200 transition-colors';
                button.addEventListener('click', () => selectTemplate(template));
                li.appendChild(button);
                templateList.appendChild(li);
                
                // Select the first template by default
                if (index === 0) {
                    setTimeout(() => selectTemplate(template), 100);
                }
            });
        })
        .catch(error => {
            showToast('error', 'Failed to load templates: ' + error.message);
        });
}

// Select a template for editing
function selectTemplate(templateName) {
    currentTemplate = templateName;
    
    // Update template list selection
    document.querySelectorAll('#template-list button').forEach(button => {
        if (button.textContent === templateName) {
            button.classList.add('bg-zinc-800', 'text-zinc-200');
            button.classList.remove('text-zinc-500');
        } else {
            button.classList.remove('bg-zinc-800', 'text-zinc-200');
            button.classList.add('text-zinc-500');
        }
    });
    
    // Update template name display
    document.getElementById('current-template-name').textContent = 'Editing: ' + templateName;
    
    // Load template content
    fetch(`/api/templates/${templateName}`)
        .then(response => response.text())
        .then(content => {
            templateEditor.setValue(content);
            document.getElementById('save-button').disabled = true;
            
            // Refresh preview for the new template
            refreshPreview();
        })
        .catch(error => {
            showToast('error', 'Failed to load template: ' + error.message);
        });
        
    // Load sample data if on the sample data tab
    if (document.getElementById('samples-section').classList.contains('active')) {
        loadSampleData(templateName);
    } else {
        // Load sample data in the background for preview
        loadSampleData(templateName, true);
    }
}

// Load sample data for a template
function loadSampleData(templateName, skipDisable) {
    fetch(`/api/samples/${templateName}`)
        .then(response => {
            if (!response.ok) {
                // If template-specific sample doesn't exist, load the default
                return fetch('/api/samples/default');
            }
            return response;
        })
        .then(response => response.json())
        .then(data => {
            currentSampleData = data;
            const formattedJson = JSON.stringify(data, null, 2);
            sampleDataEditor.setValue(formattedJson);
            if (!skipDisable) {
                document.getElementById('save-sample-button').disabled = true;
            }
        })
        .catch(error => {
            showToast('error', 'Failed to load sample data: ' + error.message);
        });
}

// Save template changes
function saveTemplate() {
    if (!currentTemplate) return;
    
    const content = templateEditor.getValue();
    
    fetch(`/api/templates/${currentTemplate}`, {
        method: 'PUT',
        headers: {
            'Content-Type': 'text/plain'
        },
        body: content
    })
    .then(response => {
        if (response.ok) {
            showToast('success', 'Template saved successfully');
            document.getElementById('save-button').disabled = true;
            refreshPreview();
        } else {
            throw new Error('Failed to save template');
        }
    })
    .catch(error => {
        showToast('error', 'Failed to save template: ' + error.message);
    });
}

// Save sample data changes
function saveSampleData() {
    if (!currentTemplate) return;
    
    try {
        const content = sampleDataEditor.getValue();
        const jsonData = JSON.parse(content); // Validate JSON
        
        fetch(`/api/samples/${currentTemplate}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json'
            },
            body: content
        })
        .then(response => {
            if (response.ok) {
                showToast('success', 'Sample data saved successfully');
                document.getElementById('save-sample-button').disabled = true;
                currentSampleData = jsonData;
                
                // Refresh preview with new sample data
                refreshPreview();
            } else {
                throw new Error('Failed to save sample data');
            }
        })
        .catch(error => {
            showToast('error', 'Failed to save sample data: ' + error.message);
        });
    } catch (e) {
        showToast('error', 'Invalid JSON: ' + e.message);
    }
}

// Refresh the template preview
function refreshPreview() {
    if (!currentTemplate) return;
    
    const content = templateEditor.getValue();
    
    fetch(`/api/templates/preview/${currentTemplate}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'text/plain'
        },
        body: content
    })
    .then(response => response.text())
    .then(html => {
        const iframe = document.getElementById('preview-frame');
        iframe.srcdoc = html;
    })
    .catch(error => {
        showToast('error', 'Failed to preview template: ' + error.message);
    });
}

// Show a toast notification
function showToast(type, message) {
    const toast = document.getElementById('toast');
    const toastIcon = document.getElementById('toast-icon');
    const toastMessage = document.getElementById('toast-message');
    
    // Set icon and color based on type
    if (type === 'success') {
        toast.className = 'fixed top-4 right-4 p-4 bg-green-100 text-green-800 rounded-lg shadow-lg z-50';
        toastIcon.innerHTML = '<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg>';
    } else if (type === 'info') {
        toast.className = 'fixed top-4 right-4 p-4 bg-blue-100 text-blue-800 rounded-lg shadow-lg z-50';
        toastIcon.innerHTML = '<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>';
    } else {
        toast.className = 'fixed top-4 right-4 p-4 bg-red-100 text-red-800 rounded-lg shadow-lg z-50';
        toastIcon.innerHTML = '<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>';
    }
    
    // Set message
    toastMessage.textContent = message;
    
    // Show toast
    toast.classList.remove('hidden');
    
    // Hide toast after 3 seconds
    setTimeout(() => {
        toast.classList.add('hidden');
    }, 3000);
}

// Utility function for debouncing
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
} 