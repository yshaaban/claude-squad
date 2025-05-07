// Terminal streaming test script
// This can be used in the browser console to verify WebSocket functionality

// Configuration
const config = {
    instance: null, // Will be populated with first instance
    useFallback: true,
    logLevel: 'debug', // 'debug', 'info', 'warn', 'error'
    retryOnFailure: true,
    maxRetries: 3
};

// Log levels
const LOG_LEVELS = {
    debug: 0,
    info: 1,
    warn: 2,
    error: 3
};

// Initialize logger
const logger = {
    debug: function(message, data) {
        if (LOG_LEVELS[config.logLevel] <= LOG_LEVELS.debug) {
            console.debug(`[TEST:${new Date().toISOString()}] ${message}`, data || '');
        }
    },
    info: function(message, data) {
        if (LOG_LEVELS[config.logLevel] <= LOG_LEVELS.info) {
            console.info(`[TEST:${new Date().toISOString()}] ${message}`, data || '');
        }
    },
    warn: function(message, data) {
        if (LOG_LEVELS[config.logLevel] <= LOG_LEVELS.warn) {
            console.warn(`[TEST:${new Date().toISOString()}] ${message}`, data || '');
        }
    },
    error: function(message, data) {
        if (LOG_LEVELS[config.logLevel] <= LOG_LEVELS.error) {
            console.error(`[TEST:${new Date().toISOString()}] ${message}`, data || '');
        }
    }
};

// Get instance information
async function getInstances() {
    try {
        logger.info('Fetching instances');
        const response = await fetch('/api/instances');
        if (!response.ok) {
            throw new Error(`HTTP error: ${response.status}`);
        }
        const data = await response.json();
        logger.info(`Found ${data.instances.length} instances`, data.instances);
        
        if (data.instances.length > 0) {
            config.instance = data.instances[0].title;
            logger.info(`Selected instance: ${config.instance}`);
            return data.instances;
        } else {
            logger.error('No instances found');
            return [];
        }
    } catch (error) {
        logger.error('Error fetching instances:', error);
        return [];
    }
}

// Create a fallback terminal element for testing
function createTestTerminal() {
    logger.info('Creating test terminal element');
    
    // Check if it already exists
    let testTerminal = document.getElementById('test-terminal');
    if (testTerminal) {
        logger.info('Test terminal already exists, clearing');
        testTerminal.innerHTML = '';
        return testTerminal;
    }
    
    // Create container
    const container = document.createElement('div');
    container.id = 'test-terminal';
    container.style.position = 'fixed';
    container.style.top = '20px';
    container.style.right = '20px';
    container.style.width = '600px';
    container.style.height = '400px';
    container.style.backgroundColor = '#000';
    container.style.color = '#fff';
    container.style.fontFamily = 'monospace';
    container.style.padding = '10px';
    container.style.overflow = 'auto';
    container.style.zIndex = '9999';
    container.style.border = '2px solid #333';
    container.style.borderRadius = '5px';
    
    // Add header
    const header = document.createElement('div');
    header.style.padding = '5px';
    header.style.backgroundColor = '#333';
    header.style.marginBottom = '10px';
    header.style.borderRadius = '3px';
    header.textContent = 'Terminal Test Console';
    container.appendChild(header);
    
    // Add content area
    const content = document.createElement('div');
    content.id = 'test-terminal-content';
    content.style.whiteSpace = 'pre-wrap';
    content.style.height = 'calc(100% - 50px)';
    content.style.overflow = 'auto';
    content.textContent = 'Terminal test initialized. Waiting for content...\n\n';
    container.appendChild(content);
    
    // Add to document
    document.body.appendChild(container);
    logger.info('Test terminal created and added to document');
    
    return container;
}

// Connect to WebSocket and test terminal streaming
function connectTestWebSocket(instanceTitle) {
    if (!instanceTitle) {
        logger.error('No instance title provided');
        return null;
    }
    
    // Create test terminal if it doesn't exist
    const testTerminal = createTestTerminal();
    const contentArea = document.getElementById('test-terminal-content');
    
    // Set up WebSocket connection
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws/terminal/${instanceTitle}?format=ansi&privileges=read-only`;
    
    logger.info(`Connecting to WebSocket: ${wsUrl}`);
    contentArea.textContent += `Connecting to: ${instanceTitle}...\n`;
    
    try {
        const ws = new WebSocket(wsUrl);
        let messageCount = 0;
        
        ws.onopen = function() {
            logger.info('WebSocket connection established');
            contentArea.textContent += `Connected successfully!\n`;
        };
        
        ws.onmessage = function(event) {
            messageCount++;
            try {
                const data = JSON.parse(event.data);
                logger.debug(`Message #${messageCount} received`, data);
                
                if (data.type === 'config') {
                    logger.info('Received config message', data);
                    contentArea.textContent += `Received config: ${JSON.stringify(data)}\n`;
                    return;
                }
                
                // Terminal update
                if (data.content) {
                    const contentLength = data.content.length;
                    logger.info(`Received content update, length: ${contentLength}`);
                    
                    // Update test terminal content
                    contentArea.textContent += `\n--- Update #${messageCount} (${new Date().toLocaleTimeString()}) ---\n`;
                    contentArea.textContent += `Content length: ${contentLength}\n`;
                    contentArea.textContent += `Content preview: ${data.content.substring(0, Math.min(100, contentLength))}...\n`;
                    
                    // Auto-scroll
                    contentArea.scrollTop = contentArea.scrollHeight;
                    
                    // Check if real terminal is displaying content
                    testDisplayInRealTerminal(data.content);
                } else {
                    logger.warn('Received message without content');
                    contentArea.textContent += `Received empty update #${messageCount}\n`;
                }
            } catch (error) {
                logger.error('Error processing message', error);
                contentArea.textContent += `Error processing message: ${error.message}\n`;
            }
        };
        
        ws.onerror = function(error) {
            logger.error('WebSocket error', error);
            contentArea.textContent += `WebSocket error: ${error}\n`;
        };
        
        ws.onclose = function(event) {
            logger.info(`WebSocket closed. Clean: ${event.wasClean}, Code: ${event.code}`);
            contentArea.textContent += `\nConnection closed. Code: ${event.code}\n`;
            
            if (config.retryOnFailure && !event.wasClean) {
                const retryDelay = 3000;
                logger.info(`Will retry connection in ${retryDelay}ms`);
                contentArea.textContent += `Will retry connection in ${retryDelay}ms...\n`;
                
                setTimeout(() => {
                    contentArea.textContent += `Retrying connection...\n`;
                    connectTestWebSocket(instanceTitle);
                }, retryDelay);
            }
        };
        
        return ws;
    } catch (error) {
        logger.error('Error creating WebSocket', error);
        contentArea.textContent += `Error creating WebSocket: ${error.message}\n`;
        return null;
    }
}

// Check if the content is displayed in the actual terminal
function testDisplayInRealTerminal(content) {
    const terminalFallback = document.getElementById('terminal-fallback');
    const xtermDiv = document.getElementById('xterm-div');
    
    if (terminalFallback && terminalFallback.style.display !== 'none') {
        logger.info('Checking fallback terminal display');
        // Check if content is in the fallback terminal
        const fallbackContent = terminalFallback.textContent || '';
        
        if (fallbackContent.length > 0) {
            logger.info(`Fallback terminal has content (length: ${fallbackContent.length})`);
            document.getElementById('test-terminal-content').textContent += 
                `✅ Fallback terminal has content\n`;
        } else {
            logger.error('Fallback terminal is empty!');
            document.getElementById('test-terminal-content').textContent += 
                `❌ Fallback terminal is empty despite receiving content!\n`;
        }
    } else if (xtermDiv) {
        logger.info('xterm.js is being used, cannot directly verify content');
        document.getElementById('test-terminal-content').textContent += 
            `ℹ️ xterm.js terminal in use - content display can't be verified programmatically\n`;
    } else {
        logger.warn('Neither fallback nor xterm terminal found');
        document.getElementById('test-terminal-content').textContent += 
            `⚠️ No terminal element found!\n`;
    }
}

// Run the test
async function runTerminalTest() {
    const instances = await getInstances();
    if (instances.length > 0 && config.instance) {
        // Force fallback mode first if configured
        if (config.useFallback) {
            const toggleBtn = document.getElementById('toggle-mode-button');
            if (toggleBtn && toggleBtn.textContent !== 'Use xterm') {
                logger.info('Forcing fallback mode');
                toggleBtn.click();
            }
        }
        
        // Connect test WebSocket
        const ws = connectTestWebSocket(config.instance);
        
        return {
            instances,
            connection: ws,
            config,
            // Add methods to control the test
            setLogLevel: function(level) {
                if (LOG_LEVELS[level] !== undefined) {
                    config.logLevel = level;
                    logger.info(`Log level set to: ${level}`);
                }
            },
            toggleFallback: function() {
                const toggleBtn = document.getElementById('toggle-mode-button');
                if (toggleBtn) {
                    toggleBtn.click();
                    logger.info('Terminal display mode toggled');
                }
            },
            reconnect: function() {
                if (ws) ws.close();
                logger.info('Reconnecting...');
                return connectTestWebSocket(config.instance);
            }
        };
    } else {
        logger.error('Cannot run test: no instances available');
        return null;
    }
}

// Run the test and return the controller
const terminalTest = runTerminalTest();
console.log("Terminal Test initialized. Access the controller via the 'terminalTest' variable.");
console.log("Available methods: setLogLevel(), toggleFallback(), reconnect()");