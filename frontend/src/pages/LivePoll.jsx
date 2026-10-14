import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import axios from 'axios';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';
const WS_BASE = API_BASE.replace(/^http/, 'ws');

export default function LivePoll() {
  const { id } = useParams();
  const [poll, setPoll] = useState(null);
  const [counts, setCounts] = useState({});
  const [hasVoted, setHasVoted] = useState(false);
  const [copied, setCopied] = useState(false);

  // 1. Initial poll fetch
  useEffect(() => {
    axios
      .get(`${API_BASE}/api/polls/${id}`)
      .then((res) => {
        setPoll(res.data.poll);
        setCounts(res.data.counts || {});
      })
      .catch(() => alert('Poll not found.'));
  }, [id]);

  // 2. Real-time WebSocket connection to Redis pub/sub
  useEffect(() => {
    const ws = new WebSocket(`${WS_BASE}/ws/polls/${id}`);

    ws.onmessage = (event) => {
      try {
        const liveCounts = JSON.parse(event.data);
        setCounts(liveCounts);
      } catch (e) {
        console.error('Failed to parse WebSocket message', e);
      }
    };

    return () => ws.close();
  }, [id]);

  const handleVote = async (optionId) => {
    if (hasVoted) return;
    try {
      await axios.post(`${API_BASE}/api/polls/${id}/vote`, { option_id: optionId });
      setHasVoted(true);
    } catch {
      alert('Could not record vote. Please try again.');
    }
  };

  const copyShareLink = () => {
    navigator.clipboard.writeText(window.location.href);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (!poll) return <div className="card"><p>Loading live poll...</p></div>;

  const totalVotes = Object.values(counts).reduce((acc, curr) => acc + Number(curr || 0), 0);

  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <h2>{poll.question}</h2>
          <p style={{ color: '#64748b', fontSize: '0.9rem', marginBottom: '1.5rem' }}>
            {totalVotes} total {totalVotes === 1 ? 'vote' : 'votes'} • Updates live
          </p>
        </div>
        <button
          onClick={copyShareLink}
          style={{ padding: '0.4rem 0.8rem', fontSize: '0.85rem', background: '#475569' }}
        >
          {copied ? 'Copied!' : 'Share Link'}
        </button>
      </div>

      <div>
        {poll.options.map((opt) => {
          const voteCount = Number(counts[opt.id] || 0);
          const percent = totalVotes > 0 ? Math.round((voteCount / totalVotes) * 100) : 0;

          return (
            <button
              key={opt.id}
              disabled={hasVoted}
              onClick={() => handleVote(opt.id)}
              className="poll-option-btn"
            >
              <div className="bar-fill" style={{ width: `${percent}%` }} />
              <div className="option-content">
                <span style={{ fontWeight: 600 }}>{opt.text}</span>
                <span style={{ color: '#475569', fontSize: '0.9rem' }}>
                  {voteCount} ({percent}%)
                </span>
              </div>
            </button>
          );
        })}
      </div>

      {hasVoted && (
        <div className="alert-success">
          ✓ Vote recorded! Watch the bars move in real-time as others vote.
        </div>
      )}
    </div>
  );
}