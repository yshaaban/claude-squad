#!/bin/bash

# Color output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color
BLUE='\033[0;34m'

# Print a colorized message
print_colored() {
  echo -e "${GREEN}$1${NC}"
}

# Print error message
print_error() {
  echo -e "${RED}$1${NC}" 
}

# Print info message
print_info() {
  echo -e "${BLUE}$1${NC}"
}

# First make the script executable
chmod +x "$0"

print_colored "Starting robust React frontend test for Claude Squad..."

# Check if port 8086 is already in use
if lsof -i:8086 | grep LISTEN >/dev/null 2>&1; then
  print_error "Port 8086 is already in use. Please close the application using it and try again."
  exit 1
fi

# Check if the dist directory exists
if [ ! -d "./web/static/dist" ]; then
  print_error "React build not found at ./web/static/dist"
  print_info "Building React frontend first..."
  
  # Check if build_frontend.sh exists and run it
  if [ -f "./build_frontend.sh" ]; then
    ./build_frontend.sh
  else
    print_error "build_frontend.sh not found. Please build the React frontend manually."
    exit 1
  fi
fi

# Create test files in the dist directory to help with debugging
print_colored "Creating test HTML files for debugging..."

# 1. Create a simple test.html file
cat > ./web/static/dist/test.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
  <title>Simple Test Page</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 40px; }
    .success { color: green; }
    .failure { color: red; }
  </style>
</head>
<body>
  <h1>Simple Test Page</h1>
  <p>If you can see this page, basic static file serving is working.</p>
  
  <h2>Asset Loading Test</h2>
  <p>Testing image with relative path:</p>
  <div>
    <img src="./test-image.png" alt="Test image (relative)" width="100" 
         onerror="this.parentNode.innerHTML='<span class=\'failure\'>Failed to load ./test-image.png</span>'" 
         onload="this.parentNode.innerHTML='<span class=\'success\'>Successfully loaded ./test-image.png</span>'">
  </div>
  
  <p>Testing image with absolute path:</p>
  <div>
    <img src="/test-image.png" alt="Test image (absolute)" width="100"
         onerror="this.parentNode.innerHTML='<span class=\'failure\'>Failed to load /test-image.png</span>'" 
         onload="this.parentNode.innerHTML='<span class=\'success\'>Successfully loaded /test-image.png</span>'">
  </div>
  
  <h2>WebSocket Test</h2>
  <button onclick="testWebSocket()">Test WebSocket Connection</button>
  <div id="ws-result"></div>
  
  <script>
    function testWebSocket() {
      const resultDiv = document.getElementById('ws-result');
      resultDiv.innerHTML = 'Attempting WebSocket connection...';
      
      // Create WebSocket connection
      const ws = new WebSocket(`ws://${window.location.host}/ws?instance=test`);
      
      ws.onopen = () => {
        resultDiv.innerHTML = '<span class="success">WebSocket connection successful!</span>';
        ws.send(JSON.stringify({ type: 'ping' }));
        setTimeout(() => ws.close(), 1000);
      };
      
      ws.onerror = (error) => {
        resultDiv.innerHTML = `<span class="failure">WebSocket connection failed: ${error}</span>`;
      };
      
      ws.onclose = (event) => {
        if (event.code !== 1000) {
          resultDiv.innerHTML += `<br><span class="failure">WebSocket closed with code: ${event.code}. Reason: ${event.reason}</span>`;
        } else {
          resultDiv.innerHTML += '<br><span class="success">WebSocket closed normally</span>';
        }
      };
    }
  </script>
</body>
</html>
EOF

# 2. Create a test-image.png (1x1 pixel transparent PNG)
echo -n -e "\x89\x50\x4E\x47\x0D\x0A\x1A\x0A\x00\x00\x00\x0D\x49\x48\x44\x52\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1F\x15\xC4\x89\x00\x00\x00\x0A\x49\x44\x41\x54\x78\x9C\x63\x00\x01\x00\x00\x05\x00\x01\x0D\x0A\x2D\xB4\x00\x00\x00\x00\x49\x45\x4E\x44\xAE\x42\x60\x82" > ./web/static/dist/test-image.png

# 3. Create a test for the react app's asset paths
cat > ./web/static/dist/asset-test.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
  <title>Asset Path Test</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 40px; }
    .success { color: green; }
    .failure { color: red; }
    table { border-collapse: collapse; width: 100%; margin-top: 20px; }
    td, th { border: 1px solid #ddd; padding: 8px; text-align: left; }
    th { background-color: #f2f2f2; }
  </style>
</head>
<body>
  <h1>Asset Path Test</h1>
  <p>This page tests various asset path formats to diagnose serving issues.</p>
  
  <table>
    <thead>
      <tr>
        <th>Path Format</th>
        <th>URL</th>
        <th>Status</th>
      </tr>
    </thead>
    <tbody id="results">
      <!-- Will be filled by JavaScript -->
    </tbody>
  </table>
  
  <h2>Request Headers</h2>
  <pre id="headers"></pre>
  
  <script>
    // Asset path formats to test
    const pathTests = [
      { name: 'Absolute path', url: '/assets/test-asset.txt' },
      { name: 'Relative path with ./', url: './assets/test-asset.txt' },
      { name: 'Relative path without ./', url: 'assets/test-asset.txt' },
      { name: 'Parent relative path', url: '../assets/test-asset.txt' },
      { name: 'Double slash path', url: '//assets/test-asset.txt' },
      { name: 'Favicon absolute', url: '/favicon.svg' },
      { name: 'Favicon relative', url: './favicon.svg' }
    ];
    
    // Create a test asset file
    function createTestAsset() {
      const testContent = 'This is a test asset file';
      
      fetch('/api/create-test-asset', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: testContent })
      }).catch(err => console.error('Failed to create test asset:', err));
    }
    
    // Test a specific asset path
    async function testPath(pathTest) {
      try {
        const response = await fetch(pathTest.url, { method: 'HEAD' });
        return {
          name: pathTest.name,
          url: pathTest.url,
          status: response.status,
          ok: response.ok
        };
      } catch (error) {
        return {
          name: pathTest.name,
          url: pathTest.url,
          status: 'Error',
          ok: false,
          error: error.toString()
        };
      }
    }
    
    // Run all the tests
    async function runTests() {
      createTestAsset();
      
      const results = document.getElementById('results');
      results.innerHTML = '';
      
      // Show request headers
      const headerResponse = await fetch('/test.html');
      const headers = document.getElementById('headers');
      headers.textContent = 'Request Headers used by browser:\n';
      
      const headerLines = [];
      for (const [key, value] of Object.entries(getAllRequestHeaders())) {
        headerLines.push(`${key}: ${value}`);
      }
      headers.textContent += headerLines.join('\n');
      
      // Run each path test
      for (const pathTest of pathTests) {
        const row = document.createElement('tr');
        row.innerHTML = `
          <td>${pathTest.name}</td>
          <td>${pathTest.url}</td>
          <td>Testing...</td>
        `;
        results.appendChild(row);
        
        const result = await testPath(pathTest);
        
        row.cells[2].innerHTML = result.ok 
          ? `<span class="success">OK (${result.status})</span>` 
          : `<span class="failure">Failed (${result.status}${result.error ? ': ' + result.error : ''})</span>`;
      }
    }
    
    // Helper to get all request headers
    function getAllRequestHeaders() {
      // This is a best-effort approximation since browsers limit access to request headers
      return {
        'User-Agent': navigator.userAgent,
        'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
        'Accept-Language': navigator.language || 'en-US',
        'Host': window.location.host,
        'Origin': window.location.origin,
        'Referer': window.location.href,
        'Connection': 'keep-alive',
        'Upgrade-Insecure-Requests': '1'
      };
    }
    
    // Run tests when page loads
    window.onload = runTests;
  </script>
</body>
</html>
EOF

# Create a simple test asset
mkdir -p ./web/static/dist/assets
echo "This is a test asset file" > ./web/static/dist/assets/test-asset.txt

# Create a simple favicon
cat > ./web/static/dist/favicon.svg << 'EOF'
<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32">
  <rect width="32" height="32" fill="#4CAF50" />
  <text x="8" y="22" fill="white" font-family="Arial" font-size="20">C</text>
</svg>
EOF

# Check if the app exists and run it with React frontend
if [ -f "./cs" ]; then
  print_colored "Starting Claude Squad with React web UI..."
  print_colored "This is a robust test with additional diagnostic tools!"
  
  print_colored "Access these test URLs:"
  print_info "  http://localhost:8086/test.html     - Simple test page with WebSocket test"
  print_info "  http://localhost:8086/asset-test.html - Test asset loading with different paths"
  print_info "  http://localhost:8086/              - React frontend"
  print_colored ""
  
  print_colored "Press Ctrl+C to stop"
  print_colored ""
  
  # Run Claude Squad with React frontend and modifications to fix rate limiting
  # Add --simple to avoid TTY requirements
  ./cs --web --react --web-port=8086 --log-to-file --simple
else
  print_error "Claude Squad executable not found. Make sure you've built the project."
  exit 1
fi