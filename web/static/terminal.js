// Terminal connection handling
class TerminalConnection {
    constructor(instanceName, container) {
        this.instanceName = instanceName;
        this.container = container;
        this.socket = null;
        this.connected = false;
        this.reconnecting = false;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        this.xterm = null;
        
        // Message type constants
        this.MESSAGE_OUTPUT = 'o'.charCodeAt(0);
        this.MESSAGE_INPUT = 'i'.charCodeAt(0);
        this.MESSAGE_RESIZE = 'r'.charCodeAt(0);
        this.MESSAGE_PING = 'p'.charCodeAt(0);
        this.MESSAGE_PONG = 'P'.charCodeAt(0);
        this.MESSAGE_CLOSE = 'c'.charCodeAt(0);
        
        // Create terminal elements
        this.terminalElement = document.createElement('div');
        this.terminalElement.className = 'terminal-output';
        this.container.appendChild(this.terminalElement);
        
        // Create input element
        this.inputElement = document.createElement('input');
        this.inputElement.type = 'text';
        this.inputElement.className = 'terminal-input';
        this.inputElement.placeholder = 'Enter command...';
        this.container.appendChild(this.inputElement);
        
        // Try to initialize xterm.js if it's available
        this.initXterm();
        
        // Connect input event
        this.inputElement.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                this.sendInput(this.inputElement.value);
                this.inputElement.value = '';
                e.preventDefault();
            }
        });
        
        // Resize handling
        window.addEventListener('resize', () => {
            this.sendResize();
        });
    }
    
    connect() {
        if (this.connected || this.reconnecting) {
            return;
        }
        
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws/terminal/${this.instanceName}`;
        
        this.addTerminalLine('Connecting to terminal...');
        
        try {
            this.socket = new WebSocket(wsUrl);
            
            this.socket.binaryType = 'arraybuffer';
            
            this.socket.onopen = () => {
                this.connected = true;
                this.reconnectAttempts = 0;
                this.addTerminalLine('Connected to terminal.');
                
                // Send initial resize
                this.sendResize();
                
                // Start ping interval
                this.pingInterval = setInterval(() => {
                    this.sendPing();
                }, 30000);
            };
            
            this.socket.onmessage = (event) => {
                const data = new Uint8Array(event.data);
                if (data.length === 0) {
                    return;
                }
                
                const messageType = data[0];
                const payload = data.slice(1);
                
                switch (messageType) {
                    case this.MESSAGE_OUTPUT:
                        this.handleOutput(new TextDecoder().decode(payload));
                        break;
                    case this.MESSAGE_PONG:
                        // Pong received, connection is alive
                        break;
                    default:
                        console.log('Unknown message type:', messageType);
                }
            };
            
            this.socket.onclose = (event) => {
                this.connected = false;
                clearInterval(this.pingInterval);
                
                if (!event.wasClean) {
                    this.addTerminalLine('Connection lost. Attempting to reconnect...');
                    this.attemptReconnect();
                } else {
                    this.addTerminalLine('Terminal connection closed.');
                }
            };
            
            this.socket.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.addTerminalLine('Error connecting to terminal.');
            };
        } catch (error) {
            console.error('Failed to create WebSocket:', error);
            this.addTerminalLine('Failed to connect to terminal.');
            this.attemptReconnect();
        }
    }
    
    disconnect() {
        if (!this.connected) {
            return;
        }
        
        clearInterval(this.pingInterval);
        
        if (this.socket) {
            // Send close message
            this.sendMessage(this.MESSAGE_CLOSE, new Uint8Array(0));
            
            // Close socket after a short delay to allow the close message to be sent
            setTimeout(() => {
                this.socket.close();
                this.socket = null;
                this.connected = false;
            }, 100);
        }
    }
    
    attemptReconnect() {
        if (this.reconnecting || this.reconnectAttempts >= this.maxReconnectAttempts) {
            if (this.reconnectAttempts >= this.maxReconnectAttempts) {
                this.addTerminalLine('Failed to reconnect after multiple attempts.');
            }
            return;
        }
        
        this.reconnecting = true;
        this.reconnectAttempts++;
        
        setTimeout(() => {
            this.reconnecting = false;
            this.connect();
        }, this.reconnectDelay * this.reconnectAttempts);
    }
    
    // Initialize xterm.js if available
    initXterm() {
        // Check if xterm.js is available
        if (typeof Terminal !== 'undefined') {
            try {
                // Create xterm.js terminal
                console.log('Creating xterm.js terminal');
                this.xterm = new Terminal({
                    cursorBlink: true,
                    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
                    fontSize: 14,
                    theme: {
                        background: '#1e1e1e',
                        foreground: '#f0f0f0'
                    },
                    convertEol: true
                });
                
                // Hide the legacy terminal and clear it
                this.terminalElement.style.display = 'none';
                this.terminalElement.innerHTML = '';
                
                // Create xterm container
                this.xtermContainer = document.createElement('div');
                this.xtermContainer.className = 'xterm-container';
                this.xtermContainer.style.flex = '1';
                this.xtermContainer.style.overflow = 'hidden';
                
                // Insert xterm container before input element
                this.container.insertBefore(this.xtermContainer, this.inputElement);
                
                // Open terminal
                this.xterm.open(this.xtermContainer);
                
                // Hook up xterm input to websocket
                this.xterm.onData(data => {
                    if (this.connected) {
                        this.sendInput(data);
                    }
                });
                
                // Hide the regular input field when using xterm
                this.inputElement.style.display = 'none';
                
                // Add initial welcome message
                this.xterm.writeln('Terminal ready - using xterm.js for ANSI rendering');
                this.xterm.writeln('Connecting to server...');
                
                console.log('xterm.js terminal initialized');
                return true;
            } catch (e) {
                console.error('Failed to initialize xterm.js:', e);
                this.xterm = null;
                return false;
            }
        } else {
            console.log('xterm.js not available, using fallback rendering');
            return false;
        }
    }
    
    handleOutput(content) {
        // If we have xterm.js, use it for rendering
        if (this.xterm) {
            this.xterm.write(content);
            return;
        }
        
        // Otherwise use the fallback HTML-based renderer
        // Replace terminal content with the new content
        this.terminalElement.innerHTML = '';
        
        // Split content into lines
        const lines = content.split('\n');
        
        // Add each line with ANSI interpretation
        for (const line of lines) {
            const lineElement = document.createElement('div');
            lineElement.className = 'terminal-line';
            
            // Process ANSI escape sequences
            if (line.includes('\x1b[')) {
                // Create a styled element for ANSI content
                lineElement.innerHTML = this.processAnsiSequences(line);
            } else {
                lineElement.textContent = line;
            }
            
            this.terminalElement.appendChild(lineElement);
        }
        
        // Scroll to bottom
        this.terminalElement.scrollTop = this.terminalElement.scrollHeight;
    }
    
    // Process ANSI escape sequences and convert to HTML (fallback method)
    processAnsiSequences(text) {
        // Replace common ANSI escape sequences with HTML/CSS
        const ansiToHtml = {
            // Reset
            '\x1b[0m': '</span>',
            
            // Bold
            '\x1b[1m': '<span style="font-weight: bold;">',
            
            // Colors (foreground)
            '\x1b[30m': '<span style="color: black;">',
            '\x1b[31m': '<span style="color: red;">',
            '\x1b[32m': '<span style="color: green;">',
            '\x1b[33m': '<span style="color: yellow;">',
            '\x1b[34m': '<span style="color: blue;">',
            '\x1b[35m': '<span style="color: magenta;">',
            '\x1b[36m': '<span style="color: cyan;">',
            '\x1b[37m': '<span style="color: white;">',
            
            // Bright colors
            '\x1b[90m': '<span style="color: #888;">',
            '\x1b[91m': '<span style="color: #f55;">',
            '\x1b[92m': '<span style="color: #5f5;">',
            '\x1b[93m': '<span style="color: #ff5;">',
            '\x1b[94m': '<span style="color: #55f;">',
            '\x1b[95m': '<span style="color: #f5f;">',
            '\x1b[96m': '<span style="color: #5ff;">',
            '\x1b[97m': '<span style="color: #fff;">',
            
            // Background colors
            '\x1b[40m': '<span style="background-color: black;">',
            '\x1b[41m': '<span style="background-color: red;">',
            '\x1b[42m': '<span style="background-color: green;">',
            '\x1b[43m': '<span style="background-color: yellow;">',
            '\x1b[44m': '<span style="background-color: blue;">',
            '\x1b[45m': '<span style="background-color: magenta;">',
            '\x1b[46m': '<span style="background-color: cyan;">',
            '\x1b[47m': '<span style="background-color: white;">'
        };
        
        // Replace all ANSI escape codes with their HTML equivalents
        let result = text;
        
        // First handle the complex ANSI sequences with regex
        result = result.replace(/\x1b\[\d+(;\d+)*(m|K)/g, (match) => {
            // Handle clear to end of line
            if (match.endsWith('K')) {
                return '';
            }
            
            // Handle known color codes
            if (ansiToHtml[match]) {
                return ansiToHtml[match];
            }
            
            // Clean other ANSI sequences we don't specifically handle
            return '';
        });
        
        // Clean up any remaining unhandled escape sequences
        result = result.replace(/\x1b\[[\d;]*[A-Za-z]/g, '');
        
        // Ensure all opened spans are closed
        const openSpans = (result.match(/<span/g) || []).length;
        const closeSpans = (result.match(/<\/span>/g) || []).length;
        
        // Add closing spans if needed
        if (openSpans > closeSpans) {
            result += '</span>'.repeat(openSpans - closeSpans);
        }
        
        return result;
    }
    
    addTerminalLine(text) {
        // If using xterm.js, send to xterm
        if (this.xterm) {
            // Add a system message with green color and newline
            this.xterm.write('\r\n\x1b[32m' + text + '\x1b[0m\r\n');
            return;
        }
        
        // Legacy HTML rendering
        const lineElement = document.createElement('div');
        lineElement.className = 'terminal-line system';
        lineElement.textContent = text;
        this.terminalElement.appendChild(lineElement);
        this.terminalElement.scrollTop = this.terminalElement.scrollHeight;
    }
    
    sendInput(text) {
        if (!this.connected || !text) {
            return;
        }
        
        const data = new TextEncoder().encode(text);
        this.sendMessage(this.MESSAGE_INPUT, data);
    }
    
    sendResize() {
        if (!this.connected) {
            return;
        }
        
        // Calculate terminal dimensions based on container size
        // This is a simple approximation - a real implementation would use character metrics
        const containerWidth = this.container.clientWidth;
        const containerHeight = this.container.clientHeight;
        
        // Assume average character width of 8px and height of 16px
        const cols = Math.floor(containerWidth / 8);
        const rows = Math.floor(containerHeight / 16);
        
        // Send resize message
        const resizeData = JSON.stringify({ cols, rows });
        this.sendMessage(this.MESSAGE_RESIZE, new TextEncoder().encode(resizeData));
    }
    
    sendPing() {
        if (!this.connected) {
            return;
        }
        
        this.sendMessage(this.MESSAGE_PING, new Uint8Array(0));
    }
    
    sendMessage(type, data) {
        if (!this.connected || !this.socket) {
            return;
        }
        
        // Create message with type prefix
        const message = new Uint8Array(data.length + 1);
        message[0] = type;
        message.set(data, 1);
        
        try {
            this.socket.send(message);
        } catch (error) {
            console.error('Failed to send message:', error);
        }
    }
}

// Initialize terminal connections
function initializeTerminals() {
    const terminalContainers = document.querySelectorAll('.terminal-container');
    
    terminalContainers.forEach(container => {
        const instanceName = container.getAttribute('data-instance');
        
        if (instanceName) {
            const terminal = new TerminalConnection(instanceName, container);
            terminal.connect();
            
            // Store terminal instance for later access
            container.terminal = terminal;
        }
    });
}

// Connect to a specific instance
function connectToInstance(instanceName, containerId) {
    const container = document.getElementById(containerId);
    
    if (container) {
        const terminal = new TerminalConnection(instanceName, container);
        terminal.connect();
        
        // Store terminal instance for later access
        container.terminal = terminal;
        
        return terminal;
    }
    
    return null;
}

// Disconnect from a specific instance
function disconnectFromInstance(containerId) {
    const container = document.getElementById(containerId);
    
    if (container && container.terminal) {
        container.terminal.disconnect();
        delete container.terminal;
    }
}

// Initialize terminals when document is ready
document.addEventListener('DOMContentLoaded', initializeTerminals);