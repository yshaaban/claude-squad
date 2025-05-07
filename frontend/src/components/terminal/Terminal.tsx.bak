import { useEffect, useRef, useState, useCallback } from 'react'
import { Terminal as XTerm } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebLinksAddon } from 'xterm-addon-web-links'
import 'xterm/css/xterm.css'

// Message type constants for binary protocol
const OUTPUT_MESSAGE = 'o'.charCodeAt(0)
const INPUT_MESSAGE = 'i'.charCodeAt(0)
const RESIZE_MESSAGE = 'r'.charCodeAt(0)
const PING_MESSAGE = 'p'.charCodeAt(0)
const PONG_MESSAGE = 'P'.charCodeAt(0)
const CLOSE_MESSAGE = 'c'.charCodeAt(0)

interface TerminalProps {
  instanceName: string
  onConnectionChange?: (connected: boolean) => void
  onMessageReceived?: (count: number) => void
  onError?: (message: string) => void
}

interface TerminalStats {
  messagesReceived: number
  lastContentLength: number
}

const Terminal = ({ 
  instanceName, 
  onConnectionChange, 
  onMessageReceived,
  onError 
}: TerminalProps) => {
  const terminalRef = useRef<HTMLDivElement>(null)
  const [terminal, setTerminal] = useState<XTerm | null>(null)
  const [connected, setConnected] = useState(false)
  const [socket, setSocket] = useState<WebSocket | null>(null)
  const [stats, setStats] = useState<TerminalStats>({
    messagesReceived: 0,
    lastContentLength: 0
  })
  const fitAddonRef = useRef<FitAddon | null>(null)
  const pingIntervalRef = useRef<number | null>(null)
  const missedHeartbeatsRef = useRef<number>(0)
  
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
  }, [instanceName, onError])

  // Send binary message with type prefix
  const sendMessage = useCallback((type: number, data: Uint8Array) => {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      log('warn', 'Cannot send message - not connected')
      return false
    }
    
    try {
      // Create message with type prefix
      const message = new Uint8Array(data.length + 1)
      message[0] = type
      message.set(data, 1)
      
      socket.send(message)
      return true
    } catch (error) {
      log('error', `Failed to send message: ${error}`)
      return false
    }
  }, [socket, log])

  // Send ping to keep connection alive
  const sendPing = useCallback(() => {
    if (!socket || socket.readyState !== WebSocket.OPEN) return
    
    sendMessage(PING_MESSAGE, new Uint8Array(0))
    missedHeartbeatsRef.current++
    log('info', `Sending keep-alive ping (missed: ${missedHeartbeatsRef.current})`)
  }, [socket, sendMessage, log])

  // Send input to terminal
  const sendInput = useCallback((text: string) => {
    if (!socket || socket.readyState !== WebSocket.OPEN || !text) {
      log('warn', 'Cannot send input - not connected or empty text')
      return
    }
    
    try {
      // Try JSON format first (new protocol)
      const message = {
        content: text,
        isCommand: false
      }
      
      socket.send(JSON.stringify(message))
      log('info', `Sent input: ${text}`)
    } catch (error) {
      // Fallback to binary protocol
      const data = new TextEncoder().encode(text)
      sendMessage(INPUT_MESSAGE, data)
      log('info', `Sent input using binary protocol: ${text}`)
    }
  }, [socket, sendMessage, log])

  // Send terminal resize
  const sendResize = useCallback(() => {
    if (!socket || socket.readyState !== WebSocket.OPEN || !terminalRef.current) {
      return
    }
    
    // Get container dimensions
    const containerWidth = terminalRef.current.clientWidth
    const containerHeight = terminalRef.current.clientHeight
    
    // Approximate character dimensions
    const charWidth = 8
    const charHeight = 16
    
    // Calculate columns and rows
    const cols = Math.floor(containerWidth / charWidth)
    const rows = Math.floor(containerHeight / charHeight)
    
    try {
      // Try JSON format first
      const message = {
        cols: cols,
        rows: rows,
        isCommand: true,
        content: 'resize'
      }
      
      socket.send(JSON.stringify(message))
      log('info', `Sent resize as JSON: ${cols}x${rows}`)
    } catch (error) {
      // Fallback to binary protocol
      const resizeData = JSON.stringify({ cols, rows })
      sendMessage(RESIZE_MESSAGE, new TextEncoder().encode(resizeData))
      log('info', `Sent resize as binary fallback: ${cols}x${rows}`)
    }
  }, [socket, sendMessage, log])

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
      scrollback: 5000
    })
    
    // Create addons
    const fitAddon = new FitAddon()
    fitAddonRef.current = fitAddon
    const webLinksAddon = new WebLinksAddon()
    
    // Load addons
    term.loadAddon(fitAddon)
    term.loadAddon(webLinksAddon)
    
    // Store terminal instance first before opening it
    setTerminal(term)
    
    // Safely fit terminal only when the DOM is ready and visible
    const safelyFitTerminal = () => {
      try {
        // Check if the terminal container has dimensions before fitting
        if (terminalRef.current && 
            terminalRef.current.offsetWidth > 0 && 
            terminalRef.current.offsetHeight > 0) {
          fitAddon.fit()
          log('info', `Terminal sized to ${terminalRef.current.offsetWidth}x${terminalRef.current.offsetHeight}`)
        } else {
          log('warn', 'Terminal container has no dimensions yet, skipping fit')
        }
      } catch (err) {
        log('error', `Failed to fit terminal: ${err}`)
      }
    }
    
    // Wait for the next frame to ensure DOM is ready
    requestAnimationFrame(() => {
      // Open terminal
      if (terminalRef.current) {
        term.open(terminalRef.current)
        
        // Try to fit after a short delay to allow DOM to fully render
        setTimeout(() => {
          safelyFitTerminal()
          
          // Write welcome message after we're sure the terminal is ready
          term.writeln('Initializing terminal...')
          term.writeln(`Connecting to instance: ${instanceName}`)
        }, 100)
      }
    })
    
    // Handle user input
    term.onData(data => {
      if (connected && socket && socket.readyState === WebSocket.OPEN) {
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
        safelyFitTerminal()
        sendResize()
        resizeTimeout = null
      }, 100) // 100ms debounce
    }
    
    window.addEventListener('resize', handleResize)
    
    // Also handle visibility changes
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        // When tab becomes visible again, re-fit the terminal
        setTimeout(safelyFitTerminal, 100)
      }
    }
    
    document.addEventListener('visibilitychange', handleVisibilityChange)
    
    // Cleanup
    return () => {
      window.removeEventListener('resize', handleResize)
      document.removeEventListener('visibilitychange', handleVisibilityChange)
      if (resizeTimeout) {
        window.clearTimeout(resizeTimeout)
      }
      term.dispose()
    }
  }, [instanceName]) // Remove connected and socket deps to avoid re-creating terminal
  
  // Start ping interval
  const startPingInterval = useCallback(() => {
    // Clear existing interval if any
    if (pingIntervalRef.current) {
      clearInterval(pingIntervalRef.current)
    }
    
    // Reset heartbeat counter
    missedHeartbeatsRef.current = 0
    
    // Start new interval
    const MAX_MISSED_HEARTBEATS = 3
    
    pingIntervalRef.current = window.setInterval(() => {
      if (socket && socket.readyState === WebSocket.OPEN) {
        // Check if we've missed too many heartbeats
        if (missedHeartbeatsRef.current >= MAX_MISSED_HEARTBEATS) {
          log('warn', `Missed ${missedHeartbeatsRef.current} heartbeats - reconnecting`)
          
          // Force close and reconnect
          socket.close()
          clearInterval(pingIntervalRef.current!)
          pingIntervalRef.current = null
          
          // Schedule reconnection after a delay - reconnectWebSocket is defined later
          setTimeout(() => {
            log('info', 'Attempting to reconnect after missed heartbeats')
            // We will manually reconnect here rather than using the not-yet-defined function
            setSocket(null)
            setConnected(false)
          }, 1000)
          
          return
        }
        
        // Send ping
        sendPing()
      } else {
        // Not connected
        log('warn', 'Cannot send ping - connection not open')
        clearInterval(pingIntervalRef.current!)
        pingIntervalRef.current = null
      }
    }, 15000) // Every 15 seconds
  }, [socket, sendPing, log])
  
  // Reconnection with exponential backoff
  const reconnectAttemptsRef = useRef(0)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const maxReconnectDelay = 30000 // Max 30 seconds between reconnects
  
  // Connect to WebSocket with proper connection limiting and backoff
  const connectWebSocket = useCallback(() => {
    // Don't try to connect if already connecting or connected
    if (socket) {
      if (socket.readyState === WebSocket.OPEN) {
        log('warn', 'Already connected')
        return
      } else if (socket.readyState === WebSocket.CONNECTING) {
        log('warn', 'Connection already in progress')
        return
      }
    }
    
    // Clear any existing reconnect timeout
    if (reconnectTimeoutRef.current !== null) {
      window.clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    
    if (terminal) {
      terminal.writeln('Connecting to WebSocket...')
    }
    
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
        
        if (terminal) {
          terminal.writeln('\r\nWebSocket connected')
          terminal.writeln('Terminal ready\r\n')
        }
        
        setConnected(true)
        setSocket(ws)
        if (onConnectionChange) onConnectionChange(true)
        
        // Send initial resize after a brief delay
        setTimeout(() => {
          sendResize()
          startPingInterval()
        }, 300)
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
          // Handle binary data (old protocol)
          if (event.data instanceof ArrayBuffer) {
            const data = new Uint8Array(event.data)
            if (data.length === 0) {
              log('warn', 'Received empty binary message')
              return
            }
            
            const messageType = data[0]
            const payload = data.slice(1)
            
            switch (messageType) {
              case OUTPUT_MESSAGE:
                const content = new TextDecoder().decode(payload)
                if (terminal) {
                  terminal.write(content)
                }
                break
              case PONG_MESSAGE:
                missedHeartbeatsRef.current = 0
                log('info', 'Received pong')
                break
              default:
                log('warn', `Unknown message type: ${messageType}`)
            }
          }
          // Handle text data (JSON format - new protocol)
          else if (typeof event.data === 'string') {
            try {
              const data = JSON.parse(event.data)
              
              if (data.type === 'config') {
                log('info', 'Received terminal config')
              } else if (data.content) {
                if (terminal) {
                  terminal.write(data.content)
                }
              }
            } catch (e) {
              // Not JSON, try to display as raw text
              if (terminal) {
                terminal.write(event.data)
              }
            }
          }
        } catch (error) {
          log('error', `Error handling message: ${error}`)
        }
      }
      
      ws.onclose = (event) => {
        clearTimeout(connectionTimeout)
        log('info', `WebSocket connection closed: ${event.code} ${event.reason || ''}`)
        setConnected(false)
        setSocket(null)
        if (onConnectionChange) onConnectionChange(false)
        
        // Clear ping interval
        if (pingIntervalRef.current) {
          clearInterval(pingIntervalRef.current)
          pingIntervalRef.current = null
        }
        
        // Don't attempt to reconnect for certain close codes
        const noReconnectCodes = [1000, 1008] // Normal close, Policy violation
        if (noReconnectCodes.includes(event.code)) {
          if (event.code === 1008) {
            // Rate limiting - don't automatically reconnect
            log('warn', 'Connection closed due to policy violation (rate limiting)')
            if (terminal) {
              terminal.writeln('\r\nConnection closed due to server policy (rate limiting).')
              terminal.writeln('Please wait at least 60 seconds before trying again.')
            }
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
        
        if (terminal) {
          terminal.writeln(`\r\nConnection lost. Reconnect attempt ${reconnectAttemptsRef.current} in ${Math.round(reconnectDelay/1000)} seconds...`)
        }
        
        log('info', `Scheduling reconnect in ${reconnectDelay/1000}s (attempt ${reconnectAttemptsRef.current})`)
        
        // Schedule reconnection
        reconnectTimeoutRef.current = window.setTimeout(() => {
          reconnectTimeoutRef.current = null
          
          // Limit total reconnection attempts
          if (reconnectAttemptsRef.current > 10) {
            log('warn', 'Maximum reconnection attempts reached')
            if (terminal) {
              terminal.writeln('\r\nMaximum reconnection attempts reached. Please refresh the page to try again.')
            }
            return
          }
          
          // Only try to reconnect if we're not already connected
          connectWebSocket()
        }, reconnectDelay)
      }
      
      ws.onerror = (error) => {
        log('error', `WebSocket error: ${error}`)
        if (terminal) {
          terminal.writeln('\r\nConnection error. See console for details.\r\n')
        }
        // No need to close the socket here, the onclose handler will fire
      }
      
      setSocket(ws)
    } catch (error) {
      log('error', `Failed to create WebSocket: ${error}`)
      if (terminal) {
        terminal.writeln(`\r\nFailed to connect: ${error}\r\n`)
      }
      
      // Schedule a retry after a delay
      const retryDelay = 3000 // 3 seconds
      reconnectTimeoutRef.current = window.setTimeout(() => {
        reconnectTimeoutRef.current = null
        connectWebSocket()
      }, retryDelay)
    }
  }, [
    socket, 
    terminal, 
    instanceName, 
    onConnectionChange, 
    onMessageReceived, 
    sendResize, 
    startPingInterval, 
    log
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
      
      // Clear ping interval
      if (pingIntervalRef.current) {
        clearInterval(pingIntervalRef.current)
        pingIntervalRef.current = null
      }
      
      // Close socket if it's still open
      if (socket) {
        // Send graceful close message if possible
        if (socket.readyState === WebSocket.OPEN) {
          try {
            // Try to send a close message to the server
            sendMessage(CLOSE_MESSAGE, new Uint8Array(0))
            
            // Give it a moment to send before actually closing
            setTimeout(() => {
              socket.close(1000, "Terminal component unmounting")
            }, 100)
          } catch (err) {
            // Just close directly if sending fails
            socket.close()
          }
        } else {
          // If not open, just close it
          socket.close()
        }
      }
      
      // Update state
      setConnected(false)
      if (onConnectionChange) onConnectionChange(false)
    }
  }, [
    terminal, 
    instanceName, 
    connectWebSocket,
    socket,
    sendMessage,
    onConnectionChange
  ])
  
  return (
    <div className="terminal-wrapper" style={{ width: '100%', height: '100%' }}>
      <div
        ref={terminalRef}
        className="xterm-container"
        style={{ width: '100%', height: '100%' }}
      />
    </div>
  )
}

export default Terminal