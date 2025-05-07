import { Link } from 'react-router-dom'

const HomePage = () => {
  return (
    <div className="container">
      <h1>Claude Squad</h1>
      <p>Welcome to the Claude Squad terminal interface.</p>
      
      <div className="actions" style={{ marginTop: '2rem' }}>
        <Link to="/instances">
          <button>View All Instances</button>
        </Link>
      </div>
    </div>
  )
}

export default HomePage