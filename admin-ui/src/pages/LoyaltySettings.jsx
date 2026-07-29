import React, { useEffect, useState } from 'react';
import { Save, Activity, Award, Plus, Trash2 } from 'lucide-react';
import toast from 'react-hot-toast';
import ConfirmModal from '../components/ConfirmModal';

export default function LoyaltySettings() {
  const [pointsConfig, setPointsConfig] = useState(null);
  const [loading, setLoading] = useState(true);
  const [savingPoints, setSavingPoints] = useState(false);
  const [modalConfig, setModalConfig] = useState({ isOpen: false, title: '', message: '', onConfirm: null, type: 'danger' });

  const confirmAction = (title, message, onConfirm, type = 'danger') => {
    setModalConfig({ isOpen: true, title, message, onConfirm, type });
  };
  const closeModal = () => setModalConfig({ ...modalConfig, isOpen: false });

  useEffect(() => {
    fetch('/admin/get_points_config')
      .then(res => res.json())
      .then(data => {
        setPointsConfig(data);
        setLoading(false);
      })
      .catch(err => {
        console.error(err);
        setLoading(false);
      });
  }, []);

  const handlePointsSubmit = async (e) => {
    e.preventDefault();
    setSavingPoints(true);
    
    try {
      const res = await fetch('/admin/save_points_config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(pointsConfig)
      });
      if (res.ok) toast.success('Points configuration saved successfully!');
      else toast.error('Failed to save points configuration.');
    } catch (err) {
      toast.error('Error saving points configuration.');
    }
    setSavingPoints(false);
  };

  const updatePromo = (index, field, value) => {
    const newConfig = { ...pointsConfig };
    newConfig.promos[index][field] = field === 'name' ? value : Number(value);
    setPointsConfig(newConfig);
  };

  const addPromo = () => {
    const newConfig = { ...pointsConfig };
    if (!newConfig.promos) newConfig.promos = [];
    newConfig.promos.push({ id: Date.now(), name: "New Promo", cost: 10, minutes: 60 });
    setPointsConfig(newConfig);
  };

  const removePromo = (index) => {
    const newConfig = { ...pointsConfig };
    newConfig.promos.splice(index, 1);
    setPointsConfig(newConfig);
  };

  if (loading) return (
    <div className="flex items-center justify-center h-64 text-gray-500 dark:text-gray-400">
      <Activity className="animate-spin w-8 h-8 mr-3" />
      <span>Loading loyalty settings...</span>
    </div>
  );

  if (!pointsConfig) return <div className="text-red-500">Error loading settings.</div>;

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
      
      <form onSubmit={handlePointsSubmit} className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-6 md:p-8">
        <h3 className="flex items-center justify-between gap-2 text-base font-bold mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
          <div className="flex items-center gap-2">
            <Award size={20} /> Rewards System
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input 
              type="checkbox" 
              checked={pointsConfig.enabled} 
              onChange={(e) => setPointsConfig({...pointsConfig, enabled: e.target.checked})}
              className="peer sr-only" 
            />
            <div className="w-11 h-6 bg-gray-300 dark:bg-zinc-700 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-black/20 dark:peer-focus:ring-blue-600/20 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-black dark:peer-checked:bg-blue-600"></div>
          </label>
        </h3>

        <div className={!pointsConfig.enabled ? 'opacity-50 pointer-events-none transition-opacity duration-300' : 'transition-opacity duration-300'}>
          <h4 className="text-sm font-bold uppercase tracking-widest text-gray-500 mb-4">Points Conversion (Coins to Points)</h4>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-10">
            {[1, 5, 10, 20].map(coin => (
              <div key={coin} className="space-y-1">
                <label className="text-xs font-bold text-gray-500 dark:text-gray-400">₱{coin} Coin → Points</label>
                <input 
                  type="number" 
                  step="0.1"
                  value={pointsConfig.coin_map[coin.toString()] || ''}
                  onChange={(e) => setPointsConfig({
                    ...pointsConfig, 
                    coin_map: { ...pointsConfig.coin_map, [coin.toString()]: parseFloat(e.target.value) || 0 }
                  })}
                  className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white"
                />
              </div>
            ))}
          </div>

          <div className="flex items-center justify-between mb-4">
             <h4 className="text-sm font-bold uppercase tracking-widest text-gray-500">Redemption Promos</h4>
             <button type="button" onClick={addPromo} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-bold border border-gray-300 dark:border-zinc-700 rounded hover:bg-gray-100 dark:hover:bg-zinc-800 transition-colors">
               <Plus size={14} /> Add Promo
             </button>
          </div>

          <div className="space-y-3 mb-8">
            {(!pointsConfig.promos || pointsConfig.promos.length === 0) ? (
               <div className="p-8 border-2 border-dashed border-gray-200 dark:border-zinc-800 rounded-md text-center text-gray-500 dark:text-gray-400">
                 No active promos. Click 'Add Promo' to create one.
               </div>
            ) : (
               pointsConfig.promos.map((promo, idx) => (
                 <div key={promo.id || idx} className="flex flex-col sm:flex-row gap-3 bg-gray-50 dark:bg-zinc-900/50 p-3 rounded-md border border-gray-200 dark:border-zinc-800 items-end">
                   <div className="flex-1 w-full">
                     <label className="block text-[10px] font-bold text-gray-500 uppercase tracking-widest mb-1">Promo Title</label>
                     <input type="text" value={promo.name} onChange={(e) => updatePromo(idx, 'name', e.target.value)} className="w-full px-3 py-1.5 bg-white dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none text-sm" />
                   </div>
                   <div className="w-full sm:w-24">
                     <label className="block text-[10px] font-bold text-gray-500 uppercase tracking-widest mb-1">Points Cost</label>
                     <input type="number" value={promo.cost} onChange={(e) => updatePromo(idx, 'cost', e.target.value)} className="w-full px-3 py-1.5 bg-white dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none text-sm" />
                   </div>
                   <div className="w-full sm:w-24">
                     <label className="block text-[10px] font-bold text-gray-500 uppercase tracking-widest mb-1">Time (Mins)</label>
                     <input type="number" value={promo.minutes} onChange={(e) => updatePromo(idx, 'minutes', e.target.value)} className="w-full px-3 py-1.5 bg-white dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none text-sm" />
                   </div>
                   <button type="button" onClick={() => removePromo(idx)} className="p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 rounded transition-colors" title="Delete Promo">
                     <Trash2 size={18} />
                   </button>
                 </div>
               ))
            )}
          </div>
        </div>

        <div className="flex justify-end pt-6 border-t border-gray-200 dark:border-zinc-800">
           <button type="submit" disabled={savingPoints} className="flex items-center gap-2 px-6 py-3 bg-black text-white dark:bg-white dark:text-black font-bold rounded hover:bg-gray-800 dark:hover:bg-gray-200 transition-colors disabled:opacity-50">
             <Save size={18} /> {savingPoints ? 'Saving...' : 'Save Loyalty Configuration'}
           </button>
        </div>
      </form>
    </div>
  );
}
