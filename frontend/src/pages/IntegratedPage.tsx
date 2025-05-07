import { useEffect, useState, useCallback, useRef } from 'react'
import Terminal from '@/components/terminal/Terminal'
import { Instance, InstancesResponse } from '@/types/instance'

const API_BASE_URL = '/api'

// API function with proper retry and backoff logic
const fetchInstancesApi = async (): Promise<InstancesResponse> => {
  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), 10000)
  
  try {
    const response = await fetch(`${API_BASE_URL}/instances`, {
      signal: controller.signal,
      headers: {
        'Cache-Control': 'no-cache',
      }
    })
    
    clearTimeout(timeoutId)
    
    if (!response.ok) {
      // Don't retry rate limiting errors, just return empty response
      if (response.status === 429) {
        console.warn('Rate limited on API request. Using cached data.')
        return { instances: [] } 
      }
      
      throw new Error(`Failed to fetch instances: ${response.statusText}`)
    }
    
    return response.json()
  } catch (error) {
    clearTimeout(timeoutId)
    
    if (error instanceof Error) {
      // Don't retry on abort errors
      if (error.name === 'AbortError') {
        console.warn('Request timed out')
        return { instances: [] }
      }
    }
    
    // For network errors, throw to let caller handle it
    throw error
  }
}

const IntegratedPage = () => {
  const [instances, setInstances] = useState<Instance[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date())
  const [selectedInstance, setSelectedInstance] = useState<string | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  
  // Use refs for tracking state that shouldn't trigger re-renders
  const fetchIntervalRef = useRef<number | null>(null)
  const fetchInProgressRef = useRef(false)
  const errorCountRef = useRef(0)
  const lastSuccessfulFetchRef = useRef<Date>(new Date())
  
  // Fetch instances with rate limiting and error count awareness
  const fetchInstances = useCallback(async () => {
    // Prevent concurrent fetches
    if (fetchInProgressRef.current) {
      console.log('Fetch already in progress, skipping')
      return
    }
    
    // Skip fetching if we've had too many consecutive errors
    if (errorCountRef.current > 5) {
      setError('Too many consecutive errors. Please refresh the page.')
      return
    }
    
    // Check if we've fetched recently (within 5 seconds)
    const timeSinceLastFetch = Date.now() - lastSuccessfulFetchRef.current.getTime()
    if (timeSinceLastFetch < 5000 && instances.length > 0) {
      console.log('Fetched recently, skipping')
      return
    }
    
    fetchInProgressRef.current = true
    const initialLoad = instances.length === 0
    
    if (initialLoad) {
      setLoading(true)
    }
    
    try {
      const data = await fetchInstancesApi()
      setInstances(data.instances || [])
      setError(null)
      setLastRefresh(new Date())
      lastSuccessfulFetchRef.current = new Date()
      errorCountRef.current = 0
      
      // Select first instance if none selected
      if (data.instances && data.instances.length > 0 && !selectedInstance) {
        setSelectedInstance(data.instances[0].title)
      }
    } catch (err) {
      console.error('Error fetching instances:', err)
      setError(err instanceof Error ? err.message : 'Failed to fetch instances')
      errorCountRef.current++
    } finally {
      if (initialLoad) {
        setLoading(false)
      }
      fetchInProgressRef.current = false
    }
  }, [instances.length, selectedInstance])
  
  // Set up polling interval (CORRECTLY - only once on mount)
  useEffect(() => {
    // Initial fetch
    fetchInstances()
    
    // Set up polling - 30 second interval
    fetchIntervalRef.current = window.setInterval(() => {
      fetchInstances()
    }, 30000)
    
    // Cleanup interval on unmount
    return () => {
      if (fetchIntervalRef.current !== null) {
        clearInterval(fetchIntervalRef.current)
        fetchIntervalRef.current = null
      }
    }
  }, []) // Empty dependency array - only run on mount and unmount
  
  // Get status color
  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'running':
        return '#28a745'  // Green
      case 'paused':
        return '#ffc107'  // Yellow
      case 'stopped':
        return '#dc3545'  // Red
      case 'error':
        return '#dc3545'  // Red
      default:
        return '#6c757d'  // Gray
    }
  }
  
  // Format timestamp
  const formatTime = (timestamp?: string) => {
    if (!timestamp) return 'Unknown'
    const date = new Date(timestamp)
    return date.toLocaleString()
  }
  
  // Refreshes the instance list
  const handleRefresh = () => {
    fetchInstances()
  }
  
  // Handle instance selection
  const handleInstanceSelect = (instanceName: string) => {
    setSelectedInstance(instanceName)
  }
  
  if (loading) {
    return <div className="container">Loading instances...</div>
  }
  
  return (
    <div className="integrated-page" style={{ 
      display: 'flex',
      flexDirection: 'column',
      height: '100vh',
      padding: '1rem'
    }}>
      <div style={{ 
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: '1rem'
      }}>
        <h1>Claude Squad</h1>
        <div>
          <button
            onClick={handleRefresh}
            disabled={fetchInProgressRef.current}
            style={{
              padding: '8px 16px',
              backgroundColor: '#4a90e2',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: fetchInProgressRef.current ? 'not-allowed' : 'pointer',
              opacity: fetchInProgressRef.current ? 0.7 : 1
            }}
          >
            {fetchInProgressRef.current ? 'Refreshing...' : 'Refresh'}
          </button>
        </div>
      </div>
      
      {error && (
        <div style={{
          padding: '1rem',
          backgroundColor: '#f8d7da',
          borderRadius: '4px',
          color: '#721c24',
          marginBottom: '1rem'
        }}>
          Error: {error}
        </div>
      )}
      
      <div style={{ fontSize: '0.9rem', color: '#666', marginBottom: '1rem' }}>
        Last updated: {lastRefresh.toLocaleString()}
        {isConnected && selectedInstance && (
          <span style={{ marginLeft: '1rem', color: '#28a745' }}>
            ● Connected to {selectedInstance}
          </span>
        )}
        {!isConnected && selectedInstance && (
          <span style={{ marginLeft: '1rem', color: '#dc3545' }}>
            ● Disconnected from {selectedInstance}
          </span>
        )}
      </div>
      
      <div style={{ 
        display: 'flex', 
        flexGrow: 1,
        height: 'calc(100vh - 120px)',
        gap: '1rem'
      }}>
        {/* Instances list - left column */}
        <div style={{ 
          width: '300px', 
          overflowY: 'auto',
          padding: '0.5rem',
          backgroundColor: '#f5f5f5',
          borderRadius: '4px'
        }}>
          <h2>Instances</h2>
          
          {instances.length === 0 ? (
            <div style={{ 
              padding: '1rem', 
              textAlign: 'center', 
              backgroundColor: '#fff', 
              borderRadius: '4px',
              marginTop: '1rem'
            }}>
              <p>No instances found</p>
            </div>
          ) : (
            <div>
              {instances.map(instance => (
                <div
                  key={instance.title}
                  onClick={() => handleInstanceSelect(instance.title)}
                  style={{
                    padding: '0.75rem',
                    borderRadius: '4px',
                    marginBottom: '0.5rem',
                    backgroundColor: selectedInstance === instance.title ? '#e0e0ff' : '#fff',
                    cursor: 'pointer',
                    border: '1px solid #ddd',
                    transition: 'background-color 0.2s'
                  }}
                >
                  <div style={{ 
                    display: 'flex', 
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    marginBottom: '0.5rem'
                  }}>
                    <div style={{ fontWeight: 'bold' }}>{instance.title}</div>
                    <div style={{
                      display: 'inline-block',
                      padding: '0.2rem 0.5rem',
                      borderRadius: '20px',
                      backgroundColor: getStatusColor(instance.status),
                      color: 'white',
                      fontSize: '0.75rem',
                    }}>
                      {instance.status}
                    </div>
                  </div>
                  
                  <div style={{ fontSize: '0.8rem', color: '#666' }}>
                    {instance.path && (
                      <div style={{ 
                        whiteSpace: 'nowrap',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis'
                      }}>
                        <strong>Path:</strong> {instance.path}
                      </div>
                    )}
                    {instance.updatedAt && (
                      <div>
                        <strong>Updated:</strong> {formatTime(instance.updatedAt)}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
        
        {/* Terminal - right column */}
        <div style={{ 
          flexGrow: 1,
          display: 'flex',
          flexDirection: 'column',
          height: '100%'
        }}>
          <div style={{ 
            padding: '0.5rem 1rem',
            backgroundColor: '#333',
            color: '#fff',
            borderTopLeftRadius: '4px',
            borderTopRightRadius: '4px',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center'
          }}>
            <div>Terminal: {selectedInstance || 'No instance selected'}</div>
            <div style={{ 
              display: 'flex',
              alignItems: 'center',
              fontSize: '0.8rem'
            }}>
              <div style={{
                width: '10px',
                height: '10px',
                borderRadius: '50%',
                backgroundColor: isConnected ? '#28a745' : '#dc3545',
                marginRight: '0.5rem'
              }}></div>
              {isConnected ? 'Connected' : 'Disconnected'}
            </div>
          </div>
          
          <div style={{ 
            flexGrow: 1,
            backgroundColor: '#1e1e1e',
            borderBottomLeftRadius: '4px',
            borderBottomRightRadius: '4px',
            overflow: 'hidden'
          }}>
            {selectedInstance ? (
              <Terminal
                instanceName={selectedInstance}
                onConnectionChange={setIsConnected}
              />
            ) : (
              <div style={{ 
                height: '100%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: '#666'
              }}>
                <p>Select an instance to connect</p>
              </div>
            )}
          </div>
        </div>
      </div>
      
      <div style={{ 
        marginTop: '1rem',
        textAlign: 'center',
        color: '#666',
        fontSize: '0.875rem'
      }}>
        Claude Squad &copy; {new Date().getFullYear()}
      </div>
    </div>
  )
}

export default IntegratedPage