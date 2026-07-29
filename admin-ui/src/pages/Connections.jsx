import React, { useEffect, useState } from 'react';
import { Activity, Search, Users, ChevronDown, ChevronLeft, ChevronRight, X } from 'lucide-react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import CustomSelect from '../components/CustomSelect';

export default function Connections() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const searchQuery = searchParams.get('q') || '';
  const sortBy = searchParams.get('sort') || 'status';
  const currentPage = parseInt(searchParams.get('page') || '1', 10);

  const setSearchQuery = (val) => {
    setSearchParams(prev => {
      prev.set('page', '1');
      if (val) prev.set('q', val);
      else prev.delete('q');
      return prev;
    }, { replace: true });
  };

  const setSortBy = (val) => {
    setSearchParams(prev => {
      prev.set('page', '1');
      if (val !== 'status') prev.set('sort', val);
      else prev.delete('sort');
      return prev;
    }, { replace: true });
  };

  const setCurrentPage = (val) => {
    setSearchParams(prev => {
      prev.set('page', val.toString());
      return prev;
    }, { replace: true });
  };

  useEffect(() => {
    setLoading(true);
    const timer = setTimeout(() => {
      const params = new URLSearchParams();
      if (searchQuery) params.set('search', searchQuery);
      params.set('page', currentPage);
      params.set('sort', sortBy);
      
      fetch(`/admin/api/dashboard_data?${params.toString()}`)
        .then(res => res.json())
        .then(json => {
          setData(json);
          setLoading(false);
        })
        .catch(err => {
          console.error("Failed to fetch dashboard data", err);
          setLoading(false);
        });
    }, 300);
    return () => clearTimeout(timer);
  }, [searchQuery, sortBy, currentPage]);

  if (loading && !data) {
    return (
      <div className="flex items-center justify-center h-64 text-gray-500 dark:text-gray-400">
        <Activity className="animate-spin w-8 h-8 mr-3" />
        <span>Loading connections...</span>
      </div>
    );
  }

  if (!data) {
    return <div className="text-red-500">Error loading data.</div>;
  }

  const { users, total_pages, current_page, total_filtered, total_users, active_users } = data;

  const statusOrder = { connected: 1, paused: 2, expired: 3, new: 4 };
  const getStatusWeight = (status) => statusOrder[status?.toLowerCase()] || 5;

  // The backend already filtered and paginated the data, so we only have ~10 users here.
  // We still need to sort them because JS objects (users map) don't guarantee key order.
  const displayMacs = Object.keys(users).sort((a, b) => {
    const ua = users[a];
    const ub = users[b];
    if (sortBy === 'status') {
      const weightA = getStatusWeight(ua.status);
      const weightB = getStatusWeight(ub.status);
      if (weightA !== weightB) return weightA - weightB;
      return (ub.time || 0) - (ua.time || 0);
    }
    if (sortBy === 'points') {
      return (ub.points || 0) - (ua.points || 0);
    }
    return (ub.time || 0) - (ua.time || 0);
  });

  const ITEMS_PER_PAGE = 10;
  // Ensure total_filtered falls back safely
  const totalItems = total_filtered !== undefined ? total_filtered : total_users;
  const safeTotalPages = total_pages || 1;
  const safeCurrentPage = current_page || 1;

  return (
    <div className="space-y-6">
      <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm flex flex-col p-6 md:p-8">
        <div className="flex flex-wrap gap-4 justify-between items-center mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
          <div className="flex items-center gap-3">
            <div className="hidden sm:flex">
               <Users size={20} className="text-gray-900 dark:text-white" />
            </div>
            <div>
              <h3 className="text-base font-bold text-gray-900 dark:text-white">Active Connections</h3>
            </div>
            <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-semibold border border-gray-200 dark:border-zinc-700 bg-gray-50 dark:bg-zinc-900 text-gray-900 dark:text-white ml-2 shrink-0 whitespace-nowrap">
               <span className="w-1.5 h-1.5 rounded-full bg-green-500 shrink-0"></span>
               {active_users} Active
            </span>
          </div>
          <div className="flex flex-col sm:flex-row items-center gap-3 w-full sm:w-auto pb-2 sm:pb-0">
            <CustomSelect
              value={sortBy}
              onChange={(val) => setSortBy(val)}
              options={[
                { value: 'status', label: 'Order by Status' },
                { value: 'time', label: 'Order by Time' },
                { value: 'points', label: 'Order by Points' }
              ]}
              className="w-full sm:w-48"
            />
            <div className="relative w-full sm:w-auto">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 w-4 h-4" />
              <input 
                type="text" 
                placeholder="Search MAC, IP, or Name..." 
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
                className="pl-9 pr-10 py-2 text-sm bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white w-full sm:w-64 transition-all" 
              />
              {searchQuery && (
                <button
                  onClick={() => setSearchQuery('')}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                >
                  <X className="w-4 h-4" />
                </button>
              )}
            </div>
          </div>
        </div>
        
        {/* Desktop Table View */}
        <div className="hidden md:block overflow-x-auto border border-gray-200 dark:border-zinc-800 rounded">
          <table className="w-full text-left border-collapse relative">
            <thead className="sticky top-0 bg-gray-50 dark:bg-zinc-900/50 z-10 border-b border-gray-200 dark:border-zinc-800">
              <tr>
                <th className="pl-6 pr-4 py-3 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider w-12">#</th>
                <th className="px-4 py-3 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Device</th>
                <th className="px-4 py-3 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">IP / MAC</th>
                <th className="px-4 py-3 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Status</th>
                <th className="pl-4 pr-6 py-3 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider text-right">Time Left</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-zinc-800">
              {displayMacs.length === 0 ? (
                <tr>
                  <td colSpan="5" className="py-12 text-center text-gray-500">
                    {searchQuery ? "No matching devices found." : "No active devices connected."}
                  </td>
                </tr>
              ) : (
                displayMacs.map((mac, idx) => {
                  const u = users[mac];
                  const absoluteIdx = (safeCurrentPage - 1) * ITEMS_PER_PAGE + idx + 1;
                  return (
                    <tr key={mac} className={`transition-colors group cursor-pointer ${
                      u.status === 'connected' ? 'bg-white dark:bg-zinc-950 hover:bg-gray-50 dark:hover:bg-zinc-900' :
                      u.status === 'paused' ? 'bg-gray-50/50 dark:bg-zinc-900/50 hover:bg-gray-100/50 dark:hover:bg-zinc-800/50' :
                      'hover:bg-gray-50/50 dark:hover:bg-zinc-900/20'
                    }`} onClick={() => navigate(`/admin/user/${mac}`)}>
                      <td className="pl-6 pr-4 py-3 text-sm font-medium text-gray-500">
                        {absoluteIdx}
                      </td>
                      <td className="px-4 py-3">
                        <div className="font-semibold text-sm text-gray-900 dark:text-white transition-colors">{u.device_name || 'Unknown Device'}</div>
                      </td>
                      <td className="px-4 py-3">
                        <div className="text-sm font-medium text-gray-700 dark:text-gray-300">{u.ip}</div>
                        <div className="text-xs text-gray-400 font-mono mt-0.5">{mac}</div>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex items-center px-2 py-1 rounded text-[10px] font-bold uppercase tracking-wider border ${
                          u.status === 'connected' ? 'border-green-200 dark:border-green-900/50 text-green-700 dark:text-green-400 bg-green-50 dark:bg-green-500/10' :
                          u.status === 'paused' ? 'border-amber-200 dark:border-amber-900/50 text-amber-700 dark:text-amber-400 bg-amber-50 dark:bg-amber-500/10' :
                          'border-red-200 dark:border-red-900/50 text-red-700 dark:text-red-400 bg-red-50 dark:bg-red-500/10'
                        }`}>
                          {u.status}
                        </span>
                      </td>
                      <td className="pl-4 pr-6 py-3 text-right">
                        <div className="flex flex-wrap items-center justify-end gap-2">
                           {u.points > 0 && (
                             <span className="border border-amber-200 dark:border-amber-900/50 bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400 px-2 py-1 rounded text-xs font-bold">
                               ★ {u.points}
                             </span>
                           )}
                           <div className="font-semibold text-sm text-gray-900 dark:text-white">{u.time > 0 ? u.time_formatted : '0s'}</div>
                        </div>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>

        {/* Mobile List View (Flat, Edge-to-Edge) */}
        <div className="md:hidden flex flex-col -mx-6 sm:-mx-8 border-y border-gray-200 dark:border-zinc-800">
          {displayMacs.length === 0 ? (
            <div className="py-12 text-center text-gray-500">
               {searchQuery ? "No matching devices found." : "No active devices connected."}
            </div>
          ) : (
            displayMacs.map((mac, idx) => {
              const u = users[mac];
              const absoluteIdx = (safeCurrentPage - 1) * ITEMS_PER_PAGE + idx + 1;
              return (
                <div key={mac} onClick={() => navigate(`/admin/user/${mac}`)} className={`flex flex-col py-3 px-6 border-b border-gray-200 dark:border-zinc-800 last:border-0 transition-colors cursor-pointer ${
                  u.status === 'connected' ? 'bg-white dark:bg-zinc-950 active:bg-gray-50 dark:active:bg-zinc-900' :
                  u.status === 'paused' ? 'bg-gray-50/50 dark:bg-zinc-900/50 active:bg-gray-100/50 dark:active:bg-zinc-800/50' :
                  'active:bg-gray-50 dark:active:bg-zinc-900'
                }`}>
                  <div className="flex justify-between items-start">
                    <div className="flex items-center gap-2">
                      <div className="text-gray-500 font-medium text-xs">{absoluteIdx}.</div>
                      <div className="flex flex-col">
                        <div className="font-bold text-gray-900 dark:text-white text-[15px] leading-tight truncate max-w-[160px]">{u.device_name || 'Unknown Device'}</div>
                        <div className="text-[9px] text-gray-400 font-mono mt-0.5">{mac}</div>
                      </div>
                    </div>
                    <span className={`inline-flex items-center px-1.5 py-0.5 rounded border text-[9px] font-bold uppercase tracking-wider shrink-0 ${
                      u.status === 'connected' ? 'border-green-200 dark:border-green-900/50 text-green-700 dark:text-green-400 bg-green-50 dark:bg-green-500/10' :
                      u.status === 'paused' ? 'border-amber-200 dark:border-amber-900/50 text-amber-700 dark:text-amber-400 bg-amber-50 dark:bg-amber-500/10' :
                      'border-red-200 dark:border-red-900/50 text-red-700 dark:text-red-400 bg-red-50 dark:bg-red-500/10'
                    }`}>
                      {u.status}
                    </span>
                  </div>
                  <div className="flex justify-between items-end mt-1.5 pl-[22px]">
                    <div className="text-[13px] font-medium text-gray-500 dark:text-gray-400">{u.ip}</div>
                    <div className="flex items-center gap-2">
                      {u.points > 0 && (
                        <div className="text-[10px] text-amber-600 dark:text-amber-400 font-bold uppercase tracking-wider">★ {u.points}</div>
                      )}
                      <div className="font-bold text-gray-900 dark:text-gray-100 text-[15px]">{u.time > 0 ? u.time_formatted : '0s'}</div>
                    </div>
                  </div>
                </div>
              );
            })
          )}
        </div>

        {/* Pagination Controls */}
        <div className="flex flex-col sm:flex-row items-center justify-between pt-6 mt-6 gap-4">
          <div className="text-sm text-gray-500 dark:text-gray-400">
            Showing <span className="font-medium text-gray-900 dark:text-white">{totalItems > 0 ? (safeCurrentPage - 1) * ITEMS_PER_PAGE + 1 : 0}</span> to <span className="font-medium text-gray-900 dark:text-white">{Math.min(safeCurrentPage * ITEMS_PER_PAGE, totalItems)}</span> of <span className="font-medium text-gray-900 dark:text-white">{totalItems}</span> entries
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setCurrentPage(Math.max(1, safeCurrentPage - 1))}
              disabled={safeCurrentPage <= 1}
              className="p-1.5 rounded border border-gray-300 dark:border-zinc-700 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors bg-white dark:bg-zinc-900"
            >
              <ChevronLeft className="w-5 h-5" />
            </button>
            <div className="text-sm font-medium text-gray-700 dark:text-gray-300 px-2">
              Page {safeCurrentPage} of {safeTotalPages}
            </div>
            <button
              onClick={() => setCurrentPage(Math.min(safeTotalPages, safeCurrentPage + 1))}
              disabled={safeCurrentPage >= safeTotalPages}
              className="p-1.5 rounded border border-gray-300 dark:border-zinc-700 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors bg-white dark:bg-zinc-900"
            >
              <ChevronRight className="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
