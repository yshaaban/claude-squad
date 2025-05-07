import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
// Terminal component will be implemented later
import Terminal from '@/components/terminal/Terminal'

const TerminalPage = () => {
  const { instanceName } = useParams<{ instanceName: string }>()
  const [isConnected, setIsConnected] = useState(false)
  
  // Placeholder for instance data
  const [instanceData, setInstanceData] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  
  useEffect(() => {
    // This will be replaced with actual API call later
    const fetchInstanceData = async () => {
      try {
        // Placeholder - will be implemented with actual API
        setInstanceData({ title: instanceName })
        setLoading(false)
      } catch (err) {
        setError('Failed to fetch instance data')
        setLoading(false)
      }
    }
    
    fetchInstanceData()
  }, [instanceName])
  
  if (loading) {
    return <div className="container">Loading instance data...</div>
  }
  
  if (error) {
    return (
      <div className="container">
        <div className="error">{error}</div>
        <Link to="/instances">Back to Instances</Link>
      </div>
    )
  }
  
  return (
    <div className="container">
      <div className="terminal-header">
        <h1>{instanceData?.title || 'Terminal'}</h1>
        <div className="status">
          <span className={`status-indicator ${isConnected ? 'connected' : 'disconnected'}`}></span>
          <span>{isConnected ? 'Connected' : 'Disconnected'}</span>
        </div>
      </div>
      
      <div className="terminal-container">
        <Terminal 
          instanceName={instanceName || ''} 
          onConnectionChange={setIsConnected}
        />
      </div>
      
      <div className="terminal-controls" style={{ marginTop: '1rem' }}>
        <Link to="/instances">
          <button style={{ marginRight: '1rem' }}>Back to Instances</button>
        </Link>
      </div>
    </div>
  )
}

export default TerminalPage