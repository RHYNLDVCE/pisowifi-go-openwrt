import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import '@fontsource/outfit/300.css';
import '@fontsource/outfit/400.css';
import '@fontsource/outfit/500.css';
import '@fontsource/outfit/600.css';
import '@fontsource/outfit/700.css';
import '@fontsource/outfit/800.css';
import toast from 'react-hot-toast';
import App from './App.jsx'

const originalFetch = window.fetch;
window.fetch = async function (...args) {
  try {
    const response = await originalFetch.apply(this, args);
    // If the server redirected to the login page (token expired or unauthorized)
    if (response.url && response.url.endsWith('/login')) {
      toast.error('Session expired. Please log in again.');
      setTimeout(() => {
        window.location.href = '/login';
      }, 1500);
      // Return a pending promise so the calling component stays in loading state
      return new Promise(() => {});
    }
    return response;
  } catch (error) {
    throw error;
  }
};

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
