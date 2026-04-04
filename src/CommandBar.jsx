import React, { useState, useRef, useEffect } from 'react';

export default function CommandBar({ visible, onSubmit }) {
  const [text, setText] = useState('');
  const inputRef = useRef(null);

  useEffect(() => {
    if (visible && inputRef.current) {
      inputRef.current.focus();
    }
  }, [visible]);

  function handleKeyDown(e) {
    if (e.key === 'Enter' && text.trim()) {
      onSubmit(text.trim());
      setText('');
    }
    if (e.key === 'Escape') {
      setText('');
      onSubmit(null); // signal close without query
    }
  }

  return (
    <div
      style={{
        position: 'absolute',
        bottom: 24,
        left: 12,
        right: 12,
        opacity: visible ? 1 : 0,
        transform: visible ? 'translateY(0)' : 'translateY(6px)',
        transition: 'opacity 0.2s ease, transform 0.2s ease',
        pointerEvents: visible ? 'auto' : 'none',
      }}
    >
      <input
        ref={inputRef}
        type="text"
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="Ask Ronin..."
        style={{
          width: '100%',
          boxSizing: 'border-box',
          background: 'rgba(0, 0, 0, 0.75)',
          color: '#e0e0e0',
          fontFamily: "'Press Start 2P', monospace",
          fontSize: 7,
          padding: '8px 10px',
          border: 'none',
          borderBottom: '2px solid rgba(255, 255, 255, 0.15)',
          outline: 'none',
          caretColor: '#90beff',
          letterSpacing: '0.05em',
          WebkitAppRegion: 'no-drag',
        }}
      />
    </div>
  );
}
