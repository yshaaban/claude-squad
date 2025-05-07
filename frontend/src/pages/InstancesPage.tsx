import { useEffect, useState, useRef } from 'react'
import { Link } from 'react-router-dom'
import { Instance, InstancesResponse } from '@/types/instance'

const API_BASE_URL = '/api'

// API functions with proper error handling - no recursive retry
const fetchInstancesApi = async (): Promise<InstancesResponse> => {
  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), 10000) // 10 second timeout
  
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

const InstancesPage = () => {
  const [instances, setInstances] = useState<Instance[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date())
  
  // Use refs for tracking state that shouldn't trigger re-renders
  const errorCountRef = useRef(0)
  const fetchInProgressRef = useRef(false)
  const lastFetchTimeRef = useRef<Date>(new Date())
  
  const fetchInstances = async () => {
    // Prevent concurrent fetches
    if (fetchInProgressRef.current) {
      console.log('Fetch already in progress, skipping')
      return
    }
    
    // Skip fetching if we've had too many consecutive errors
    if (errorCountRef.current > 5) {
      setError('Too many consecutive errors. Please reload the page.')
      return
    }
    
    // Check if we've fetched recently (within 5 seconds)
    const timeSinceLastFetch = Date.now() - lastFetchTimeRef.current.getTime()
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
      lastFetchTimeRef.current = new Date()
      errorCountRef.current = 0 // Reset error count on success
    } catch (err) {
      console.error('Error fetching instances:', err)
      setError(err instanceof Error ? err.message : 'Failed to fetch instances')
      errorCountRef.current++ // Increment error count
    } finally {
      if (initialLoad) {
        setLoading(false)
      }
      fetchInProgressRef.current = false
    }
  }
  
  // Initial fetch - with proper cleanup and refs
  // This is the CORRECT way to set up an interval in React
  useEffect(() => {
    // Initial fetch on mount
    fetchInstances()
    
    // Set up polling with fixed interval - only runs once
    const intervalId = setInterval(() => {
      fetchInstances()
    }, 30000) // 30 second interval
    
    // Proper cleanup
    return () => {
      clearInterval(intervalId)
    }
  }, []) // Empty dependency array = run ONCE on mount
  
  if (loading) {
    return <div className="container">Loading instances...</div>
  }
  
  if (error) {
    return <div className="container">Error: {error}</div>
  }
  
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
  
  return (
    <div className="container" style={{ maxWidth: '1200px', margin: '0 auto', padding: '20px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
        <h1>Claude Squad Instances</h1>
        <div>
          <button 
            onClick={handleRefresh} 
            disabled={loading}
            style={{ 
              padding: '8px 16px',
              backgroundColor: '#4a90e2',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: loading ? 'not-allowed' : 'pointer',
              opacity: loading ? 0.7 : 1
            }}
          >
            {loading ? 'Refreshing...' : 'Refresh'}
          </button>
        </div>
      </div>
      
      <div style={{ fontSize: '0.9rem', color: '#666', marginBottom: '2rem' }}>
        Last updated: {lastRefresh.toLocaleString()} {/* Show when data was last refreshed */}
      </div>
      
      {error && (
        <div style={{ 
          padding: '1rem', 
          backgroundColor: '#f8d7da', 
          borderRadius: '4px', 
          color: '#721c24',
          marginBottom: '2rem' 
        }}>
          Error: {error}
        </div>
      )}
      
      <div className="instances-list" style={{ marginTop: '1rem' }}>
        {instances.length === 0 && !loading ? (
          <div style={{ 
            padding: '2rem', 
            textAlign: 'center', 
            backgroundColor: '#f8f9fa', 
            borderRadius: '4px'
          }}>
            <h3>No instances found</h3>
            <p>Create a new Claude instance to get started.</p>
          </div>
        ) : (
          <div style={{ 
            display: 'grid', 
            gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', 
            gap: '1rem' 
          }}>
            {instances.map(instance => (
              <div key={instance.title} style={{ 
                padding: '1.5rem', 
                border: '1px solid #ddd',
                borderRadius: '8px',
                backgroundColor: 'white',
                boxShadow: '0 2px 4px rgba(0,0,0,0.05)',
                transition: 'transform 0.2s, box-shadow 0.2s',
                ':hover': {
                  transform: 'translateY(-2px)',
                  boxShadow: '0 4px 8px rgba(0,0,0,0.1)'
                }
              }}>
                <div style={{ 
                  display: 'flex', 
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  marginBottom: '1rem'
                }}>
                  <h2 style={{ fontSize: '1.25rem', margin: 0 }}>{instance.title}</h2>
                  <div style={{
                    display: 'inline-block',
                    padding: '0.25rem 0.75rem',
                    borderRadius: '20px',
                    backgroundColor: getStatusColor(instance.status),
                    color: 'white',
                    fontSize: '0.875rem',
                    fontWeight: 'bold'
                  }}>
                    {instance.status}
                  </div>
                </div>
                
                <div style={{ fontSize: '0.875rem', color: '#666', marginBottom: '1rem' }}>
                  {instance.path && (
                    <div style={{ marginBottom: '0.5rem' }}>
                      <strong>Path:</strong> {instance.path}
                    </div>
                  )}
                  {instance.createdAt && (
                    <div style={{ marginBottom: '0.5rem' }}>
                      <strong>Created:</strong> {formatTime(instance.createdAt)}
                    </div>
                  )}
                  {instance.updatedAt && (
                    <div>
                      <strong>Last Updated:</strong> {formatTime(instance.updatedAt)}
                    </div>
                  )}
                </div>
                
                <div style={{ marginTop: '1.5rem' }}>
                  <Link to={`/terminal/${instance.title}`} style={{ textDecoration: 'none' }}>
                    <button style={{ 
                      width: '100%',
                      padding: '0.75rem',
                      backgroundColor: '#4a90e2',
                      color: 'white',
                      border: 'none',
                      borderRadius: '4px',
                      cursor: 'pointer',
                      fontWeight: 'bold',
                      transition: 'background-color 0.2s',
                      ':hover': {
                        backgroundColor: '#3a80d2'
                      }
                    }}>
                      Open Terminal
                    </button>
                  </Link>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
      
      <div style={{ marginTop: '3rem', textAlign: 'center', color: '#666', fontSize: '0.875rem' }}>
        <div>Claude Squad &copy; {new Date().getFullYear()}</div>
        <div style={{ marginTop: '0.5rem' }}>
          <Link to="/" style={{ color: '#4a90e2', textDecoration: 'none' }}>Home</Link>
        </div>
      </div>
    </div>
  )
}

export default InstancesPage