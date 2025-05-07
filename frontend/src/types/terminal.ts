// Terminal WebSocket message types
export interface WebSocketMessage {
  type: string
  payload: any
  timestamp?: number
}

// Terminal output from server
export interface TerminalUpdate {
  instanceTitle: string
  content: string
  timestamp: string
  status: string
  hasPrompt: boolean
}

// Terminal input to server
export interface TerminalInput {
  content: string
  isCommand: boolean
}

// Terminal resize message
export interface TerminalResize {
  cols: number
  rows: number
  isCommand: boolean
  content: string
}

// Terminal configuration from server
export interface TerminalConfig {
  type: 'config'
  privileges: 'read-only' | 'read-write'
  theme: 'dark' | 'light'
  fontFamily: string
  fontSize: number
}