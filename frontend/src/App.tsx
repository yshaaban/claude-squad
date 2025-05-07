import { Routes, Route } from 'react-router-dom'
import HomePage from './pages/HomePage'
import TerminalPage from './pages/TerminalPage'
import InstancesPage from './pages/InstancesPage'
import IntegratedPage from './pages/IntegratedPage'
import NotFoundPage from './pages/NotFoundPage'

function App() {
  return (
    <div className="app">
      <Routes>
        {/* Use the integrated page as the default */}
        <Route path="/" element={<IntegratedPage />} />
        
        {/* Keep original routes for backward compatibility */}
        <Route path="/home" element={<HomePage />} />
        <Route path="/terminal/:instanceName" element={<TerminalPage />} />
        <Route path="/instances" element={<InstancesPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </div>
  )
}

export default App