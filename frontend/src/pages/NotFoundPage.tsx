import { Link } from 'react-router-dom'

const NotFoundPage = () => {
  return (
    <div className="container" style={{ textAlign: 'center', marginTop: '4rem' }}>
      <h1>404 - Page Not Found</h1>
      <p>The page you are looking for does not exist.</p>
      
      <div style={{ marginTop: '2rem' }}>
        <Link to="/">
          <button>Go to Home</button>
        </Link>
      </div>
    </div>
  )
}

export default NotFoundPage