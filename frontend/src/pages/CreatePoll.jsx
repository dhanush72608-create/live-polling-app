import React, { useState } from 'react';
import axios from 'axios';
import { useNavigate } from 'react-router-dom';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export default function CreatePoll() {
  const [question, setQuestion] = useState('');
  const [options, setOptions] = useState(['', '']);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  const handleOptionChange = (index, value) => {
    const updated = [...options];
    updated[index] = value;
    setOptions(updated);
  };

  const addOption = () => {
    if (options.length < 6) setOptions([...options, '']);
  };

  const handleCreate = async (e) => {
    e.preventDefault();
    const token = localStorage.getItem('token');
    if (!token) {
      navigate('/login');
      return;
    }

    const filteredOptions = options.map((o) => o.trim()).filter(Boolean);
    if (filteredOptions.length < 2) {
      setError('Please provide at least 2 non-empty options.');
      return;
    }

    try {
      const res = await axios.post(
        `${API_BASE}/api/polls`,
        { question, options: filteredOptions },
        { headers: { Authorization: `Bearer ${token}` } }
      );
      navigate(`/poll/${res.data.poll_id}`);
    } catch (err) {
      if (err.response?.status === 401) {
        localStorage.removeItem('token');
        navigate('/login');
      } else {
        setError(err.response?.data?.error || 'Failed to create poll');
      }
    }
  };

  return (
    <div className="card">
      <h2>Create a Live Poll</h2>
      {error && <div className="alert-error">{error}</div>}
      <form onSubmit={handleCreate}>
        <label style={{ fontWeight: 600, display: 'block', marginBottom: '0.5rem' }}>Question</label>
        <input
          type="text"
          placeholder="e.g. Which programming language do you use the most?"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          required
        />

        <label style={{ fontWeight: 600, display: 'block', marginBottom: '0.5rem' }}>Options</label>
        {options.map((opt, i) => (
          <input
            key={i}
            type="text"
            placeholder={`Option ${i + 1}`}
            value={opt}
            onChange={(e) => handleOptionChange(i, e.target.value)}
            required
          />
        ))}

        <div style={{ display: 'flex', gap: '0.75rem', marginTop: '0.5rem' }}>
          {options.length < 6 && (
            <button type="button" onClick={addOption} style={{ background: '#64748b' }}>
              + Add Option
            </button>
          )}
          <button type="submit">Create & Get Link</button>
        </div>
      </form>
    </div>
  );
}