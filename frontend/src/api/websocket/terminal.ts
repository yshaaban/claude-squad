import { Terminal } from 'xterm'
import { TerminalInput, TerminalResize, TerminalUpdate } from '@/types'

// Constants for binary messages
const OUTPUT_MESSAGE = 'o'.charCodeAt(0)
const INPUT_MESSAGE = 'i'.charCodeAt(0)
const RESIZE_MESSAGE = 'r'.charCodeAt(0)
const PING_MESSAGE = 'p'.charCodeAt(0)
const PONG_MESSAGE = 'P'.charCodeAt(0)
const CLOSE_MESSAGE = 'c'.charCodeAt(0)

export interface TerminalWebSocketOptions {
  instanceName: string
  terminal: Terminal
  onConnectionChange?: (connected: boolean) => void
  onError?: (error: Error) => void
}

export class TerminalWebSocket {
  private socket: WebSocket | null = null
  private terminal: Terminal
  private instanceName: string
  private connected = false
  private reconnecting = false
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000
  private pingInterval: number | null = null
  private missedHeartbeats = 0
  private maxMissedHeartbeats = 3
  private onConnectionChange?: (connected: boolean) => void
  private onError?: (error: Error) => void
  
  constructor(options: TerminalWebSocketOptions) {
    this.terminal = options.terminal
    this.instanceName = options.instanceName
    this.onConnectionChange = options.onConnectionChange
    this.onError = options.onError
  }
  
  /**
   * Connect to the WebSocket
   */
  connect() {
    if (this.connected || this.reconnecting) {
      return
    }
    
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws/${this.instanceName}?format=ansi`
    
    this.terminal.writeln(`\r\nConnecting to ${wsUrl}...`)
    
    try {
      this.socket = new WebSocket(wsUrl)
      
      this.socket.onopen = this.handleOpen.bind(this)
      this.socket.onmessage = this.handleMessage.bind(this)
      this.socket.onclose = this.handleClose.bind(this)
      this.socket.onerror = this.handleError.bind(this)
    } catch (error) {
      this.terminal.writeln(`\r\nError connecting: ${error}`)
      if (this.onError) {
        this.onError(error instanceof Error ? error : new Error(String(error)))
      }
    }
  }
  
  /**
   * Disconnect from the WebSocket
   */
  disconnect() {
    if (!this.connected || !this.socket) {
      return
    }
    
    // Stop ping interval
    this.stopPingInterval()
    
    // Send close message
    this.sendMessage(CLOSE_MESSAGE, new Uint8Array(0))
    
    // Close socket after a short delay
    setTimeout(() => {
      if (this.socket) {
        this.socket.close()
        this.socket = null
      }
      this.setConnected(false)
    }, 100)
  }
  
  /**
   * Send input to the terminal
   */
  sendInput(text: string) {
    if (!this.connected || !text) {
      return
    }
    
    // For now, we're supporting both formats - the old binary format for backward
    // compatibility, and the new JSON format
    try {
      // New JSON format
      const input: TerminalInput = {
        content: text,
        isCommand: false
      }
      
      this.socket?.send(JSON.stringify(input))
    } catch (error) {
      // Fallback to binary format
      const data = new TextEncoder().encode(text)
      this.sendMessage(INPUT_MESSAGE, data)
    }
  }
  
  /**
   * Send terminal resize event
   */
  sendResize(cols: number, rows: number) {
    if (!this.connected) {
      return
    }
    
    try {
      // New JSON format
      const resize: TerminalResize = {
        cols,
        rows,
        isCommand: true,
        content: 'resize'
      }
      
      this.socket?.send(JSON.stringify(resize))
    } catch (error) {
      // Fallback to binary format
      const resizeData = JSON.stringify({ cols, rows })
      this.sendMessage(RESIZE_MESSAGE, new TextEncoder().encode(resizeData))
    }
  }
  
  // Private methods
  
  private handleOpen() {
    this.terminal.writeln('\r\nConnected!')
    this.setConnected(true)
    this.startPingInterval()
  }
  
  private handleMessage(event: MessageEvent) {
    // Handle both binary and JSON formats
    if (event.data instanceof ArrayBuffer) {
      this.handleBinaryMessage(event.data)
    } else if (typeof event.data === 'string') {
      this.handleJsonMessage(event.data)
    }
  }
  
  private handleBinaryMessage(data: ArrayBuffer) {
    const buffer = new Uint8Array(data)
    if (buffer.length === 0) {
      return
    }
    
    const messageType = buffer[0]
    const content = buffer.length > 1 ? new TextDecoder().decode(buffer.slice(1)) : ''
    
    switch (messageType) {
      case OUTPUT_MESSAGE:
        this.terminal.write(content)
        break
      case PONG_MESSAGE:
        this.missedHeartbeats = 0
        break
      default:
        console.warn(`Unknown message type: ${messageType}`)
    }
  }
  
  private handleJsonMessage(data: string) {
    try {
      const message = JSON.parse(data)
      
      if (message.type === 'config') {
        // Config message - could apply settings to terminal
        console.log('Terminal config:', message)
      } else if (message.content) {
        // Terminal content
        this.terminal.write(message.content)
      }
    } catch (error) {
      console.error('Error parsing JSON message:', error)
      
      // Try to handle as plain text
      if (data.includes('\x1b[')) {
        this.terminal.write(data)
      }
    }
  }
  
  private handleClose(event: CloseEvent) {
    this.setConnected(false)
    this.stopPingInterval()
    
    this.terminal.writeln(`\r\nConnection closed: ${event.code} ${event.reason || ''}`)
    
    // Try to reconnect
    if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
      this.attemptReconnect()
    }
  }
  
  private handleError(event: Event) {
    console.error('WebSocket error:', event)
    this.terminal.writeln('\r\nConnection error')
    
    if (this.onError) {
      this.onError(new Error('WebSocket error'))
    }
  }
  
  private setConnected(connected: boolean) {
    this.connected = connected
    
    if (this.onConnectionChange) {
      this.onConnectionChange(connected)
    }
  }
  
  private attemptReconnect() {
    if (this.reconnecting) {
      return
    }
    
    this.reconnecting = true
    this.reconnectAttempts++
    
    const delay = this.reconnectDelay * this.reconnectAttempts
    this.terminal.writeln(`\r\nAttempting to reconnect in ${delay / 1000} seconds...`)
    
    setTimeout(() => {
      this.reconnecting = false
      this.connect()
    }, delay)
  }
  
  private startPingInterval() {
    this.stopPingInterval()
    
    this.pingInterval = window.setInterval(() => {
      if (this.missedHeartbeats >= this.maxMissedHeartbeats) {
        this.terminal.writeln(`\r\nMissed ${this.missedHeartbeats} heartbeats - reconnecting...`)
        
        // Force close and reconnect
        if (this.socket) {
          this.socket.close()
          this.socket = null
        }
        
        this.setConnected(false)
        this.stopPingInterval()
        
        // Try to reconnect
        setTimeout(() => {
          this.connect()
        }, 1000)
        
        return
      }
      
      this.missedHeartbeats++
      this.sendMessage(PING_MESSAGE, new Uint8Array(0))
    }, 15000) as unknown as number
  }
  
  private stopPingInterval() {
    if (this.pingInterval !== null) {
      clearInterval(this.pingInterval)
      this.pingInterval = null
    }
  }
  
  private sendMessage(type: number, data: Uint8Array) {
    if (!this.connected || !this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return
    }
    
    const message = new Uint8Array(data.length + 1)
    message[0] = type
    message.set(data, 1)
    
    try {
      this.socket.send(message)
    } catch (error) {
      console.error('Failed to send message:', error)
    }
  }
}