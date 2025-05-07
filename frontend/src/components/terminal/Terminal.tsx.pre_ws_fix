import { useEffect, useRef, useState, useCallback } from 'react'
import { Terminal as XTerm } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebLinksAddon } from 'xterm-addon-web-links'
import 'xterm/css/xterm.css'

// Removed binary protocol message type constants - fully using JSON protocol now

interface TerminalProps {
  instanceName: string
  onConnectionChange?: (connected: boolean) => void
  onMessageReceived?: (count: number) => void
  onError?: (message: string) => void
  showStatusMessages?: boolean // Added option to hide status messages
}

interface TerminalStats {
  messagesReceived: number
  lastContentLength: number
}

const Terminal = ({ 
  instanceName, 
  onConnectionChange, 
  onMessageReceived,
  onError,
  showStatusMessages = false // Default to not showing status in terminal output
}: TerminalProps) => {
  const terminalRef = useRef<HTMLDivElement>(null)
  const socketRef = useRef<WebSocket | null>(null) // Add ref for WebSocket to maintain stable reference
  const [terminal, setTerminal] = useState<XTerm | null>(null)
  const [connected, setConnected] = useState(false)
  const [socket, setSocket] = useState<WebSocket | null>(null) // Keep for UI updates
  const [stats, setStats] = useState<TerminalStats>({
    messagesReceived: 0,
    lastContentLength: 0
  })
  
  // Status element for showing connection status
  const statusRef = useRef<HTMLDivElement>(null)
  const [statusMessage, setStatusMessage] = useState('Initializing...')
  const [statusClass, setStatusClass] = useState('info')
  
  const fitAddonRef = useRef<FitAddon | null>(null)
  
  // Log helper
  const log = useCallback((type: 'info' | 'error' | 'warn', message: string) => {
    const prefix = `[Terminal:${instanceName}]`
    switch (type) {
      case 'error':
        console.error(`${prefix} ${message}`)
        if (onError) onError(message)
        break
      case 'warn':
        console.warn(`${prefix} ${message}`)
        break
      default:
        console.log(`${prefix} ${message}`)
    }
    
    // Update status message based on log
    if (type === 'error') {
      setStatusMessage(message)
      setStatusClass('error')
    } else if (message.includes('connected') || message.includes('ready')) {
      setStatusMessage(message)
      setStatusClass('success')
    } else if (type === 'warn') {
      setStatusMessage(message)
      setStatusClass('warning')
    }
  }, [instanceName, onError])

  // Update terminal status (without writing to terminal)
  const updateStatus = useCallback((message: string, type: 'info' | 'error' | 'success' | 'warning' = 'info') => {
    setStatusMessage(message)
    setStatusClass(type)
    
    if (showStatusMessages && terminal) {
      // Optionally write to terminal if showStatusMessages is true
      terminal.write(`\r\n\x1b[33m[Status] ${message}\x1b[0m\r\n`)
    }
  }, [terminal, showStatusMessages])

  // Removed sendMessage function (binary protocol)

  // NOTE: Custom ping/pong mechanism removed in favor of standard WebSocket protocol

  // Send input to terminal
  const sendInput = useCallback((text: string) => {
    const currentSocket = socketRef.current; // Use ref instead of state
    if (!currentSocket || currentSocket.readyState !== WebSocket.OPEN || !text) {
      log('warn', 'Cannot send input - not connected or empty text')
      return
    }
    
    try {
      // Send using JSON protocol
      const message = {
        content: text,
        isCommand: false
      }
      
      currentSocket.send(JSON.stringify(message))
      log('info', `Sent input: ${text}`)
    } catch (error) {
      // Log error - don't fallback to binary protocol as server expects JSON
      log('error', `Failed to send input: ${error}. Connection may be unstable.`)
    }
  }, [log])

  // Clear terminal
  const clearTerminal = useCallback(() => {
    if (terminal) {
      terminal.clear()
      log('info', 'Terminal cleared')
      
      // Also try to tell server to clear (if supported)
      const currentSocket = socketRef.current;
      if (currentSocket && currentSocket.readyState === WebSocket.OPEN) {
        try {
          // Send using JSON protocol
          const message = {
            isCommand: true,
            content: 'clear_terminal'
          }
          currentSocket.send(JSON.stringify(message))
        } catch (error) {
          log('error', `Failed to send clear command: ${error}`)
        }
      }
    }
  }, [terminal, log])

  // Send terminal resize
  const sendResize = useCallback(() => {
    const currentSocket = socketRef.current;
    if (!currentSocket || currentSocket.readyState !== WebSocket.OPEN || !terminalRef.current) {
      return
    }
    
    // Only proceed if we have valid dimensions
    if (!terminal || !fitAddonRef.current) return
    
    try {
      // Get terminal dimensions directly from the terminal instance
      // instead of from options (which are just initial values)
      const cols = terminal.cols
      const rows = terminal.rows
      
      if (!cols || !rows) {
        log('warn', 'Invalid terminal dimensions')
        return
      }
      
      // Send JSON format
      const message = {
        cols: cols,
        rows: rows,
        isCommand: true,
        content: 'resize'
      }
      
      currentSocket.send(JSON.stringify(message))
      log('info', `Sent resize: ${cols}x${rows}`)
    } catch (error) {
      log('error', `Failed to send resize command: ${error}`)
    }
  }, [terminal, log])

  // Shared function to attempt fitting and resizing
  const attemptFitAndResize = useCallback(() => {
    if (!terminalRef.current || !terminal || !fitAddonRef.current) {
      log('warn', 'Fit attempt skipped: terminal or addons not ready.');
      return false;
    }
    if (terminalRef.current.offsetWidth === 0 || terminalRef.current.offsetHeight === 0) {
      log('warn', 'Fit attempt skipped: terminal container has no dimensions.');
      return false;
    }
    try {
      fitAddonRef.current.fit();
      log('info', `Terminal sized to ${terminalRef.current.offsetWidth}x${terminalRef.current.offsetHeight}`);
      // Send resize immediately after a successful fit if socket is open
      setTimeout(sendResize, 50); // Small delay to ensure terminal has updated its internal dimensions
      return true;
    } catch (err) {
      log('error', `Failed to fit terminal: ${err}`);
      if (err instanceof Error && err.message && err.message.includes('dimensions')) {
        log('error', "XTerm fit error likely due to terminal core not ready or element not measurable.");
      }
      return false;
    }
  }, [terminal, log, sendResize]);

  // Initialize terminal with proper dimension handling
  useEffect(() => {
    if (!terminalRef.current) return
    
    // Create xterm.js instance
    const term = new XTerm({
      cursorBlink: true,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      fontSize: 14,
      theme: {
        background: '#1e1e1e',
        foreground: '#f0f0f0',
        cursor: '#f0f0f0',
        selectionBackground: 'rgba(240, 240, 240, 0.3)'
      },
      convertEol: true,
      scrollback: 5000,
      // Explicit initial dimensions to avoid error
      cols: 80,
      rows: 24
    })
    
    // Create addons
    const fitAddon = new FitAddon()
    fitAddonRef.current = fitAddon
    const webLinksAddon = new WebLinksAddon()
    
    // Load addons
    term.loadAddon(fitAddon)
    term.loadAddon(webLinksAddon)
    
    // Important: Open terminal BEFORE storing it in state
    // to ensure DOM node is ready when other effects run
    if (terminalRef.current) {
      term.open(terminalRef.current)
      updateStatus('Initializing terminal...')
    }
    
    // THEN store terminal instance in state
    setTerminal(term)
    
    // Create a reference for timeouts to clear on unmount
    const timeoutRefs: number[] = []
    
    // Attempt initial fit after DOM has time to render
    const initialFitTimeout = window.setTimeout(() => {
      if (attemptFitAndResize()) {
        updateStatus(`Connecting to instance: ${instanceName}...`, 'info')
      }
    }, 150); // Slightly longer delay for initial fit
    timeoutRefs.push(initialFitTimeout)
    
    // Handle user input
    term.onData(data => {
      // Use socketRef directly for more stable reference
      if (connected && socketRef.current && socketRef.current.readyState === WebSocket.OPEN) {
        sendInput(data)
      }
    })
    
    // Handle window resize with debouncing to prevent excessive calls
    let resizeTimeout: number | null = null
    const handleResize = () => {
      if (resizeTimeout) {
        window.clearTimeout(resizeTimeout)
      }
      
      resizeTimeout = window.setTimeout(() => {
        attemptFitAndResize()
        resizeTimeout = null
      }, 150) // 150ms debounce
    }
    
    window.addEventListener('resize', handleResize)
    
    // Also handle visibility changes
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        // When tab becomes visible again, re-fit the terminal
        const visibilityTimeout = window.setTimeout(() => {
          attemptFitAndResize()
        }, 150)
        timeoutRefs.push(visibilityTimeout)
      }
    }
    
    document.addEventListener('visibilitychange', handleVisibilityChange)
    
    // Add keyboard shortcut for clearing terminal (Ctrl+L)
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.key === 'l') {
        clearTerminal()
        e.preventDefault()
      }
    }
    
    document.addEventListener('keydown', handleKeyDown)
    
    // Cleanup
    return () => {
      window.removeEventListener('resize', handleResize)
      document.removeEventListener('visibilitychange', handleVisibilityChange)
      document.removeEventListener('keydown', handleKeyDown)
      if (resizeTimeout) {
        window.clearTimeout(resizeTimeout)
      }
      // Clear all timeouts
      timeoutRefs.forEach(timeoutId => window.clearTimeout(timeoutId))
      term.dispose()
      setTerminal(null) // Clear terminal from state
      fitAddonRef.current = null // Clear ref to addon
    }
  }, [instanceName, updateStatus, sendResize, clearTerminal, log, attemptFitAndResize, connected])
  
  // NOTE: Custom ping interval removed in favor of standard WebSocket protocol
  // The browser will automatically handle standard WebSocket ping/pong frames
  
  // Reconnection with exponential backoff
  const reconnectAttemptsRef = useRef(0)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const maxReconnectDelay = 30000 // Max 30 seconds between reconnects
  const processedContentHashRef = useRef(new Set<string>()) // Track processed content
  
  // Hash function for deduplication
  const hashContent = useCallback((content: string): string => {
    // Simple hash for content deduplication
    let hash = 0
    for (let i = 0; i < content.length; i++) {
      const char = content.charCodeAt(i)
      hash = ((hash << 5) - hash) + char
      hash |= 0 // Convert to 32bit integer
    }
    return hash.toString(16)
  }, [])
  
  // Connect to WebSocket with proper connection limiting and backoff
  const connectWebSocket = useCallback(() => {
    // Reset content tracking on new connection
    processedContentHashRef.current.clear()
    
    // Don't try to connect if already connecting or connected
    // Use socketRef to check current socket state
    const currentSocket = socketRef.current;
    if (currentSocket) {
      if (currentSocket.readyState === WebSocket.OPEN) {
        log('warn', 'Already connected')
        return
      } else if (currentSocket.readyState === WebSocket.CONNECTING) {
        log('warn', 'Connection already in progress')
        return
      }
    }
    
    // Clear any existing reconnect timeout
    if (reconnectTimeoutRef.current !== null) {
      window.clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    
    // Update status to connecting (don't write to terminal)
    updateStatus('Connecting to WebSocket...', 'info')
    
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    // Using the path format expected by the server
    // Make sure to include 'format=ansi' to get raw ANSI codes for xterm.js
    const wsUrl = `${protocol}//${window.location.host}/ws/${instanceName}?format=ansi&privileges=read-write`
    
    log('info', `Connecting to WebSocket: ${wsUrl}`)
    
    try {
      const ws = new WebSocket(wsUrl)
      ws.binaryType = 'arraybuffer'
      
      // Set a connection timeout
      const connectionTimeout = window.setTimeout(() => {
        if (ws.readyState !== WebSocket.OPEN) {
          log('warn', 'WebSocket connection timeout')
          ws.close()
        }
      }, 10000) // 10 second connection timeout
      
      ws.onopen = () => {
        clearTimeout(connectionTimeout)
        log('info', 'WebSocket connection established')
        
        // Reset reconnection attempts on successful connection
        reconnectAttemptsRef.current = 0
        
        // Update status instead of writing to terminal
        updateStatus('WebSocket connected', 'success')
        
        // Update both ref and state atomically
        socketRef.current = ws; // Set ref first
        setConnected(true)
        setSocket(ws) // Update state for UI
        if (onConnectionChange) onConnectionChange(true)
        
        // Send initial resize after a brief delay to ensure terminal is ready
        if (terminal && fitAddonRef.current) {
          setTimeout(() => {
            // Make sure terminal has dimensions before sending resize
            if (terminal.cols === 0 || terminal.rows === 0) {
              log('warn', 'Terminal has invalid dimensions (0x0), attempting fit before resize')
              attemptFitAndResize()
            } else {
              sendResize()
              log('info', `Initial resize sent: ${terminal.cols}x${terminal.rows}`)
            }
          }, 300)
        } else {
          log('warn', 'Cannot send initial resize - terminal or fitAddon not ready')
        }
      }
      
      ws.onmessage = (event) => {
        setStats(prev => ({
          messagesReceived: prev.messagesReceived + 1,
          lastContentLength: typeof event.data === 'string' ? event.data.length : event.data.byteLength
        }))
        
        if (onMessageReceived) {
          onMessageReceived(stats.messagesReceived + 1)
        }
        
        try {
          // Handle JSON messages - we only support JSON protocol now
          if (typeof event.data === 'string') {
            try {
              const data = JSON.parse(event.data)
              
              if (data.type === 'config') {
                log('info', 'Received terminal config')
              } else if (data.type === 'instance_terminated') {
                // Handle instance termination notification from server
                log('warn', `Instance "${data.instance_title}" terminated: ${data.message}`)
                updateStatus(`Instance terminated: ${data.message}`, 'error')
                
                // Notify UI if callback provided
                if (onError) {
                  onError(`Instance "${data.instance_title}" terminated: ${data.message}`)
                }
                
                // Write message to terminal too
                if (terminal) {
                  terminal.write('\r\n\x1b[31m[ERROR] Instance terminated. The connection will now close.\x1b[0m\r\n')
                }
                
                // Close connection gracefully
                if (socketRef.current) {
                  socketRef.current.close(1000, "Instance terminated")
                  socketRef.current = null
                }
                
                // Update state
                setConnected(false)
                setSocket(null)
                if (onConnectionChange) onConnectionChange(false)
                
                // Don't attempt to reconnect in this case
                reconnectAttemptsRef.current = 11 // Exceed max reconnect attempts
                
                return
              } else if (data.type === 'error_response') {
                // Handle general error responses
                log('error', `Server error: ${data.error}`)
                updateStatus(`Server error: ${data.error}`, 'error')
                
                // Write error to terminal
                if (terminal) {
                  terminal.write(`\r\n\x1b[31m[ERROR] ${data.error}\x1b[0m\r\n`)
                }
                
                // For instance not found errors, handle similar to termination
                if (data.error.includes('not found') || data.error.includes('no longer exists')) {
                  // Close connection gracefully
                  if (socketRef.current) {
                    socketRef.current.close(1000, "Instance not found")
                    socketRef.current = null
                  }
                  
                  // Update state
                  setConnected(false)
                  setSocket(null)
                  if (onConnectionChange) onConnectionChange(false)
                  
                  // Don't attempt to reconnect
                  reconnectAttemptsRef.current = 11 // Exceed max reconnect attempts
                }
                
                return
              } else if (data.content) {
                // Process content only if it's not a duplicate
                const contentHash = hashContent(data.content)
                
                if (!processedContentHashRef.current.has(contentHash)) {
                  if (terminal) {
                    terminal.write(data.content)
                  }
                  
                  // Add to processed content set (limited size)
                  processedContentHashRef.current.add(contentHash)
                  
                  // Keep the set size reasonable
                  if (processedContentHashRef.current.size > 100) {
                    const iterator = processedContentHashRef.current.values()
                    processedContentHashRef.current.delete(iterator.next().value)
                  }
                } else {
                  log('info', 'Skipped duplicate content (JSON)')
                }
              }
            } catch (e) {
              // Not JSON, try to display as raw text
              if (terminal) {
                // Check for duplicates of raw text too
                const contentHash = hashContent(event.data)
                
                if (!processedContentHashRef.current.has(contentHash)) {
                  terminal.write(event.data)
                  processedContentHashRef.current.add(contentHash)
                }
              }
            }
          } else if (event.data instanceof ArrayBuffer) {
            // Handle binary protocol messages (for backward compatibility)
            const buffer = new Uint8Array(event.data)
            if (buffer.length > 0) {
              // Only handle OUTPUT_MESSAGE type for binary protocol
              const messageType = buffer[0]
              if (messageType === 'o'.charCodeAt(0) && buffer.length > 1) {
                const content = new TextDecoder().decode(buffer.slice(1))
                if (terminal) {
                  const contentHash = hashContent(content)
                  if (!processedContentHashRef.current.has(contentHash)) {
                    terminal.write(content)
                    processedContentHashRef.current.add(contentHash)
                  }
                }
              } else {
                log('warn', `Received unsupported binary message type: ${String.fromCharCode(messageType)}`)
              }
            }
          }
        } catch (error) {
          log('error', `Error handling message: ${error}`)
        }
      }
      
      ws.onclose = (event) => {
        clearTimeout(connectionTimeout)
        // Enhanced logging with more details about the close event
        log('info', `WebSocket connection closed: Code=${event.code}, Reason="${event.reason || 'None'}", Clean=${event.wasClean}, Instance=${instanceName}`)
        
        // Reset both ref and state
        socketRef.current = null; // Clear ref first
        setConnected(false)
        setSocket(null) // Update state for UI
        if (onConnectionChange) onConnectionChange(false)
        
        // Create more descriptive status message based on close code
        let statusMsg = '';
        let statusType = 'warning';
        
        // Common WebSocket close codes with user-friendly descriptions
        switch (event.code) {
          case 1000:
            statusMsg = 'Normal closure: Connection closed cleanly';
            break;
          case 1001:
            statusMsg = 'Server going down or browser navigating away';
            break;
          case 1002:
            statusMsg = 'Protocol error';
            statusType = 'error';
            break;
          case 1003:
            statusMsg = 'Invalid data received';
            statusType = 'error';
            break;
          case 1005:
            statusMsg = 'Connection closed without a status code';
            break;
          case 1006:
            statusMsg = 'Connection lost unexpectedly';
            statusType = 'error';
            break;
          case 1007:
            statusMsg = 'Message format error';
            statusType = 'error';
            break;
          case 1008:
            statusMsg = 'Policy violation';
            statusType = 'error';
            break;
          case 1009:
            statusMsg = 'Message too large';
            statusType = 'error';
            break;
          case 1010:
            statusMsg = 'Extension negotiation failed';
            break;
          case 1011:
            statusMsg = 'Server error';
            statusType = 'error';
            break;
          case 1012:
            statusMsg = 'Server restarting';
            break;
          case 1013:
            statusMsg = 'Try again later';
            break;
          case 1015:
            statusMsg = 'TLS handshake failed';
            statusType = 'error';
            break;
          default:
            statusMsg = `Unknown close code: ${event.code}`;
        }
        
        // Add reason if provided
        if (event.reason) {
          statusMsg += `: ${event.reason}`;
        }
        
        // Update status with detailed message
        updateStatus(`Connection closed: ${statusMsg}`, statusType as 'info' | 'warning' | 'error' | 'success')
        
        // Ping interval removed - using standard WebSocket protocol
        
        // Don't attempt to reconnect for certain close codes
        const noReconnectCodes = [1000, 1001, 1008, 1013] // Normal close, Going away, Policy violation, Try again later
        if (noReconnectCodes.includes(event.code)) {
          if (event.code === 1008) {
            // Rate limiting - don't automatically reconnect
            log('warn', 'Connection closed due to policy violation (rate limiting)')
            updateStatus('Connection closed due to rate limiting. Please wait before reconnecting.', 'error')
          }
          return
        }
        
        // Exponential backoff for reconnection
        reconnectAttemptsRef.current++
        
        // Calculate delay: 1s, 2s, 4s, 8s, 16s, up to maxReconnectDelay
        const baseDelay = 1000 // Start with 1 second
        const exponentialDelay = Math.min(
          maxReconnectDelay,
          baseDelay * Math.pow(2, reconnectAttemptsRef.current - 1)
        )
        
        // Add some randomness (jitter) to prevent thundering herd
        const jitter = Math.random() * 1000
        const reconnectDelay = exponentialDelay + jitter
        
        // Update status instead of writing to terminal
        updateStatus(`Connection lost. Reconnecting in ${Math.round(reconnectDelay/1000)} seconds...`, 'warning')
        
        log('info', `Scheduling reconnect in ${reconnectDelay/1000}s (attempt ${reconnectAttemptsRef.current})`)
        
        // Schedule reconnection
        reconnectTimeoutRef.current = window.setTimeout(() => {
          reconnectTimeoutRef.current = null
          
          // Limit total reconnection attempts
          if (reconnectAttemptsRef.current > 10) {
            log('warn', 'Maximum reconnection attempts reached')
            updateStatus('Maximum reconnection attempts reached. Please refresh the page.', 'error')
            return
          }
          
          // Only try to reconnect if we're not already connected
          connectWebSocket()
        }, reconnectDelay)
      }
      
      ws.onerror = (event) => {
        // Enhanced error logging with more details
        const errorTime = new Date().toISOString();
        log('error', `WebSocket error at ${errorTime} for ${instanceName}, ReadyState: ${ws.readyState}`)
        
        // Add additional debugging info about the connection
        console.error('WebSocket error details:', {
          url: wsUrl,
          instance: instanceName,
          readyState: ws.readyState,
          bufferedAmount: ws.bufferedAmount,
          extensions: ws.extensions,
          protocol: ws.protocol,
          timestamp: errorTime
        })
        
        updateStatus('Connection error. Check console for details. The system will attempt to reconnect automatically.', 'error')
        // No need to close the socket here, the onclose handler will fire
      }
      
      setSocket(ws)
    } catch (error) {
      log('error', `Failed to create WebSocket: ${error}`)
      updateStatus(`Failed to connect: ${error}`, 'error')
      
      // Make sure socketRef is cleared in case of error
      socketRef.current = null;
      
      // Schedule a retry after a delay
      const retryDelay = 3000 // 3 seconds
      reconnectTimeoutRef.current = window.setTimeout(() => {
        reconnectTimeoutRef.current = null
        connectWebSocket()
      }, retryDelay)
    }
  }, [
    // Include correct dependencies
    terminal, 
    instanceName, 
    onConnectionChange, 
    onMessageReceived, 
    sendResize, 
    hashContent,
    updateStatus,
    log,
    attemptFitAndResize
  ])
  
  // Connect to WebSocket
  useEffect(() => {
    if (!terminal || !instanceName) return
    
    // Connect on mount
    connectWebSocket()
    
    // Cleanup on unmount
    return () => {
      // Clear any pending reconnect timeouts
      if (reconnectTimeoutRef.current !== null) {
        window.clearTimeout(reconnectTimeoutRef.current)
        reconnectTimeoutRef.current = null
      }
      
      // Use socketRef to ensure we close the most up-to-date socket
      const currentSocket = socketRef.current;
      
      // Close socket if it's still open
      if (currentSocket) {
        // Send graceful close message if possible
        if (currentSocket.readyState === WebSocket.OPEN) {
          try {
            // Standard WebSocket close with reason
            currentSocket.close(1000, "Terminal component unmounting")
          } catch (err) {
            // If closing with reason fails, just close it
            currentSocket.close()
          }
        } else if (currentSocket.readyState === WebSocket.CONNECTING) {
          // Also close if it's still connecting
          currentSocket.close()
        }
        
        // Clear the ref
        socketRef.current = null;
      }
      
      // Update state
      setConnected(false)
      if (onConnectionChange) onConnectionChange(false)
    }
  }, [
    terminal, 
    instanceName, 
    connectWebSocket,
    // Removed socket and sendMessage from dependencies
    onConnectionChange
  ])
  
  // Define CSS for status indicator
  const statusStyles = {
    info: {
      backgroundColor: '#2c2c2c',
      color: '#ccc',
      border: '1px solid #444'
    },
    warning: {
      backgroundColor: '#3a2e00',
      color: '#ffcc00',
      border: '1px solid #664d00'
    },
    error: {
      backgroundColor: '#3a0000',
      color: '#ff5555',
      border: '1px solid #660000'
    },
    success: {
      backgroundColor: '#00360c',
      color: '#4cff83',
      border: '1px solid #006d18'
    }
  }
  
  // Status indicator styles
  const statusContainerStyle = {
    padding: '4px 8px',
    fontSize: '12px',
    borderRadius: '4px',
    margin: '4px 0',
    maxWidth: '100%',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap' as 'nowrap',
    ...statusStyles[statusClass as keyof typeof statusStyles]
  }
  
  return (
    <div className="terminal-wrapper" style={{ 
      width: '100%', 
      height: '100%',
      display: 'flex',
      flexDirection: 'column' 
    }}>
      {/* Status indicator */}
      <div 
        ref={statusRef} 
        className={`terminal-status ${statusClass}`}
        style={statusContainerStyle}
      >
        {statusMessage}
      </div>
      
      {/* Terminal container */}
      <div
        ref={terminalRef}
        className="xterm-container"
        style={{ 
          width: '100%', 
          height: 'calc(100% - 30px)', // Reserve space for status
          border: '1px solid #444',
          borderRadius: '4px'
        }}
      />
    </div>
  )
}

export default Terminal