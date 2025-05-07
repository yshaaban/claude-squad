// Enhanced terminal WebSocket client with xterm.js support
document.addEventListener('DOMContentLoaded', function() {
    console.log("Loading enhanced terminal client with xterm.js support");
    
    // Constants
    const OUTPUT_MESSAGE = 'o'.charCodeAt(0);
    const INPUT_MESSAGE = 'i'.charCodeAt(0);
    const RESIZE_MESSAGE = 'r'.charCodeAt(0);
    const PING_MESSAGE = 'p'.charCodeAt(0);
    const PONG_MESSAGE = 'P'.charCodeAt(0);
    const CLOSE_MESSAGE = 'c'.charCodeAt(0);

    // Get instance from URL or default
    const urlParams = new URLSearchParams(window.location.search);
    const instanceName = urlParams.get('instance') || 'simple-default';
    
    console.log(`Starting terminal for instance: ${instanceName}`);
    
    // Get terminal elements
    const terminalContainer = document.getElementById('terminal-container');
    const outputElement = document.getElementById('terminal-output');
    const inputElement = document.getElementById('terminal-input');
    
    if (!terminalContainer || !outputElement || !inputElement) {
        console.error('Terminal elements not found');
        return;
    }
    
    // Setup WebSocket connection
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    // Use the new WebSocket path structure that the server expects
    const wsUrl = `${protocol}//${window.location.host}/ws/${instanceName}?format=ansi`;
    
    console.log(`Connecting to WebSocket: ${wsUrl}`);
    addSystemMessage('Connecting to terminal...');
    
    let socket = null;
    let terminal = null;
    
    function setupTerminal() {
        // We'll use xterm.js for proper ANSI rendering
        // If it's not already loaded, you'd need to add these to your HTML:
        // <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/xterm@5.1.0/css/xterm.css" />
        // <script src="https://cdn.jsdelivr.net/npm/xterm@5.1.0/lib/xterm.js"></script>
        
        if (typeof Terminal !== 'undefined') {
            terminal = new Terminal({
                cursorBlink: true,
                fontFamily: 'Menlo, Monaco, "Courier New", monospace',
                fontSize: 14
            });
            terminal.open(outputElement);
            
            // Clear existing content
            outputElement.innerHTML = '';
            
            // Set up input handling from xterm.js if needed
            terminal.onData(data => {
                sendInput(data);
            });
            
            return true;
        } else {
            console.warn("Xterm.js not loaded - falling back to basic rendering");
            return false;
        }
    }
    
    function connect() {
        try {
            // Try to set up the terminal first
            const usingXterm = setupTerminal();
            
            socket = new WebSocket(wsUrl);
            
            // We'll now expect JSON, not binary data
            socket.onopen = function() {
                console.log('WebSocket connection established');
                if (terminal) {
                    terminal.writeln('\r\nConnected to terminal\r\n');
                } else {
                    addSystemMessage('Connected to terminal');
                }
                
                // Send initial resize
                sendResize();
            };
            
            socket.onmessage = function(event) {
                try {
                    // Parse JSON data
                    const data = JSON.parse(event.data);
                    
                    if (data.type === 'config') {
                        console.log('Received terminal config:', data);
                        return;
                    }
                    
                    if (data.content) {
                        if (terminal) {
                            // Direct ANSI to xterm.js terminal
                            terminal.write(data.content);
                        } else {
                            // Fallback for non-xterm display
                            displayOutput(data.content);
                        }
                    }
                } catch (e) {
                    console.error('Error parsing message:', e);
                    
                    // Fallback: try to display as text
                    if (typeof event.data === 'string') {
                        displayOutput(event.data);
                    }
                }
            };
            
            socket.onclose = function(event) {
                console.log('WebSocket connection closed:', event);
                addSystemMessage('Connection lost. Attempting to reconnect...');
                
                // Attempt to reconnect after delay
                setTimeout(connect, 3000);
            };
            
            socket.onerror = function(error) {
                console.error('WebSocket error:', error);
                addSystemMessage('Error connecting to terminal');
            };
        } catch (error) {
            console.error('Failed to create WebSocket:', error);
            addSystemMessage('Failed to connect to terminal');
        }
    }
    
    function displayOutput(content) {
        console.log(`Received output (${content.length} bytes)`);
        
        if (terminal) {
            // If using xterm.js, write directly to terminal
            terminal.write(content);
            return;
        }
        
        // Fallback for basic HTML rendering
        outputElement.innerHTML = '';
        
        // Split content into lines
        const lines = content.split('\n');
        
        // Add each line
        for (const line of lines) {
            const lineElement = document.createElement('div');
            lineElement.className = 'terminal-line';
            
            // Process ANSI escape codes for basic formatting in fallback mode
            if (line.includes('\x1b[')) {
                // Create a span with a basic ANSI-to-CSS converter
                const span = document.createElement('span');
                span.style.whiteSpace = 'pre';
                span.textContent = line; // Preserve ANSI codes for now
                
                // In a real implementation, you would apply color/style based on codes
                // Basic implementation left to preserve original behavior
                
                lineElement.appendChild(span);
            } else {
                lineElement.textContent = line;
            }
            
            outputElement.appendChild(lineElement);
        }
        
        // Scroll to bottom
        outputElement.scrollTop = outputElement.scrollHeight;
    }
    
    function addSystemMessage(text) {
        const lineElement = document.createElement('div');
        lineElement.className = 'terminal-line system';
        lineElement.textContent = text;
        outputElement.appendChild(lineElement);
        outputElement.scrollTop = outputElement.scrollHeight;
    }
    
    function sendInput(text) {
        if (!socket || socket.readyState !== WebSocket.OPEN || !text) {
            return;
        }
        
        // Send JSON format message
        const message = {
            content: text,
            isCommand: false
        };
        
        try {
            socket.send(JSON.stringify(message));
            console.log(`Sent input: ${text}`);
        } catch (error) {
            console.error('Failed to send input:', error);
        }
    }
    
    function sendResize() {
        if (!socket || socket.readyState !== WebSocket.OPEN) {
            return;
        }
        
        // Get container dimensions
        const containerWidth = terminalContainer.clientWidth;
        const containerHeight = terminalContainer.clientHeight;
        
        // Approximate character dimensions
        const charWidth = 8;
        const charHeight = 16;
        
        // Calculate columns and rows
        const cols = Math.floor(containerWidth / charWidth);
        const rows = Math.floor(containerHeight / charHeight);
        
        // Create resize message
        const message = {
            cols: cols,
            rows: rows,
            isCommand: true,
            content: 'resize'
        };
        
        try {
            socket.send(JSON.stringify(message));
            console.log(`Sent resize: ${cols}x${rows}`);
        } catch (error) {
            console.error('Failed to send resize:', error);
        }
    }
    
    function sendPing() {
        if (!socket || socket.readyState !== WebSocket.OPEN) {
            return;
        }
        
        sendMessage(PING_MESSAGE, new Uint8Array(0));
        console.log('Sent ping');
    }
    
    function sendMessage(type, data) {
        if (!socket || socket.readyState !== WebSocket.OPEN) {
            return;
        }
        
        // Create message with type prefix
        const message = new Uint8Array(data.length + 1);
        message[0] = type;
        message.set(data, 1);
        
        try {
            socket.send(message);
        } catch (error) {
            console.error('Failed to send message:', error);
        }
    }
    
    // Setup input handler
    inputElement.addEventListener('keydown', function(e) {
        if (e.key === 'Enter') {
            const text = inputElement.value;
            sendInput(text);
            inputElement.value = '';
            e.preventDefault();
        }
    });
    
    // Setup resize handler
    window.addEventListener('resize', sendResize);
    
    // Start connection
    connect();
});