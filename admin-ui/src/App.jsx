import React, { useEffect, useState } from 'react';
import { BrowserRouter, Routes, Route, Link, useLocation } from 'react-router-dom';
import { Toaster } from 'react-hot-toast';
import { Home, Users, Image, Activity, ShieldAlert, Sun, Moon, Menu, Wifi, MonitorSmartphone, Coins, Award, Server, LogOut, X, Clock, Settings, Ticket } from 'lucide-react';
import Dashboard from './pages/Dashboard';
import Connections from './pages/Connections';
import ManageUser from './pages/ManageUser';
import Devices from './pages/Devices';
import Logs from './pages/Logs';
import NetworkSettings from './pages/NetworkSettings';
import PortalSettings from './pages/PortalSettings';
import SessionSettings from './pages/SessionSettings';
import CoinSettings from './pages/CoinSettings';
import LoyaltySettings from './pages/LoyaltySettings';
import SystemStats from './pages/SystemStats';
import SystemSettings from './pages/SystemSettings';
import Vouchers from './pages/Vouchers';

function Layout({ children }) {
  const location = useLocation();
  const [darkMode, setDarkMode] = useState(() => {
    return localStorage.getItem('theme') === 'dark';
  });
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [showLogoutConfirm, setShowLogoutConfirm] = useState(false);
  const [currentTime, setCurrentTime] = useState(new Date());

  useEffect(() => {
    if (darkMode) {
      document.documentElement.classList.add('dark');
      localStorage.setItem('theme', 'dark');
    } else {
      document.documentElement.classList.remove('dark');
      localStorage.setItem('theme', 'light');
    }
  }, [darkMode]);

  useEffect(() => {
    const timer = setInterval(() => setCurrentTime(new Date()), 1000);
    return () => clearInterval(timer);
  }, []);

  const formatDate = (date) => {
    return date.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric', year: 'numeric' });
  };
  const formatTime = (date) => {
    return date.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit', second: '2-digit', hour12: true });
  };

  const navItems = [
    { path: '/admin', icon: <Home size={20} />, label: 'Analytics' },
    { path: '/admin/connections', icon: <Users size={20} />, label: 'Active Connections' },
    { path: '/admin/system', icon: <Server size={20} />, label: 'System Stats' },
    { path: '/admin/network', icon: <Wifi size={20} />, label: 'Network Bandwidth' },
    { path: '/admin/session', icon: <Clock size={20} />, label: 'Session & Access' },
    { path: '/admin/portal', icon: <MonitorSmartphone size={20} />, label: 'Portal UI & Sounds' },
    { path: '/admin/coins', icon: <Coins size={20} />, label: 'Coin Configuration' },
    { path: '/admin/loyalty', icon: <Award size={20} />, label: 'Loyalty & Rewards' },
    { path: '/admin/vouchers', icon: <Ticket size={20} />, label: 'Vouchers' },
    { path: '/admin/devices', icon: <Activity size={20} />, label: 'Infrastructure' },
    { path: '/admin/maintenance', icon: <Settings size={20} />, label: 'System Maintenance' },
    { path: '/admin/logs', icon: <ShieldAlert size={20} />, label: 'System Logs' },
  ];

  const currentNavItem = navItems.find(item => item.path === location.pathname);
  const pageTitle = currentNavItem ? currentNavItem.label : (location.pathname.startsWith('/admin/user') ? 'User Details' : 'Control Center');

  return (
    <div className="flex h-screen overflow-hidden bg-gray-50 text-gray-900 dark:bg-black dark:text-zinc-300 transition-colors duration-200">
      {/* Mobile Full-Screen Menu Overlay */}
      {sidebarOpen && (
        <div className="fixed inset-0 z-50 bg-gray-50 dark:bg-black md:hidden flex flex-col">
          <div className="h-20 flex items-center justify-between px-6 border-b border-gray-200 dark:border-zinc-800 shrink-0">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 flex items-center justify-center">
                <Menu className="w-6 h-6" />
              </div>
              <div className="font-bold text-lg leading-tight">Menu</div>
            </div>
            <button 
              onClick={() => setSidebarOpen(false)}
              className="p-2 bg-gray-200 dark:bg-zinc-800 rounded-full"
            >
              <X size={20} />
            </button>
          </div>
          <nav className="flex-1 px-4 py-6 space-y-2 overflow-y-auto">
            {navItems.map(item => {
              const isActive = location.pathname === item.path;
              return (
                <Link 
                  key={item.path} 
                  to={item.path} 
                  onClick={() => setSidebarOpen(false)}
                  className={`flex items-center gap-4 px-5 py-4 rounded-xl text-base font-semibold transition-all ${
                    isActive 
                      ? 'bg-gray-200 text-black dark:bg-zinc-800 dark:text-white' 
                      : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-zinc-900'
                  }`}
                >
                  {item.icon}
                  <span>{item.label}</span>
                </Link>
              );
            })}
            <button onClick={() => { setSidebarOpen(false); setShowLogoutConfirm(true); }} className="w-full flex items-center gap-4 px-5 py-4 mt-2 rounded-xl text-base font-bold text-red-600 hover:bg-red-50 dark:hover:bg-red-500/10 transition-colors">
              <LogOut size={20} />
              Logout
            </button>
          </nav>
        </div>
      )}

      {/* Desktop Sidebar (hidden on mobile) */}
      <aside className="hidden md:flex flex-col inset-y-0 left-0 z-30 w-64 bg-white dark:bg-zinc-950 border-r border-gray-200 dark:border-zinc-800 transition-transform duration-300 shrink-0">
        <div className="h-16 flex items-center px-6 border-b border-gray-200 dark:border-zinc-800 shrink-0">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 bg-black dark:bg-white rounded flex items-center justify-center">
              <Wifi className="text-white dark:text-black w-5 h-5" />
            </div>
            <div>
              <div className="font-bold text-lg leading-tight">PisoWifi</div>
              <div className="text-[10px] font-bold uppercase tracking-wider text-gray-500 dark:text-gray-400">Admin Panel</div>
            </div>
          </div>
        </div>

        <nav className="flex-1 px-4 py-6 space-y-1 overflow-y-auto">
          {navItems.map(item => {
            const isActive = location.pathname === item.path;
            return (
              <Link 
                key={item.path} 
                to={item.path} 
                onClick={() => setSidebarOpen(false)}
                className={`flex items-center gap-3.5 px-4 py-3 rounded-xl text-sm font-semibold transition-all ${
                  isActive 
                    ? 'bg-gray-200 text-black dark:bg-zinc-800 dark:text-white shadow-sm' 
                    : 'text-gray-500 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-zinc-900 dark:hover:text-gray-200'
                }`}
              >
                {item.icon}
                <span>{item.label}</span>
              </Link>
            );
          })}
        </nav>
        
        <div className="p-4 border-t border-gray-200 dark:border-zinc-800 shrink-0">
          <button onClick={() => setShowLogoutConfirm(true)} className="flex items-center justify-center gap-2 w-full px-4 py-2.5 text-sm font-bold text-red-600 bg-red-50 dark:bg-red-500/10 dark:text-red-500 rounded-md hover:bg-red-100 dark:hover:bg-red-500/20 transition-colors">
            <LogOut size={16} />
            Logout
          </button>
        </div>
      </aside>

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col min-w-0 pb-16 md:pb-0">
        <header className="h-20 bg-gray-50 dark:bg-black flex items-center justify-between px-4 sm:px-8 z-10 shrink-0 shadow-[0_4px_6px_-1px_rgba(0,0,0,0.05)] dark:shadow-none">
          <div className="flex items-center gap-3 sm:gap-4 overflow-hidden">
            <h1 className="text-lg sm:text-xl font-bold truncate">{pageTitle}</h1>
          </div>

          <div className="flex items-center gap-3 sm:gap-6 shrink-0">
            <div className="flex flex-col items-end">
              <span className="text-[10px] sm:text-xs font-bold">{formatTime(currentTime)}</span>
              <span className="text-[7px] sm:text-[9px] font-bold text-gray-500 uppercase tracking-widest">{formatDate(currentTime)}</span>
            </div>
            
            <button
              onClick={() => setDarkMode(!darkMode)}
              className="p-2 rounded-full bg-gray-100 hover:bg-gray-200 dark:bg-zinc-900 dark:hover:bg-zinc-800 transition-colors border border-gray-200 dark:border-zinc-800"
              aria-label="Toggle Dark Mode"
            >
              {darkMode ? <Sun size={18} className="text-gray-400 hover:text-white" /> : <Moon size={18} className="text-gray-600 hover:text-black" />}
            </button>
          </div>
        </header>

        <main className="flex-1 px-4 sm:px-8 pt-6 pb-8 overflow-y-auto">
          <div className="max-w-7xl mx-auto">
            {children}
          </div>
        </main>
      </div>

      {showLogoutConfirm && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-[1000] flex items-center justify-center p-4 overflow-y-auto">
          <div className="bg-white dark:bg-zinc-900 w-full max-w-sm rounded-2xl shadow-xl overflow-hidden animate-in fade-in zoom-in-95 duration-200">
            <div className="p-6 text-center">
              <div className="w-16 h-16 bg-red-100 dark:bg-red-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
                <LogOut className="w-8 h-8 text-red-600 dark:text-red-500" />
              </div>
              <h3 className="text-xl font-bold mb-2">Logout</h3>
              <p className="text-gray-500 dark:text-gray-400 text-sm mb-6">
                Are you sure you want to log out of the admin panel?
              </p>
              <div className="flex gap-3">
                <button 
                  className="flex-1 px-4 py-3 bg-gray-100 text-gray-700 dark:bg-zinc-800 dark:text-gray-300 font-bold rounded-xl hover:bg-gray-200 dark:hover:bg-zinc-700 transition-colors"
                  onClick={() => setShowLogoutConfirm(false)}
                >
                  Cancel
                </button>
                <button 
                  className="flex-1 px-4 py-3 bg-red-600 text-white font-bold rounded-xl hover:bg-red-700 transition-colors"
                  onClick={() => window.location.href = '/logout'}
                >
                  Logout
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Mobile Bottom Navigation */}
      <div className="md:hidden fixed bottom-0 left-0 right-0 h-16 bg-white dark:bg-zinc-950 border-t border-gray-100 dark:border-zinc-900 flex justify-around items-center z-40 px-2 pb-safe shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.05)] dark:shadow-none">
        {navItems.slice(0, 3).map(item => {
           const isActive = location.pathname === item.path;
           return (
             <Link 
               key={item.path} 
               to={item.path} 
               onClick={() => setSidebarOpen(false)}
               className={`flex flex-col items-center justify-center w-full h-full space-y-1 transition-colors ${
                 isActive ? 'text-gray-900 dark:text-white' : 'text-gray-400 dark:text-gray-500'
               }`}
             >
               {React.cloneElement(item.icon, { size: isActive ? 24 : 22 })}
               <span className="text-[10px] font-bold">{item.label.split(' ')[0]}</span>
             </Link>
           );
        })}
        <button 
          onClick={() => setSidebarOpen(true)} 
          className={`flex flex-col items-center justify-center w-full h-full space-y-1 transition-colors ${
            sidebarOpen ? 'text-gray-900 dark:text-white' : 'text-gray-400 dark:text-gray-500'
          }`}
        >
          <Menu size={sidebarOpen ? 24 : 22} />
          <span className="text-[10px] font-bold">Menu</span>
        </button>
      </div>
      <Toaster position="top-right" toastOptions={{
        className: 'dark:bg-zinc-900 dark:text-white',
        style: {
          borderRadius: '12px',
          background: '#fff',
          color: '#363636',
          boxShadow: '0 4px 12px rgba(0,0,0,0.1)',
        }
      }} />
    </div>
  );
}

export default function App() {
  return (
    <BrowserRouter>
       <Layout>
         <Routes>
           <Route path="/admin" element={<Dashboard />} />
           <Route path="/admin/connections" element={<Connections />} />
           <Route path="/admin/system" element={<SystemStats />} />
           <Route path="/admin/network" element={<NetworkSettings />} />
           <Route path="/admin/session" element={<SessionSettings />} />
           <Route path="/admin/portal" element={<PortalSettings />} />
           <Route path="/admin/coins" element={<CoinSettings />} />
           <Route path="/admin/loyalty" element={<LoyaltySettings />} />
           <Route path="/admin/vouchers" element={<Vouchers />} />
           <Route path="/admin/devices" element={<Devices />} />
           <Route path="/admin/maintenance" element={<SystemSettings />} />
           <Route path="/admin/logs" element={<Logs />} />
           <Route path="/admin/user/:mac" element={<ManageUser />} />
         </Routes>
       </Layout>
    </BrowserRouter>
  );
}
