// Basic ANSI escape sequence processor for fallback rendering
// Note: For production, we'll use xterm.js directly which handles this much better

// Color map
const ANSI_COLORS = {
  // Basic colors (0-7)
  0: 'black',
  1: 'red',
  2: 'green',
  3: 'yellow',
  4: 'blue',
  5: 'magenta',
  6: 'cyan',
  7: 'white',
  
  // Bright colors (8-15)
  8: '#555', // bright black (gray)
  9: '#f55', // bright red
  10: '#5f5', // bright green
  11: '#ff5', // bright yellow
  12: '#55f', // bright blue
  13: '#f5f', // bright magenta
  14: '#5ff', // bright cyan
  15: '#fff'  // bright white
}

type AnsiStyle = {
  color?: string;
  backgroundColor?: string;
  fontWeight?: 'normal' | 'bold';
  fontStyle?: 'normal' | 'italic';
  textDecoration?: 'none' | 'underline';
}

/**
 * Process ANSI escape sequences in text and convert to HTML
 */
export function processAnsiToHtml(text: string): string {
  if (!text.includes('\x1b[')) {
    return escapeHtml(text)
  }
  
  let result = ''
  let currentStyle: AnsiStyle = {}
  let currentSpan = false
  let i = 0
  
  while (i < text.length) {
    // Look for escape sequence
    if (text[i] === '\x1b' && text[i + 1] === '[') {
      // Close any open span
      if (currentSpan) {
        result += '</span>'
        currentSpan = false
      }
      
      // Find end of escape sequence (a letter)
      let end = i + 2
      while (end < text.length && 
             !(text.charCodeAt(end) >= 65 && text.charCodeAt(end) <= 122)) {
        end++
      }
      
      if (end < text.length) {
        // Get the command and parameters
        const cmd = text.substring(i + 2, end + 1)
        const command = cmd.charAt(cmd.length - 1)
        const params = cmd.substring(0, cmd.length - 1).split(';')
        
        // Handle different commands
        if (command === 'm') { // SGR (Select Graphic Rendition)
          const cssStyle = processGraphicCommand(params, currentStyle)
          
          // Start a new span with the styles
          if (cssStyle) {
            result += `<span style="${cssStyle}">`
            currentSpan = true
          }
        }
        
        // Skip the processed escape sequence
        i = end + 1
      } else {
        // Incomplete sequence, skip the escape character
        i++
      }
    } else {
      // Regular character
      result += escapeHtml(text[i])
      i++
    }
  }
  
  // Close any open span
  if (currentSpan) {
    result += '</span>'
  }
  
  return result
}

/**
 * Process SGR (graphic) commands and return CSS styles
 */
function processGraphicCommand(params: string[], currentStyle: AnsiStyle): string {
  // Reset or empty
  if (params.length === 0 || params[0] === '0' || params[0] === '') {
    // Reset all attributes
    currentStyle = {}
    return 'color: inherit; background-color: inherit; font-weight: normal; font-style: normal; text-decoration: none;'
  }
  
  let styles = ''
  
  for (let i = 0; i < params.length; i++) {
    const param = parseInt(params[i], 10)
    
    switch (param) {
      case 1: // Bold
        currentStyle.fontWeight = 'bold'
        styles += 'font-weight: bold; '
        break
      case 3: // Italic
        currentStyle.fontStyle = 'italic'
        styles += 'font-style: italic; '
        break
      case 4: // Underline
        currentStyle.textDecoration = 'underline'
        styles += 'text-decoration: underline; '
        break
      case 22: // Not bold
        currentStyle.fontWeight = 'normal'
        styles += 'font-weight: normal; '
        break
      case 23: // Not italic
        currentStyle.fontStyle = 'normal'
        styles += 'font-style: normal; '
        break
      case 24: // Not underlined
        currentStyle.textDecoration = 'none'
        styles += 'text-decoration: none; '
        break
      // Basic foreground colors (30-37)
      case 30:
      case 31:
      case 32:
      case 33:
      case 34:
      case 35:
      case 36:
      case 37:
        const color = ANSI_COLORS[param - 30]
        currentStyle.color = color
        styles += `color: ${color}; `
        break
      case 39: // Default foreground color
        currentStyle.color = undefined
        styles += 'color: inherit; '
        break
      // Basic background colors (40-47)
      case 40:
      case 41:
      case 42:
      case 43:
      case 44:
      case 45:
      case 46:
      case 47:
        const bgColor = ANSI_COLORS[param - 40]
        currentStyle.backgroundColor = bgColor
        styles += `background-color: ${bgColor}; `
        break
      case 49: // Default background color
        currentStyle.backgroundColor = undefined
        styles += 'background-color: inherit; '
        break
      // Bright foreground colors (90-97)
      case 90:
      case 91:
      case 92:
      case 93:
      case 94:
      case 95:
      case 96:
      case 97:
        const brightColor = ANSI_COLORS[(param - 90) + 8]
        currentStyle.color = brightColor
        styles += `color: ${brightColor}; `
        break
      // Bright background colors (100-107)
      case 100:
      case 101:
      case 102:
      case 103:
      case 104:
      case 105:
      case 106:
      case 107:
        const brightBgColor = ANSI_COLORS[(param - 100) + 8]
        currentStyle.backgroundColor = brightBgColor
        styles += `background-color: ${brightBgColor}; `
        break
      // Extended colors
      case 38:
      case 48:
        // Check if there are enough parameters
        if (i + 1 < params.length) {
          const colorMode = parseInt(params[i + 1], 10)
          
          if (colorMode === 5 && i + 2 < params.length) {
            // 8-bit color (256 colors)
            const colorIndex = parseInt(params[i + 2], 10)
            let colorValue: string
            
            if (colorIndex < 16) {
              // Basic colors
              colorValue = ANSI_COLORS[colorIndex]
            } else if (colorIndex >= 232) {
              // Grayscale
              const level = Math.round(((colorIndex - 232) / 23) * 255)
              colorValue = `rgb(${level},${level},${level})`
            } else {
              // 6×6×6 color cube
              const adjustedIndex = colorIndex - 16
              const r = Math.floor(adjustedIndex / 36) * 51
              const g = Math.floor((adjustedIndex % 36) / 6) * 51
              const b = (adjustedIndex % 6) * 51
              colorValue = `rgb(${r},${g},${b})`
            }
            
            if (param === 38) {
              // Foreground
              currentStyle.color = colorValue
              styles += `color: ${colorValue}; `
            } else {
              // Background
              currentStyle.backgroundColor = colorValue
              styles += `background-color: ${colorValue}; `
            }
            
            i += 2 // Skip the next two parameters
          } else if (colorMode === 2 && i + 4 < params.length) {
            // 24-bit RGB color
            const r = parseInt(params[i + 2], 10)
            const g = parseInt(params[i + 3], 10)
            const b = parseInt(params[i + 4], 10)
            const colorValue = `rgb(${r},${g},${b})`
            
            if (param === 38) {
              // Foreground
              currentStyle.color = colorValue
              styles += `color: ${colorValue}; `
            } else {
              // Background
              currentStyle.backgroundColor = colorValue
              styles += `background-color: ${colorValue}; `
            }
            
            i += 4 // Skip the next four parameters
          }
        }
        break
    }
  }
  
  return styles
}

/**
 * Escape HTML entities to prevent XSS
 */
function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
}

/**
 * Strip all ANSI escape sequences from a string
 */
export function stripAnsi(text: string): string {
  return text.replace(/\x1b\[[0-9;]*[a-zA-Z]/g, '')
}