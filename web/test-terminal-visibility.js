// Terminal visibility test
// This script helps validate if the terminal is properly showing content in the web UI

// This is a browser-compatible script that can be pasted in the browser console
// to validate terminal visibility in the web UI

(function() {
  console.log('Running terminal visibility test...');
  
  // Check if terminal elements exist
  const terminalContainer = document.getElementById('terminal-container');
  if (!terminalContainer) {
    console.error('Terminal container element not found!');
    return false;
  }
  
  console.log('Terminal container found:', terminalContainer);
  
  // Check if xterm.js elements exist
  const xtermElements = document.getElementsByClassName('xterm');
  if (xtermElements.length === 0) {
    console.warn('No xterm elements found - checking for fallback');
    
    // Check if fallback terminal is being used
    const fallbackTerminal = document.getElementById('terminal-fallback');
    if (!fallbackTerminal || getComputedStyle(fallbackTerminal).display === 'none') {
      console.error('Neither xterm nor fallback terminal is visible!');
      
      // Try toggling to fallback mode
      console.log('Attempting to switch to fallback mode...');
      const toggleButton = document.getElementById('toggle-mode-button');
      if (toggleButton) {
        toggleButton.click();
        console.log('Clicked fallback toggle button');
        
        // Wait and check again
        setTimeout(() => {
          const visibleNow = document.getElementById('terminal-fallback');
          if (visibleNow && getComputedStyle(visibleNow).display !== 'none') {
            console.log('SUCCESS: Fallback terminal is now visible!');
            
            // Verify it has content
            if (visibleNow.textContent && visibleNow.textContent.length > 10) {
              console.log('Fallback terminal has content:', visibleNow.textContent.substring(0, 100) + '...');
            } else {
              console.warn('Fallback terminal visible but has little or no content');
            }
          } else {
            console.error('Fallback terminal still not visible after toggling!');
          }
        }, 1000);
      } else {
        console.error('Toggle button not found, cannot switch to fallback mode');
      }
      
      return false;
    } else {
      console.log('Fallback terminal is visible!');
      
      // Verify it has content
      if (fallbackTerminal.textContent && fallbackTerminal.textContent.length > 10) {
        console.log('Fallback terminal has content:', fallbackTerminal.textContent.substring(0, 100) + '...');
        return true;
      } else {
        console.warn('Fallback terminal visible but has little or no content');
        return false;
      }
    }
  }
  
  console.log('xterm elements found:', xtermElements.length);
  
  // Check if xterm has visible content
  const xtermScreen = document.querySelector('.xterm-screen');
  if (!xtermScreen) {
    console.error('xterm-screen element not found!');
    return false;
  }
  
  console.log('xterm-screen found:', xtermScreen);
  
  // Check terminal dimensions
  const canvas = document.querySelector('.xterm-text-layer');
  if (canvas) {
    const width = canvas.clientWidth;
    const height = canvas.clientHeight;
    console.log(`Terminal dimensions: ${width}x${height}px`);
    
    if (width === 0 || height === 0) {
      console.error('Terminal has zero width or height!');
      return false;
    }
  } else {
    console.warn('xterm-text-layer not found, cannot check dimensions');
  }
  
  // Look for actual content (text)
  const textElements = document.querySelectorAll('.xterm-text-layer > span, .xterm-rows > div');
  let hasVisibleText = false;
  let contentExample = '';
  
  if (textElements.length > 0) {
    for (const el of textElements) {
      if (el.textContent && el.textContent.trim().length > 0) {
        hasVisibleText = true;
        contentExample = el.textContent;
        break;
      }
    }
    
    console.log(`Terminal text elements found: ${textElements.length}`);
    
    if (hasVisibleText) {
      console.log('Terminal has visible text content:', contentExample);
      return true;
    } else {
      console.warn('Terminal text elements exist but no visible text found');
      return false;
    }
  } else {
    console.warn('No terminal text elements found');
  }
  
  // Final check - see if any terminal rows are visible
  const rows = document.querySelectorAll('.xterm-rows > div');
  console.log(`Terminal rows found: ${rows.length}`);
  
  if (rows.length > 0) {
    return true;
  } else {
    console.error('No terminal rows found!');
    return false;
  }
})();