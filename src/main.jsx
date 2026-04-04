import React from 'react';
import ReactDOM from 'react-dom/client';
import { HashRouter, Routes, Route, Navigate } from 'react-router-dom';
import App from './App';
import Settings from './Settings';
import './index.css';

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <HashRouter>
      <Routes>
        <Route path="/chibi" element={<App />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="*" element={<Navigate to="/chibi" replace />} />
      </Routes>
    </HashRouter>
  </React.StrictMode>
);
