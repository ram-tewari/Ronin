import React, { useState, useEffect, useCallback } from 'react';
import ChibiEngine from './ChibiEngine';
import CommandBar from './CommandBar';

const BUBBLE_STYLE = {
  idle:      { border: '#1a5fa8', bg: '#080f20', text: '#90beff', glow: '#1a5fa844' },
  alert:     { border: '#cc1800', bg: '#1a0500', text: '#ff9070', glow: '#cc180044' },
  hyped:     { border: '#ffaa00', bg: '#221100', text: '#ffdd88', glow: '#ffaa0044' },
  exhausted: { border: '#3a5a3a', bg: '#06100a', text: '#80b880', glow: '#3a5a3a33' },
};

function DialogueBubble({ mood, message, visible, onClick, config }) {
  const defaultStyle = BUBBLE_STYLE[mood] || BUBBLE_STYLE.idle;

  // Apply Config Overrides
  const overrideBorder = config?.aesthetics?.themeColor || defaultStyle.border;
  const isDarkMode = config?.aesthetics?.bubbleDarkMode;

  const bg = isDarkMode ? '#000000' : defaultStyle.bg;
  const border = overrideBorder;
  const glow = overrideBorder + '44';
  const text = defaultStyle.text;

  return (
    <div
      onClick={onClick}
      style={{
        position: 'absolute',
        top: 8,
        left: 8,
        right: 8,
        minHeight: 60,
        background: bg,
        border: `4px solid ${border}`,
        outline: `2px solid ${glow}`,
        outlineOffset: '2px',
        color: text,
        fontFamily: "'Press Start 2P', monospace",
        fontSize: 7,
        lineHeight: 1.8,
        padding: '8px 10px 12px',
        opacity: visible ? 1 : 0,
        transform: visible ? 'translateY(0)' : 'translateY(-4px)',
        transition: 'opacity 0.15s, transform 0.15s',
        WebkitAppRegion: 'no-drag',
        boxShadow: `inset 2px 2px 0 ${border}33, 4px 4px 0 #00000066`,
        imageRendering: 'pixelated',
        backgroundImage: `linear-gradient(transparent 50%, rgba(0,0,0,0.08) 50%)`,
        backgroundSize: '100% 4px',
        backgroundBlendMode: 'multiply',
        cursor: onClick ? 'pointer' : 'default',
        pointerEvents: visible ? 'auto' : 'none',
      }}
    >
      {message}

      <div style={{
        position: 'absolute',
        bottom: -12,
        left: '50%',
        transform: 'translateX(-50%)',
        width: 0, height: 0,
        borderLeft:  '8px solid transparent',
        borderRight: '8px solid transparent',
        borderTop:   `12px solid ${border}`,
      }} />
      <div style={{
        position: 'absolute',
        bottom: -7,
        left: '50%',
        transform: 'translateX(-50%)',
        width: 0, height: 0,
        borderLeft:  '4px solid transparent',
        borderRight: '4px solid transparent',
        borderTop:   `8px solid ${bg}`,
      }} />
    </div>
  );
}

export default function App() {
  const [mood, setMood] = useState('idle');
  const [bubbleVisible, setBubbleVisible] = useState(false);
  const [message, setMessage] = useState('');
  const [alerts, setAlerts] = useState([]);
  const [config, setConfig] = useState(null);
  const [commandBarVisible, setCommandBarVisible] = useState(false);

  useEffect(() => {
    // Load config on mount
    if (window.ronin?.getSettings) {
      window.ronin.getSettings().then(c => {
         if (c) setConfig(c);
      });
    }

    // Listen for config changes from IPC
    let cleanupIpc = null;
    if (window.ronin?.onSettingsSaved) {
      cleanupIpc = window.ronin.onSettingsSaved((newConfig) => {
        setConfig(newConfig);
      });
    }

    // Polling logic
    const fetchStatus = async () => {
      try {
        const response = await fetch('http://localhost:8080/status');
        if (!response.ok) return;
        const data = await response.json();

        // Alerts are already filtered server-side by selected teams
        const enabledAlerts = data.alerts || [];
        const activeAlert = enabledAlerts.length > 0 ? enabledAlerts[0] : null;
        let finalMessage = activeAlert ? data.message : ''; // or activeAlert.event
        let finalMood = activeAlert ? data.mood : 'idle';

        // Override message simply to show nothing if data is filtered out
        if (data.message && data.message !== message && activeAlert) {
          setMessage(finalMessage);
          setAlerts(enabledAlerts);
          setMood(finalMood);
          setBubbleVisible(true);

          // Hide bubble after 8 seconds
          setTimeout(() => {
            setBubbleVisible(false);
          }, 8000);
        } else if (!activeAlert) {
          setMessage('');
          setMood('idle');
          setBubbleVisible(false);
        }
      } catch (err) {
        console.error('Failed to fetch from brain:', err);
      }
    };

    fetchStatus(); // Initial fetch
    const interval = setInterval(fetchStatus, 15000); // Poll every 15 seconds

    return () => {
      clearInterval(interval);
      if (cleanupIpc) cleanupIpc();
    };
  }, [message]);

  function handleBubbleClick() {
    if (alerts && alerts.length > 0 && alerts[0].link) {
      // Safely call the secure IPC bridge
      if (window.ronin && window.ronin.openExternal) {
        window.ronin.openExternal(alerts[0].link);
      }
    }
  }

  function handleContextMenu(e) {
    if (window.ronin && window.ronin.showContextMenu) {
      e.preventDefault();
      window.ronin.showContextMenu();
    }
  }

  function handleChibiClick() {
    setCommandBarVisible(prev => !prev);
  }

  const handleCommandSubmit = useCallback(async (query) => {
    // null means Escape was pressed — just close
    if (query === null) {
      setCommandBarVisible(false);
      return;
    }

    setCommandBarVisible(false);

    try {
      const response = await window.ronin.sendQuery(query);
      if (response && response.mood && response.message) {
        setMood(response.mood);
        setMessage(response.message);
        setBubbleVisible(true);

        setTimeout(() => {
          setBubbleVisible(false);
        }, 8000);
      }
    } catch (err) {
      console.error('Query failed:', err);
      setMood('exhausted');
      setMessage('Backend is unreachable. Try again.');
      setBubbleVisible(true);
      setTimeout(() => setBubbleVisible(false), 5000);
    }
  }, []);

  return (
    <div
      className="overflow-hidden relative"
      style={{ width: 280, height: 360, background: 'transparent' }}
      onContextMenu={handleContextMenu}
    >
      <DialogueBubble
        mood={mood}
        message={message}
        visible={bubbleVisible}
        onClick={handleBubbleClick}
        config={config}
      />

      {/* Character in lower portion of window */}
      <div
        style={{
          position: 'absolute',
          top: 100,
          left: 0, right: 0, bottom: 28,
        }}
      >
        <ChibiEngine mood={mood} onContextMenu={handleContextMenu} onClick={handleChibiClick} />
      </div>

      <CommandBar
        visible={commandBarVisible}
        onSubmit={handleCommandSubmit}
      />

      <div
        style={{
          position: 'absolute',
          bottom: 6,
          left: '42%',
          transform: 'translateX(-50%)',
          fontFamily: "'Press Start 2P', monospace",
          fontSize: 6,
          color: '#ffffff',
          background: 'rgba(0,0,0,0.45)',
          border: '2px solid rgba(255,255,255,0.25)',
          padding: '4px 8px',
          WebkitAppRegion: 'no-drag',
          imageRendering: 'pixelated',
          letterSpacing: '0.05em',
          pointerEvents: 'none'
        }}
      >
        {mood}
      </div>
    </div>
  );
}
