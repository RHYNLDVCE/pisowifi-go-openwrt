import React, { useEffect, useState } from 'react';
import { Save, Activity, Clock, Timer, PauseCircle, Gift, Gauge } from 'lucide-react';
import toast from 'react-hot-toast';
import ConfirmModal from '../components/ConfirmModal';

export default function NetworkSettings() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [modalConfig, setModalConfig] = useState({ isOpen: false, title: '', message: '', onConfirm: null, type: 'danger' });

  const confirmAction = (title, message, onConfirm, type = 'danger') => {
    setModalConfig({ isOpen: true, title, message, onConfirm, type });
  };
  const closeModal = () => setModalConfig({ ...modalConfig, isOpen: false });

  const [toggles, setToggles] = useState({
    speed_limit: false,
    sqm: false,
    udpPriority: false,
    gamingMode: false,
    openNat: false
  });

  useEffect(() => {
    fetch('/admin/api/dashboard_data')
      .then(res => res.json())
      .then(json => {
        setData(json);
        setToggles({
          speed_limit: json.speed_limit_enabled,
          sqm: json.sqm_enabled,
          udpPriority: json.udp_priority_enabled,
          gamingMode: json.gaming_mode_enabled,
          openNat: json.open_nat_enabled
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
    
    payload.speed_limit_toggle = formData.get('speed_limit_toggle') ? 'on' : '';
    payload.sqm_enabled = formData.get('sqm_enabled') ? 'on' : '';
    payload.udp_priority = formData.get('udp_priority') ? 'on' : '';
    payload.gaming_mode = formData.get('gaming_mode') ? 'on' : '';
    payload.open_nat = formData.get('open_nat') ? 'on' : '';
    payload.custom_ttl = formData.get('custom_ttl') || (data.custom_ttl ?? 1);

    try {
      const res = await fetch('/admin/update_network_settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (res.ok) toast.success('Network settings saved successfully!');
      else toast.error('Failed to save settings.');
    } catch (err) {
      toast.error('Error saving settings.');
    }
    setSaving(false);
  };

  if (loading) return (
    <div className="flex items-center justify-center h-64 text-gray-500 dark:text-gray-400">
      <Activity className="animate-spin w-8 h-8 mr-3" />
      <span>Loading network settings...</span>
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

        {/* Global Speed Limit */}
        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-6 md:p-8">
          <div className="flex items-start sm:items-center justify-between gap-4 mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
             <h3 className="text-base font-bold text-gray-900 dark:text-white flex items-center flex-wrap gap-x-2 gap-y-1">
               <span>Global Speed Limit</span>
               <div tabIndex={0} className="group relative flex items-center justify-center text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 focus:text-gray-600 dark:focus:text-gray-300 cursor-help outline-none">
                 <div className="w-4 h-4 rounded-full border-2 border-current flex items-center justify-center text-[10px] font-bold">i</div>
                 <div className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 sm:absolute sm:top-full sm:bottom-auto sm:mt-2 w-[85vw] sm:w-72 p-3 bg-gray-900 text-white text-xs rounded-lg shadow-2xl opacity-0 pointer-events-none group-hover:opacity-100 group-focus:opacity-100 transition-opacity z-[100] text-center leading-relaxed">
                   Sets the maximum bandwidth allowed for EACH individual user. This does NOT limit the total speed of your ethernet port or USB LAN adapter.
                 </div>
               </div>
             </h3>
             <label className="relative flex items-center shrink-0 cursor-pointer">
               <input type="checkbox" name="speed_limit_toggle" checked={toggles.speed_limit} onChange={() => handleToggle('speed_limit')} className="peer sr-only" />
               <div className="w-11 h-6 bg-gray-300 dark:bg-zinc-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-black dark:peer-checked:bg-blue-600"></div>
             </label>
          </div>
          
          <div className={!toggles.speed_limit ? 'opacity-50 pointer-events-none transition-opacity duration-300' : 'transition-opacity duration-300'}>
             <div className="max-w-sm space-y-1">
                <label className="text-xs font-bold text-gray-500 dark:text-gray-400">Speed Limit (Mbps)</label>
                <input 
                  type="number" 
                  name="speed_limit_val" 
                  defaultValue={data.global_speed_limit} 
                  className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white"
                />
             </div>
          </div>
        </div>

        {/* Smart Queue Management (SQM) */}
        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-6 md:p-8">
          <div className="flex items-start sm:items-center justify-between gap-4 mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
             <h3 className="text-base font-bold text-gray-900 dark:text-white flex items-center flex-wrap gap-x-2 gap-y-1">
               <span>Smart Queue Management (SQM)</span>
               <div tabIndex={0} className="group relative flex items-center justify-center text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 focus:text-gray-600 dark:focus:text-gray-300 cursor-help outline-none">
                 <div className="w-4 h-4 rounded-full border-2 border-current flex items-center justify-center text-[10px] font-bold">i</div>
                 <div className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 sm:absolute sm:top-auto sm:bottom-full sm:mb-2 w-[85vw] sm:w-72 p-3 bg-gray-900 text-white text-xs rounded-lg shadow-2xl opacity-0 pointer-events-none group-hover:opacity-100 group-focus:opacity-100 transition-opacity z-[100] text-center leading-relaxed">
                   Prevents bufferbloat by smoothly managing your overall WAN hardware traffic. Keeps your network ping low even when overall usage is high.
                 </div>
               </div>
             </h3>
             <label className="relative flex items-center shrink-0 cursor-pointer">
               <input type="checkbox" name="sqm_enabled" checked={toggles.sqm} onChange={() => handleToggle('sqm')} className="peer sr-only" />
               <div className="w-11 h-6 bg-gray-300 dark:bg-zinc-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-black dark:peer-checked:bg-blue-600"></div>
             </label>
          </div>
          
          <div className={!toggles.sqm ? 'opacity-50 pointer-events-none transition-opacity duration-300' : 'transition-opacity duration-300'}>
             <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
               <div className="space-y-1">
                 <label className="text-xs font-bold text-gray-500 dark:text-gray-400">ISP Download Speed (Mbps)</label>
                 <input 
                   type="number" 
                   name="sqm_download_mbps" 
                   defaultValue={data.sqm_download_mbps} 
                   className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white"
                 />
                 <p className="text-[10px] text-gray-500 mt-1">Set to 95% of your true maximum ISP download speed.</p>
               </div>
               <div className="space-y-1">
                 <label className="text-xs font-bold text-gray-500 dark:text-gray-400">ISP Upload Speed (Mbps)</label>
                 <input 
                   type="number" 
                   name="sqm_upload_mbps" 
                   defaultValue={data.sqm_upload_mbps} 
                   className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white"
                 />
                 <p className="text-[10px] text-gray-500 mt-1">Set to 95% of your true maximum ISP upload speed.</p>
               </div>
             </div>
          </div>
        </div>

        {/* UDP Priority Optimization */}
        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-6 md:p-8">
          <div className="flex items-start sm:items-center justify-between gap-4">
             <h3 className="text-base font-bold text-gray-900 dark:text-white flex items-center flex-wrap gap-x-2 gap-y-1">
               <span>UDP Priority Optimization</span>
               <div tabIndex={0} className="group relative flex items-center justify-center text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 focus:text-gray-600 dark:focus:text-gray-300 cursor-help outline-none">
                 <div className="w-4 h-4 rounded-full border-2 border-current flex items-center justify-center text-[10px] font-bold">i</div>
                 <div className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 sm:absolute sm:top-auto sm:bottom-full sm:mb-2 w-[85vw] sm:w-72 p-3 bg-gray-900 text-white text-xs rounded-lg shadow-2xl opacity-0 pointer-events-none group-hover:opacity-100 group-focus:opacity-100 transition-opacity z-[100] text-center leading-relaxed">
                   Prioritizes specific gaming ports and small UDP packets (≤256 bytes) typical for games like Mobile Legends with VIP priority. Heavy streams and voice/video calls are intentionally excluded to keep gaming ping low.
                 </div>
               </div>
             </h3>
             <label className="relative flex items-center shrink-0 cursor-pointer">
               <input type="checkbox" name="udp_priority" checked={toggles.udpPriority} onChange={() => handleToggle('udpPriority')} className="peer sr-only" />
               <div className="w-11 h-6 bg-gray-300 dark:bg-zinc-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-black dark:peer-checked:bg-blue-600"></div>
             </label>
          </div>
        </div>

        {/* Gaming Mode */}
        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-6 md:p-8">
          <div className="flex items-start sm:items-center justify-between gap-4">
             <h3 className="text-base font-bold text-gray-900 dark:text-white flex items-center flex-wrap gap-x-2 gap-y-1">
               <span>Gaming Mode</span>
               <div tabIndex={0} className="group relative flex items-center justify-center text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 focus:text-gray-600 dark:focus:text-gray-300 cursor-help outline-none">
                 <div className="w-4 h-4 rounded-full border-2 border-current flex items-center justify-center text-[10px] font-bold">i</div>
                 <div className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 sm:absolute sm:top-auto sm:bottom-full sm:mb-2 w-[85vw] sm:w-80 p-3 bg-gray-900 text-white text-xs rounded-lg shadow-2xl opacity-0 pointer-events-none group-hover:opacity-100 group-focus:opacity-100 transition-opacity z-[100] text-center leading-relaxed">
                   Enables CAKE SQM specifically for each individual user. It creates priority lanes inside their personal speed limit, ensuring their gaming traffic skips ahead of their own downloads.
                 </div>
               </div>
             </h3>
             <label className="relative flex items-center shrink-0 cursor-pointer">
               <input type="checkbox" name="gaming_mode" checked={toggles.gamingMode} onChange={() => handleToggle('gamingMode')} className="peer sr-only" />
               <div className="w-11 h-6 bg-gray-300 dark:bg-zinc-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-black dark:peer-checked:bg-blue-600"></div>
             </label>
          </div>
        </div>

        {/* Open NAT */}
        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-6 md:p-8">
          <div className="flex items-start sm:items-center justify-between gap-4">
             <h3 className="text-base font-bold text-gray-900 dark:text-white flex items-center flex-wrap gap-x-2 gap-y-1">
               <span>Open NAT (Gaming)</span>
               <div tabIndex={0} className="group relative flex items-center justify-center text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 focus:text-gray-600 dark:focus:text-gray-300 cursor-help outline-none">
                 <div className="w-4 h-4 rounded-full border-2 border-current flex items-center justify-center text-[10px] font-bold">i</div>
                 <div className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 sm:absolute sm:top-auto sm:bottom-full sm:mb-2 w-[85vw] sm:w-72 p-3 bg-gray-900 text-white text-xs rounded-lg shadow-2xl opacity-0 pointer-events-none group-hover:opacity-100 group-focus:opacity-100 transition-opacity z-[100] text-center leading-relaxed">
                   Opens up UPnP routing to allow Console gaming (Playstation, Xbox) to connect more freely without Strict NAT type issues.
                 </div>
               </div>
             </h3>
             <label className="relative flex items-center shrink-0 cursor-pointer">
               <input type="checkbox" name="open_nat" checked={toggles.openNat} onChange={() => handleToggle('openNat')} className="peer sr-only" />
               <div className="w-11 h-6 bg-gray-300 dark:bg-zinc-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-black dark:peer-checked:bg-blue-600"></div>
             </label>
          </div>
        </div>

        {/* Tethering Override (TTL) */}
        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-6 md:p-8">
          <div className="flex items-start sm:items-center justify-between gap-4 mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
             <h3 className="text-base font-bold text-gray-900 dark:text-white flex items-center flex-wrap gap-x-2 gap-y-1">
               <span>Tethering Override (TTL)</span>
               <div tabIndex={0} className="group relative flex items-center justify-center text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 focus:text-gray-600 dark:focus:text-gray-300 cursor-help outline-none">
                 <div className="w-4 h-4 rounded-full border-2 border-current flex items-center justify-center text-[10px] font-bold">i</div>
                 <div className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 sm:absolute sm:top-auto sm:bottom-full sm:mb-2 w-[85vw] sm:w-72 p-3 bg-gray-900 text-white text-xs rounded-lg shadow-2xl opacity-0 pointer-events-none group-hover:opacity-100 group-focus:opacity-100 transition-opacity z-[100] text-center leading-relaxed">
                   Modifies the TTL (Time To Live) of packets leaving the router. Set to 1 to attempt blocking users from tethering/hotspotting their connection to other devices.
                 </div>
               </div>
             </h3>
          </div>
          <div className="max-w-sm space-y-1">
             <label className="text-xs font-bold text-gray-500 dark:text-gray-400">TTL Value</label>
             <input 
               type="number" 
               name="custom_ttl" 
               defaultValue={data.custom_ttl ?? 1} 
               className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white"
             />
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
