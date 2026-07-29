import React, { useEffect, useState } from 'react';
import { Save, Activity, Coins } from 'lucide-react';
import toast from 'react-hot-toast';
import ConfirmModal from '../components/ConfirmModal';

export default function CoinSettings() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [coinRates, setCoinRates] = useState({ 1: '', 5: '', 10: '', 20: '' });
  const [modalConfig, setModalConfig] = useState({ isOpen: false, title: '', message: '', onConfirm: null, type: 'danger' });

  const confirmAction = (title, message, onConfirm, type = 'danger') => {
    setModalConfig({ isOpen: true, title, message, onConfirm, type });
  };
  const closeModal = () => setModalConfig({ ...modalConfig, isOpen: false });

  useEffect(() => {
    fetch('/admin/api/dashboard_data')
      .then(res => res.json())
      .then(json => {
        setData(json);
        
        const parsedRates = { 1: '', 5: '', 10: '', 20: '' };
        if (json.coin_rates) {
          json.coin_rates.split(',').forEach(part => {
            const [coin, mins] = part.split(':');
            if (parsedRates[coin] !== undefined) {
              parsedRates[coin] = mins;
            }
          });
        }
        setCoinRates(parsedRates);
        setLoading(false);
      })
      .catch(err => {
        console.error(err);
        setLoading(false);
      });
  }, []);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    
    // Reconstruct coin_rates string
    const payload = {};
    const ratesArray = [];
    Object.keys(coinRates).forEach(coin => {
      if (coinRates[coin]) ratesArray.push(`${coin}:${coinRates[coin]}`);
    });
    payload.coin_rates = ratesArray.join(',');

    try {
      const res = await fetch('/admin/update_coin_settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (res.ok) {
        toast.success('Coin configuration saved successfully!');
        // Update local data context to not lose the string
        setData({...data, coin_rates: payload.coin_rates});
      } else {
        toast.error('Failed to save coin configuration.');
      }
    } catch (err) {
      toast.error('Error saving settings.');
    }
    setSaving(false);
  };

  if (loading) return (
    <div className="flex items-center justify-center h-64 text-gray-500 dark:text-gray-400">
      <Activity className="animate-spin w-8 h-8 mr-3" />
      <span>Loading coin settings...</span>
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
      <form onSubmit={handleSubmit} className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-6 md:p-8">
        <h3 className="flex items-center gap-2 text-base font-bold mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
          <Coins size={20} /> Pricing Structure
        </h3>
        
        <h4 className="text-sm font-bold uppercase tracking-widest text-gray-500 mb-4">Time Conversion (Coins to Minutes)</h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-10">
          {[1, 5, 10, 20].map(coin => (
            <div key={coin} className="space-y-1">
              <label className="text-xs font-bold text-gray-500 dark:text-gray-400">₱{coin} Coin → Minutes</label>
              <input 
                type="number" 
                value={coinRates[coin]}
                onChange={(e) => setCoinRates({...coinRates, [coin]: e.target.value})}
                placeholder="0"
                className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white text-sm"
              />
            </div>
          ))}
        </div>
        
        <div className="mt-10 flex justify-end">
          <button type="submit" disabled={saving} className="flex items-center gap-2 px-6 py-3 bg-black text-white dark:bg-white dark:text-black font-bold rounded hover:bg-gray-800 dark:hover:bg-gray-200 transition-colors disabled:opacity-50">
            <Save size={18} /> {saving ? 'Saving...' : 'Save Pricing'}
          </button>
        </div>
      </form>
    </div>
  );
}
