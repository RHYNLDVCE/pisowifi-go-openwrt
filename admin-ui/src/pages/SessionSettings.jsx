import React, { useEffect, useState } from 'react';
import { Save, Activity, PauseCircle, Gift } from 'lucide-react';
import toast from 'react-hot-toast';
import ConfirmModal from '../components/ConfirmModal';

export default function SessionSettings() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [modalConfig, setModalConfig] = useState({ isOpen: false, title: '', message: '', onConfirm: null, type: 'danger' });

  const confirmAction = (title, message, onConfirm, type = 'danger') => {
    setModalConfig({ isOpen: true, title, message, onConfirm, type });
  };
  const closeModal = () => setModalConfig({ ...modalConfig, isOpen: false });

  const [toggles, setToggles] = useState({
    auto_pause: false,
    free_time: false
  });

  useEffect(() => {
    fetch('/admin/api/dashboard_data')
      .then(res => res.json())
      .then(json => {
        setData(json);
        setToggles({
          auto_pause: json.auto_pause_enabled,
          free_time: json.free_time_enabled
        });
        setLoading(false);
      })
      .catch(err => {
        console.error(err);
        setLoading(false);
      });
  }, []);

  const handleToggle = (key) => setToggles(prev => ({ ...prev, [key]: !prev[key] }));

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    
    const formData = new FormData(e.target);
    const payload = Object.fromEntries(formData.entries());
    
    payload.free_time_toggle = formData.get('free_time_toggle') ? 'on' : '';
    payload.auto_pause = formData.get('auto_pause') ? 'on' : '';

    try {
      const res = await fetch('/admin/update_session_settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (res.ok) toast.success('Session settings saved successfully!');
      else toast.error('Failed to save settings.');
    } catch (err) {
      toast.error('Error saving settings.');
    }
    setSaving(false);
  };

  if (loading) return (
    <div className="flex items-center justify-center h-64 text-gray-500 dark:text-gray-400">
      <Activity className="animate-spin w-8 h-8 mr-3" />
      <span>Loading session settings...</span>
    </div>
  );

  if (!data) return <div className="text-red-500">Error loading settings.</div>;

  return (
    <div className="space-y-6 relative">
      <ConfirmModal 
        isOpen={modalConfig.isOpen}
        title={modalConfig.title}
        message={modalConfig.message}
        type={modalConfig.type}
        onConfirm={modalConfig.onConfirm}
        onClose={closeModal}
      />
      
      <form onSubmit={handleSubmit} className="space-y-6">
        
        {/* Session & Auto-Pause */}
        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-6 md:p-8">
          <div className="flex items-start sm:items-center justify-between gap-4 mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
             <h3 className="text-base font-bold text-gray-900 dark:text-white flex items-center flex-wrap gap-x-2 gap-y-1">
               <span>Automatic Pause (Idle Timeout)</span>
               <div tabIndex={0} className="group relative flex items-center justify-center text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 focus:text-gray-600 dark:focus:text-gray-300 cursor-help outline-none">
                 <div className="w-4 h-4 rounded-full border-2 border-current flex items-center justify-center text-[10px] font-bold">i</div>
                 <div className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 sm:absolute sm:top-full sm:bottom-auto sm:mt-2 w-[85vw] sm:w-72 p-3 bg-gray-900 text-white text-xs rounded-lg shadow-2xl opacity-0 pointer-events-none group-hover:opacity-100 group-focus:opacity-100 transition-opacity z-[100] text-center leading-relaxed">
                   Automatically pauses a user's time if they disconnect from the network or remain idle for the set duration.
                 </div>
               </div>
             </h3>
             <label className="relative flex items-center shrink-0 cursor-pointer">
               <input type="checkbox" name="auto_pause" checked={toggles.auto_pause} onChange={() => handleToggle('auto_pause')} className="peer sr-only" />
               <div className="w-11 h-6 bg-gray-300 dark:bg-zinc-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-black dark:peer-checked:bg-blue-600"></div>
             </label>
          </div>
          
          <div className={!toggles.auto_pause ? 'opacity-50 pointer-events-none transition-opacity duration-300' : 'transition-opacity duration-300'}>
             <div className="max-w-sm space-y-1">
                <label className="text-xs font-bold text-gray-500 dark:text-gray-400">Inactive Disconnect (Seconds)</label>
                <input 
                  type="number" 
                  name="inactive_timeout" 
                  defaultValue={data.inactive_timeout} 
                  className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white"
                />
             </div>
          </div>
        </div>

        {/* Free Time Trial */}
        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-6 md:p-8">
          <div className="flex items-start sm:items-center justify-between gap-4 mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
             <h3 className="text-base font-bold text-gray-900 dark:text-white flex items-center flex-wrap gap-x-2 gap-y-1">
               <span>Free Time Access</span>
               <div tabIndex={0} className="group relative flex items-center justify-center text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 focus:text-gray-600 dark:focus:text-gray-300 cursor-help outline-none">
                 <div className="w-4 h-4 rounded-full border-2 border-current flex items-center justify-center text-[10px] font-bold">i</div>
                 <div className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 sm:absolute sm:top-auto sm:bottom-full sm:mb-2 w-[85vw] sm:w-72 p-3 bg-gray-900 text-white text-xs rounded-lg shadow-2xl opacity-0 pointer-events-none group-hover:opacity-100 group-focus:opacity-100 transition-opacity z-[100] text-center leading-relaxed">
                   Allows users to claim limited free internet access. Great for promotions or short-term use.
                 </div>
               </div>
             </h3>
             <label className="relative flex items-center shrink-0 cursor-pointer">
               <input type="checkbox" name="free_time_toggle" checked={toggles.free_time} onChange={() => handleToggle('free_time')} className="peer sr-only" />
               <div className="w-11 h-6 bg-gray-300 dark:bg-zinc-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-black dark:peer-checked:bg-blue-600"></div>
             </label>
          </div>
          
          <div className={!toggles.free_time ? 'opacity-50 pointer-events-none transition-opacity duration-300' : 'transition-opacity duration-300'}>
             <div className="max-w-sm space-y-1">
                <label className="text-xs font-bold text-gray-500 dark:text-gray-400">Free Time Duration (Minutes)</label>
                <input 
                  type="number" 
                  name="free_time_duration" 
                  defaultValue={data.free_time_duration} 
                  className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white" 
                />
             </div>
          </div>
        </div>
        
        <div className="flex justify-end pt-6 border-t border-gray-200 dark:border-zinc-800">
          <button type="submit" disabled={saving} className="flex items-center gap-2 px-6 py-3 bg-black text-white dark:bg-white dark:text-black font-bold rounded hover:bg-gray-800 dark:hover:bg-gray-200 transition-colors disabled:opacity-50">
            <Save size={18} /> {saving ? 'Saving...' : 'Save Settings'}
          </button>
        </div>
      </form>
    </div>
  );
}
