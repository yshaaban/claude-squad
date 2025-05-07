// Instance data
export interface Instance {
  title: string
  status: string
  path?: string
  createdAt: string
  updatedAt: string
  program?: string
  inPlace?: boolean
  diffStats?: DiffStats
}

// Detailed instance data including terminal status
export interface InstanceDetail extends Instance {
  hasPrompt: boolean
  tmuxSession?: string
}

// Diff statistics for an instance
export interface DiffStats {
  added: number
  removed: number
}

// Terminal output for an instance
export interface InstanceOutput {
  content: string
  format: 'ansi' | 'html' | 'text'
  timestamp: string
  hasPrompt: boolean
}

// Server response for instance list
export interface InstancesResponse {
  instances: Instance[]
}