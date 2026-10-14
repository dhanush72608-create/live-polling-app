import React from 'react';
import { BrowserRouter, Routes, Route, Link, useNavigate } from 'react-router-dom';
import Login from './pages/Login';
import Register from './pages/Register';
import CreatePoll from './pages/CreatePoll';
import LivePoll from './pages/LivePoll';

function Navigation() {
  const navigate = useNavigate();
  const token = localStorage.getItem('token');

  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('email');
    navigate('/login');
  };

  return (
    <nav className="navbar">
      <Link to="/" style={{ fontSize: '1.25rem', color: '#0f172a', fontWeight: 'bold' }}>
        Live Polling
      </Link>
      <div>
        {token ? (
          <>
            <Link to="/create">Create Poll</Link>
            <button
              onClick={handleLogout}
              style={{ marginLeft: '1rem', padding: '0.4rem 0.8rem', fontSize: '0.85rem' }}
            >
              Logout
            </button>
          </>
        ) : (
          <>
            <Link to="/login">Login</Link>
            <Link to="/register">Register</Link>
          </>
        )}
      </div>
    </nav>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <Navigation />
      <div className="container">
        <Routes>
          <Route path="/" element={<CreatePoll />} />
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/create" element={<CreatePoll />} />
          <Route path="/poll/:id" element={<LivePoll />} />
        </Routes>
      </div>
    </BrowserRouter>
  );
}