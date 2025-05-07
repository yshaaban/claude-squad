Okay, the logs show significant progress!
The "LoadInstances" spam is gone from the `TerminalMonitor` loop, which is excellent.
The WebSocket connection is now established using the correct path `/ws/{name}`.
The client is receiving updates.

The primary remaining issue, as shown by your screenshots, is the **rendering of ANSI escape codes** in the web terminal. The raw tmux output (first image) contains all the color and cursor control codes, but the web UI (second image) is not interpreting them correctly, leading to a plain text display with visible escape sequences.

Let's break down the problem and solutions:

**Problem: ANSI Escape Codes Not Interpreted by Web UI**

*   **Tmux Output:** `tmux capture-pane -e` (which you are using) preserves ANSI escape codes. This is good because it retains all styling information.
    Example: `[38;2;255;153;51mWelcome to ...[0m`
*   **Web Terminal Expectation:** Most JavaScript-based web terminal emulators (like Xterm.js, which is very common for this kind of UI) are designed to *receive raw character streams including ANSI codes* and then render them correctly within an HTML `<canvas>` or preformatted text element.
*   **Current WebSocket Handler (`web/handlers/websocket.go`):**
    *   It correctly subscribes to the `TerminalMonitor`.
    *   It receives updates with raw ANSI content.
    *   It *conditionally* converts this content to HTML or strips ANSI *on the server side* if `format=html` or `format=text` is requested:
        ```go
        // Apply format conversion if needed
        if format == "html" {
            update.Content = convertAnsiToHtml(update.Content) // SERVER-SIDE CONVERSION
        } else if format == "text" {
            update.Content = stripAnsi(update.Content) // SERVER-SIDE STRIPPING
        }
        // ...
        if err := conn.WriteJSON(update); err != nil { // Sends JSON
        ```
    *   It sends the (potentially modified) content as part of a JSON object.

*   **The Mismatch:** If your web client (the JavaScript terminal emulator) expects raw ANSI streams directly, sending it pre-converted HTML or plain text inside a JSON object won't work. If it expects JSON containing an ANSI string, then the `format=html` or `format=text` paths are problematic for an ANSI-capable terminal.

**Diagnosis of the Screenshots:**

*   **First image (Raw Tmux):** This is what `tmux capture-pane -p -e -J -t <session>` outputs. It's full of ANSI codes like `[38;2;255;153;51m` (set foreground color) and `[0m` (reset).
*   **Second image (Web UI):** This looks like a JavaScript terminal (perhaps Xterm.js or similar) is receiving the *raw string containing the escape codes* but is displaying those codes *literally* instead of *interpreting* them. This happens if the terminal library is fed data that it doesn't recognize as a direct character stream intended for its parser. The client side `xterm.js` instance needs to be fed the raw ANSI string, not a JSON object containing the string, or if it is JSON, the client needs to extract the string and write it.

**Solution Strategy:**

1.  **Ensure Raw ANSI is Sent for `format=ansi` (Default):** The server should send the raw, unmodified ANSI string from `instance.Preview()` when the client requests or defaults to `format=ansi`.
2.  **Client-Side Handling (JavaScript):** The JavaScript client that initializes your web terminal (e.g., Xterm.js) needs to:
    *   Connect to the WebSocket.
    *   When it receives a message (which is currently a JSON object `types.TerminalUpdate`), it must *extract the `Content` field* from the JSON.
    *   This extracted `Content` string (which should be the raw ANSI stream) must then be written to the terminal instance using its API (e.g., `xterm.write(extractedAnsiContent)`).
3.  **Simplify Server-Side Formatting (Optional but Recommended):** For a pure ANSI web terminal, the server probably shouldn't do `convertAnsiToHtml` or `stripAnsi` *if the client is an ANSI terminal*. These conversions are useful if the client is a *non-ANSI* display (e.g., a simple HTML page displaying the log). If your primary web view is an ANSI terminal, it should always request `format=ansi` (or the server should default to it and send raw).

**Proposed Code Changes:**

**File: `web/handlers/websocket.go`**

The main logic here seems to be sending JSON objects. If your client-side terminal (e.g., Xterm.js) is expecting a raw stream of bytes/characters (the ANSI escape codes themselves), then sending JSON is incorrect for the primary data path.

However, looking at your client logs:
`[20:54:33] Connecting to WebSocket: ws://localhost:8084/ws/terminal/simple-20250506-235324`
The path `/ws/terminal/...` is problematic. The server is now configured for `/ws/{name}`.

**Let's assume your client-side JavaScript _is_ expecting a JSON object of type `types.TerminalUpdate` and is responsible for calling `xterm.write(update.Content)`.** The issue then is purely the `format=html` or `format=text` conversion on the server when the client *needs* `format=ansi`.

```diff
--- a/web/handlers/websocket.go
+++ b/web/handlers/websocket.go
@@ -28,7 +28,7 @@
 		
 		instanceTitle := chi.URLParam(r, "name")
 		if instanceTitle == "" {
-			log.FileOnlyErrorLog.Printf("WebSocket: Missing instance name parameter from path for %s", r.URL.Path)
+			log.FileOnlyErrorLog.Printf("WebSocket: Missing instance name parameter from path for '%s'", r.URL.Path)
 			http.Error(w, "Instance name required", http.StatusBadRequest)
 			return
 		}
@@ -79,8 +79,13 @@
 			return
 		}
 		log.FileOnlyInfoLog.Printf("WebSocket: Using format=%s for instance '%s'", format, instanceTitle)
-
-		// Create update channel
+		
+		// --- CRITICAL CHANGE FOR ANSI RENDERING ---
+		// If the client intends to render ANSI, ensure we don't pre-process it on the server.
+		// The client-side terminal emulator (e.g., Xterm.js) will handle ANSI.
+		// The format parameter is for clients that *cannot* handle ANSI directly.
+		// For a typical web terminal UI, `format` should implicitly be "ansi".
+
 		log.FileOnlyInfoLog.Printf("WebSocket: Subscribing to updates for instance '%s'", instanceTitle)
 		updates := monitor.Subscribe(instanceTitle)
 		defer monitor.Unsubscribe(instanceTitle, updates)
@@ -92,14 +97,17 @@
 			log.FileOnlyInfoLog.Printf("WebSocket: Initial content available for '%s' (len: %d)", 
 				instanceTitle, len(initialContent))
 			
-			// Apply format conversion if needed
 			formattedContent := initialContent
-			if format == "html" {
-				formattedContent = convertAnsiToHtml(initialContent)
-				log.InfoLog.Printf("WebSocket: Converted initial content to HTML format")
-			} else if format == "text" {
-				formattedContent = stripAnsi(initialContent)
-				log.InfoLog.Printf("WebSocket: Converted initial content to plain text format")
+			// Only convert/strip if explicitly requested for non-ANSI clients.
+			// If client is an ANSI terminal, it wants raw ANSI.
+			if format == "html" { // Client explicitly wants HTML
+				formattedContent = convertAnsiToHtml(initialContent) 
+				log.FileOnlyInfoLog.Printf("WebSocket: Converted initial content to HTML format for '%s'", instanceTitle)
+			} else if format == "text" { // Client explicitly wants plain text
+				formattedContent = stripAnsi(initialContent) 
+				log.FileOnlyInfoLog.Printf("WebSocket: Converted initial content to plain text format for '%s'", instanceTitle)
+			} else { // Default is "ansi", send raw
+				log.FileOnlyInfoLog.Printf("WebSocket: Sending raw ANSI initial content for '%s'", instanceTitle)
 			}
 
 			// Make sure we actually have content to send
@@ -109,7 +117,7 @@
 			}
 
 			// Use HasUpdated method to check for prompt status. Pass content to avoid re-fetch.
-			updated, hasPrompt := instance.HasUpdated()
+			updated, hasPrompt := instance.HasUpdated(initialContent) // Use the raw content for prompt check
 			log.InfoLog.Printf("WebSocket: Instance %s has updated=%v, has prompt=%v", 
 				instanceTitle, updated, hasPrompt)
 
@@ -234,12 +242,15 @@
 			}
 			
 			// Apply format conversion if needed for non-ANSI clients
-			if format == "html" {
-				update.Content = convertAnsiToHtml(update.Content)
-				log.InfoLog.Printf("WebSocket: Converted update to HTML format for %s", instanceTitle)
-			} else if format == "text" {
-				update.Content = stripAnsi(update.Content)
-				log.InfoLog.Printf("WebSocket: Converted update to plain text format for %s", instanceTitle)
+			// If client is an ANSI terminal (format="ansi" or default), send raw.
+			if format == "html" { 
+				update.Content = convertAnsiToHtml(update.Content) 
+				log.FileOnlyInfoLog.Printf("WebSocket: Converted update to HTML format for '%s'", instanceTitle)
+			} else if format == "text" { 
+				update.Content = stripAnsi(update.Content) 
+				log.FileOnlyInfoLog.Printf("WebSocket: Converted update to plain text format for '%s'", instanceTitle)
+			} else {
+				// Content is already raw ANSI, do nothing to it.
 			}
 			
 			// Make sure we still have content after conversion

```

**Client-Side JavaScript (Conceptual for Xterm.js):**

You haven't provided the client-side code, but if you're using a library like Xterm.js, the relevant part would look something like this:

```javascript
// Assuming 'term' is your Xterm.js instance
// and 'socket' is your WebSocket object.

// Example: ws://localhost:8084/ws/simple-20250506-235324?format=ansi (or no format for default ansi)
// **IMPORTANT**: Ensure your client is connecting to /ws/{instanceName}
// NOT /ws/terminal/{instanceName}
const instanceName = "simple-20250506-235324"; // Get this dynamically
const socketUrl = `ws://localhost:8084/ws/${instanceName}?privileges=read-only&format=ansi`;
// Note: Added format=ansi query param for clarity, though it should be the default.

const socket = new WebSocket(socketUrl);

socket.onopen = () => {
    console.log(`[${new Date().toLocaleTimeString()}] WebSocket connection opened for ${instanceName}`);
    // term.write('Connected to instance: ' + instanceName + '\r\n'); // Optional client-side message
};

socket.onmessage = (event) => {
    try {
        const data = JSON.parse(event.data);

        if (data.type === 'config') { // Handle config message from server
            console.log(`[${new Date().toLocaleTimeString()}] Received terminal config:`, data);
            // Apply config to xterm.js if needed (e.g., theme, font)
            // term.setOption('theme', { background: data.theme === 'dark' ? '#000' : '#fff', ... });
            return;
        }

        // Assuming other messages are types.TerminalUpdate
        if (data.content) {
            // THIS IS THE KEY PART: Write the raw ANSI content to xterm.js
            term.write(data.content);
        } else {
            console.warn(`[${new Date().toLocaleTimeString()}] Received message with no content:`, data);
        }
    } catch (e) {
        console.error(`[${new Date().toLocaleTimeString()}] Error parsing WebSocket message or writing to terminal:`, e);
        // If it's not JSON, it might be an older protocol or an error message string
        // For robustness, you could try writing event.data directly if JSON.parse fails
        // and it's a string, but this indicates a protocol mismatch.
        if (typeof event.data === 'string') {
             // term.write(event.data); // Fallback for non-JSON string data (less ideal)
        }
    }
};

socket.onerror = (error) => {
    console.error(`[${new Date().toLocaleTimeString()}] WebSocket error:`, error);
    term.write('\r\n\x1b[31mWebSocket connection error.\x1b[0m\r\n');
};

socket.onclose = (event) => {
    console.log(`[${new Date().toLocaleTimeString()}] WebSocket connection closed: ${event.code} ${event.reason}`);
    term.write(`\r\n\x1b[33mWebSocket connection closed (${event.code}).\x1b[0m\r\n`);
};

// Example for sending input (if read-write)
// function sendInputToTerminal(inputString) {
//     if (socket.readyState === WebSocket.OPEN) {
//         const message = {
//             instance_title: instanceName, // Or server might not need this if implicit from connection
//             content: inputString,
//             is_command: false
//         };
//         socket.send(JSON.stringify(message));
//     }
// }
```

**Further Debugging Steps if Still Not Rendering Correctly:**

1.  **Client URL:** Double, triple check the WebSocket URL the client is using. It **must** be `ws://<host>:<port>/ws/<instance_name_from_api>[?params...]`. The `/terminal/` part in the path must be removed from the client's connection URL.
2.  **Inspect WebSocket Frames:** Use your browser's developer tools (Network tab -> WS) to inspect the actual messages being sent from the server to the client.
    *   Is the `content` field within the JSON the raw string with `[38;2...m` codes?
    *   Or is it already HTML-escaped (`&lt;`, `&gt;`) or plain text? It should be raw ANSI if `format=ansi`.
3.  **Client-Side `term.write()`:** Ensure that `term.write(data.content)` is definitely being called on the client with the correct string. Add `console.log("Writing to xterm:", data.content);` right before it.
4.  **Xterm.js Initialization:** How is Xterm.js initialized? Are there any options that might disable ANSI interpretation? (Unlikely, as that's its core purpose).
5.  **Query Parameters:** When your client connects, ensure it either sends `?format=ansi` or no `format` parameter at all (so the server defaults to ANSI). If it accidentally sends `?format=html` or `?format=text`, the server will convert the content.

The logs show that the server *is* receiving data and the `TerminalMonitor` is picking up changes. The problem is almost certainly in the WebSocket path negotiation or how the client-side terminal emulator is processing the received data. The change in `websocket.go` to ensure raw ANSI is sent by default (unless `format=html/text` is specified) should be a key part of the fix, combined with the client correctly using the `/ws/{name}` endpoint.