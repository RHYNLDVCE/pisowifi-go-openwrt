import React, { useEffect, useState } from 'react';
import { Ticket, Search, Clock, Zap, Save, CheckCircle, XCircle } from 'lucide-react';
import toast from 'react-hot-toast';

export default function Vouchers() {
  const [data, setData] = useState({ vouchers: [], voucher_enabled: false, voucher_min_time_minutes: 5, voucher_point_promos: [] });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [search, setSearch] = useState('');
  
  const [currentPage, setCurrentPage] = useState(1);
  const [sortField, setSortField] = useState('created_at');
  const [sortDirection, setSortDirection] = useState('desc');
  const itemsPerPage = 15;

  const fetchData = async () => {
    try {
      const res = await fetch('/admin/api/vouchers');
      const json = await res.json();
      if (!json.voucher_point_promos) json.voucher_point_promos = [];
      setData(json);
      setLoading(false);
    } catch (err) {
      console.error(err);
      toast.error('Failed to fetch vouchers');
      setLoading(false);
    }
  };

  const updatePromo = (index, field, value) => {
    const newData = { ...data };
    newData.voucher_point_promos[index][field] = field === 'name' ? value : Number(value);
    setData(newData);
  };

  const addPromo = () => {
    const newData = { ...data };
    if (!newData.voucher_point_promos) newData.voucher_point_promos = [];
    newData.voucher_point_promos.push({ id: Date.now(), name: "Convert Points", cost: 50, minutes: 30 });
    setData(newData);
  };

  const removePromo = (index) => {
    const newData = { ...data };
    newData.voucher_point_promos.splice(index, 1);
    setData(newData);
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleSaveSettings = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      const res = await fetch('/admin/update_voucher_settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          voucher_enabled: Boolean(data.voucher_enabled),
          voucher_min_time_minutes: parseInt(data.voucher_min_time_minutes) || 0,
          voucher_point_promos: data.voucher_point_promos || []
        })
      });
      if (res.ok) {
        toast.success('Voucher limits saved successfully!');
      } else {
        toast.error('Failed to save settings.');
      }
    } catch (err) {
      toast.error('Network error.');
    }
    setSaving(false);
  };

  const handleSort = (field) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection(field === 'status' ? 'asc' : 'desc');
    }
    setCurrentPage(1);
  };

  const sortedVouchers = [...(data.vouchers || [])]
    .filter(v => 
      v.code.toLowerCase().includes(search.toLowerCase()) || 
      v.created_by.toLowerCase().includes(search.toLowerCase()) || 
      (v.used_by && v.used_by.toLowerCase().includes(search.toLowerCase()))
    )
    .sort((a, b) => {
      if (sortField === 'created_at') {
        return sortDirection === 'asc' ? a.created_at - b.created_at : b.created_at - a.created_at;
      } else if (sortField === 'status') {
        const valA = a.status === 'active' ? 1 : 2;
        const valB = b.status === 'active' ? 1 : 2;
        if (valA !== valB) {
          return sortDirection === 'asc' ? valA - valB : valB - valA;
        }
        return b.created_at - a.created_at; // secondary sort
      }
      return 0;
    });

  const totalPages = Math.ceil(sortedVouchers.length / itemsPerPage) || 1;
  // Ensure current page is valid when filtering
  if (currentPage > totalPages && totalPages > 0) setCurrentPage(totalPages);
  
  const paginatedVouchers = sortedVouchers.slice((currentPage - 1) * itemsPerPage, currentPage * itemsPerPage);

  if (loading) {
    return <div className="p-8 text-center text-gray-500">Loading vouchers...</div>;
  }

  return (
    <div className="space-y-6 relative">
      
      {/* Settings Form */}
      <form onSubmit={handleSaveSettings} className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-6 md:p-8">
        <h3 className="flex items-center justify-between gap-2 text-base font-bold mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
          <div className="flex items-center gap-2">
            <Ticket size={20} /> Voucher Limits Settings
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" className="sr-only peer" checked={data.voucher_enabled} onChange={e => setData({...data, voucher_enabled: e.target.checked})} />
            <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer dark:bg-zinc-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-black dark:peer-checked:bg-white"></div>
          </label>
        </h3>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-6">
          Enable or disable the voucher feature, set minimum time limits, and configure point-to-voucher promotions.
        </p>

        <div className={!data.voucher_enabled ? 'opacity-50 pointer-events-none transition-opacity duration-300' : 'transition-opacity duration-300'}>
          <div className="mb-8">
            <div className="space-y-1 w-full md:w-1/2">
              <label className="text-xs font-bold text-gray-500 dark:text-gray-400 flex items-center gap-1"><Clock size={14} /> Minimum Time (Minutes)</label>
              <input 
                type="number" 
                value={data.voucher_min_time_minutes} 
                onChange={e => setData({...data, voucher_min_time_minutes: e.target.value})}
                min="1"
                className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white text-sm" 
              />
              <div className="text-[10px] text-gray-400 mt-1">Minimum active time required to generate a time voucher.</div>
            </div>
          </div>

          <div className="flex items-center justify-between mb-4">
             <h4 className="text-sm font-bold uppercase tracking-widest text-gray-500">Points to Voucher Promos</h4>
             <button type="button" onClick={addPromo} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-bold border border-gray-300 dark:border-zinc-700 rounded hover:bg-gray-100 dark:hover:bg-zinc-800 transition-colors">
               <Ticket size={14} /> Add Promo
             </button>
          </div>

          <div className="space-y-3 mb-8">
            {(!data.voucher_point_promos || data.voucher_point_promos.length === 0) ? (
               <div className="p-8 border-2 border-dashed border-gray-200 dark:border-zinc-800 rounded-md text-center text-gray-500 dark:text-gray-400">
                 No voucher point promos configured.
               </div>
            ) : (
               data.voucher_point_promos.map((promo, idx) => (
                 <div key={promo.id || idx} className="flex flex-col sm:flex-row gap-3 bg-gray-50 dark:bg-zinc-900/50 p-3 rounded-md border border-gray-200 dark:border-zinc-800 items-end">
                   <div className="flex-1 w-full">
                     <label className="block text-[10px] font-bold text-gray-500 uppercase tracking-widest mb-1">Promo Name</label>
                     <input type="text" value={promo.name} onChange={(e) => updatePromo(idx, 'name', e.target.value)} className="w-full px-3 py-1.5 bg-white dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none text-sm" />
                   </div>
                   <div className="w-full sm:w-28">
                     <label className="block text-[10px] font-bold text-gray-500 uppercase tracking-widest mb-1">Points Cost</label>
                     <input type="number" value={promo.cost} onChange={(e) => updatePromo(idx, 'cost', e.target.value)} className="w-full px-3 py-1.5 bg-white dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none text-sm" />
                   </div>
                   <div className="w-full sm:w-28">
                     <label className="block text-[10px] font-bold text-gray-500 uppercase tracking-widest mb-1">Time (Mins)</label>
                     <input type="number" value={promo.minutes} onChange={(e) => updatePromo(idx, 'minutes', e.target.value)} className="w-full px-3 py-1.5 bg-white dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none text-sm" />
                   </div>
                   <button type="button" onClick={() => removePromo(idx)} className="p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 rounded transition-colors" title="Delete Promo">
                     <XCircle size={18} />
                   </button>
                 </div>
               ))
            )}
          </div>
        </div>

        <div className="flex justify-end">
          <button type="submit" disabled={saving} className="flex items-center gap-2 px-6 py-3 bg-black text-white dark:bg-white dark:text-black font-bold rounded hover:bg-gray-800 dark:hover:bg-gray-200 transition-colors disabled:opacity-50">
            <Save size={18} /> {saving ? 'Saving...' : 'Save Settings'}
          </button>
        </div>
      </form>

      {/* Vouchers Table */}
      <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm overflow-hidden flex flex-col">
        <div className="p-4 border-b border-gray-200 dark:border-zinc-800 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <h2 className="text-lg font-bold text-gray-900 dark:text-white flex items-center gap-2">
             Generated Vouchers
          </h2>
          <div className="relative w-full sm:w-64">
            <input 
              type="text" 
              placeholder="Search code or MAC..." 
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white text-sm"
            />
            <Search size={16} className="absolute left-3 top-2.5 text-gray-400" />
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse min-w-[800px]">
            <thead>
              <tr className="bg-gray-50 dark:bg-zinc-900/50 border-b border-gray-200 dark:border-zinc-800">
                <th className="px-4 py-3 text-xs font-bold text-gray-500 uppercase tracking-wider">Code</th>
                <th className="px-4 py-3 text-xs font-bold text-gray-500 uppercase tracking-wider">Type / Value</th>
                <th 
                  className="px-4 py-3 text-xs font-bold text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-200 dark:hover:bg-zinc-800 transition-colors"
                  onClick={() => handleSort('status')}
                >
                  Status {sortField === 'status' && (sortDirection === 'asc' ? '↑' : '↓')}
                </th>
                <th className="px-4 py-3 text-xs font-bold text-gray-500 uppercase tracking-wider">Created By</th>
                <th 
                  className="px-4 py-3 text-xs font-bold text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-200 dark:hover:bg-zinc-800 transition-colors"
                  onClick={() => handleSort('created_at')}
                >
                  Created At {sortField === 'created_at' && (sortDirection === 'asc' ? '↑' : '↓')}
                </th>
                <th className="px-4 py-3 text-xs font-bold text-gray-500 uppercase tracking-wider">Used By</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-zinc-800">
              {paginatedVouchers.length === 0 ? (
                <tr>
                  <td colSpan="6" className="px-4 py-8 text-center text-sm text-gray-500">
                    No vouchers found.
                  </td>
                </tr>
              ) : (
                paginatedVouchers.map((v) => (
                  <tr key={v.code} className="hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors">
                    <td className="px-4 py-3 text-sm font-mono font-bold text-gray-900 dark:text-white">
                      {v.code}
                    </td>
                    <td className="px-4 py-3 text-sm">
                      {v.type === 'time' ? (
                        <span className="text-blue-600 font-bold">{v.value} Mins</span>
                      ) : (
                        <span className="text-amber-500 font-bold">{v.value} Pts</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-sm">
                      {v.status === 'active' ? (
                        <span className="inline-flex items-center gap-1 text-green-600 bg-green-100 dark:bg-green-900/30 px-2 py-1 rounded text-xs font-bold">
                          <CheckCircle size={14} /> Active
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 text-gray-500 bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded text-xs font-bold">
                          <XCircle size={14} /> Used
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-sm font-mono text-gray-600 dark:text-gray-400">
                      {v.created_by}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">
                      {new Date(v.created_at * 1000).toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-sm font-mono text-gray-600 dark:text-gray-400">
                      {v.used_by || '-'}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination Controls */}
        {totalPages > 1 && (
          <div className="p-4 border-t border-gray-200 dark:border-zinc-800 flex items-center justify-between">
            <span className="text-sm text-gray-500 dark:text-gray-400">
              Showing {(currentPage - 1) * itemsPerPage + 1} to {Math.min(currentPage * itemsPerPage, sortedVouchers.length)} of {sortedVouchers.length} vouchers
            </span>
            <div className="flex items-center gap-2">
              <button 
                onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                disabled={currentPage === 1}
                className="px-3 py-1 bg-gray-100 dark:bg-zinc-900 text-gray-600 dark:text-gray-300 rounded text-sm disabled:opacity-50 hover:bg-gray-200 dark:hover:bg-zinc-800"
              >
                Prev
              </button>
              <span className="text-sm font-bold">{currentPage} / {totalPages}</span>
              <button 
                onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                disabled={currentPage === totalPages}
                className="px-3 py-1 bg-gray-100 dark:bg-zinc-900 text-gray-600 dark:text-gray-300 rounded text-sm disabled:opacity-50 hover:bg-gray-200 dark:hover:bg-zinc-800"
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
