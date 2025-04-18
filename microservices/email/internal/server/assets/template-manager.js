function templateManager() {
    return {
        templates: [],
        selectedTemplate: null,
        templateContent: '',
        showPreview: false,
        livePreview: false,
        ws: null,
        previewTimeout: null,
        
        init() {
            this.fetchTemplates();
            this.connectWebSocket();
        },
        
        connectWebSocket() {
            // Create WebSocket connection
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = protocol + '//' + window.location.host + '/ws';
            
            this.ws = new WebSocket(wsUrl);
            
            // Connection opened
            this.ws.addEventListener('open', (event) => {
                console.log('Connected to WebSocket server');
            });
            
            // Listen for messages
            this.ws.addEventListener('message', (event) => {
                const data = JSON.parse(event.data);
                if (data.type === 'template_updated' && data.templateName === this.selectedTemplate) {
                    // Reload the template content and preview
                    this.fetchTemplateContent(this.selectedTemplate);
                }
            });
            
            // Connection closed
            this.ws.addEventListener('close', (event) => {
                console.log('Disconnected from WebSocket server');
                // Try to reconnect after a delay
                setTimeout(() => this.connectWebSocket(), 2000);
            });
            
            // Connection error
            this.ws.addEventListener('error', (event) => {
                console.error('WebSocket error:', event);
            });
        },
        
        fetchTemplates() {
            fetch('/api/templates')
                .then(response => response.json())
                .then(data => {
                    this.templates = data
                })
                .catch(error => {
                    console.error('Error fetching templates:', error)
                    alert('Failed to load templates')
                })
        },
        
        selectTemplate(template) {
            this.selectedTemplate = template
            this.showPreview = false
            this.fetchTemplateContent(template)
        },
        
        fetchTemplateContent(template) {
            fetch('/api/templates/' + template)
                .then(response => response.text())
                .then(data => {
                    this.templateContent = data
                    if (this.livePreview) {
                        this.previewTemplate()
                    }
                })
                .catch(error => {
                    console.error('Error fetching template content:', error)
                    alert('Failed to load template content')
                })
        },
        
        saveTemplate() {
            fetch('/api/templates/' + this.selectedTemplate, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'text/plain'
                },
                body: this.templateContent
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to save template')
                }
                alert('Template saved successfully')
                if (this.livePreview) {
                    this.previewTemplate()
                }
            })
            .catch(error => {
                console.error('Error saving template:', error)
                alert('Failed to save template')
            })
        },
        
        onTemplateChange() {
            if (!this.livePreview) return;
            
            // Debounce preview updates to avoid too many requests
            if (this.previewTimeout) {
                clearTimeout(this.previewTimeout);
            }
            
            this.previewTimeout = setTimeout(() => {
                this.previewTemplateContent();
            }, 500);
        },
        
        toggleLivePreview() {
            this.livePreview = !this.livePreview;
            if (this.livePreview) {
                this.previewTemplate();
            }
        },
        
        previewTemplate() {
            this.showPreview = true;
            this.previewTemplateContent();
        },
        
        previewTemplateContent() {
            fetch('/api/templates/preview/' + this.selectedTemplate, {
                method: 'POST',
                headers: {
                    'Content-Type': 'text/plain'
                },
                body: this.templateContent
            })
            .then(response => response.text())
            .then(html => {
                setTimeout(() => {
                    const doc = this.$refs.preview.contentDocument;
                    doc.open();
                    doc.write(html);
                    doc.close();
                }, 100);
            })
            .catch(error => {
                console.error('Error previewing template:', error);
                
                // Show error in preview
                const doc = this.$refs.preview.contentDocument;
                doc.open();
                doc.write('<div style="color: red; padding: 20px;">Error previewing template</div>');
                doc.close();
            });
        }
    }
} 