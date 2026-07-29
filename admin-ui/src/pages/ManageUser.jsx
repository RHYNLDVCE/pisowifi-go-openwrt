import React, { useEffect, useState, useCallback } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { ArrowLeft, Clock, ShieldBan, Trash2, Check, Activity, Star, ChevronDown, AlertTriangle, Ticket } from 'lucide-react';
import toast from 'react-hot-toast';
import CustomSelect from '../components/CustomSelect';
import ConfirmModal from '../components/ConfirmModal';

export default function ManageUser() {
  const { mac } = useParams();
  const navigate = useNavigate();
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);

  const [timeAction, setTimeAction] = useState('add');
  const [timeUnit, setTimeUnit] = useState('minutes');
  const [pointsAction, setPointsAction] = useState('add');

  const [modalConfig, setModalConfig] = useState({ isOpen: false, title: '', message: '', onConfirm: null, type: 'danger' });

  const fetchData = useCallback(() => {
    fetch(`/admin/api/user/${mac}`)
      .then(res => res.json())
      .then(json => {
        setData(json);
        setLoading(false);
      })
      .catch(err => {
        console.error(err);
        setLoading(false);
      });
  }, [mac]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const confirmAction = (title, message, onConfirm, type = 'danger') => {
    setModalConfig({ isOpen: true, title, message, onConfirm, type });
  };
  const closeModal = () => setModalConfig({ ...modalConfig, isOpen: false });

  const handleAction = async (endpoint, payload, successMsg) => {
    try {
      const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mac, ...payload })
      });
      if (res.ok) {
        if (successMsg) toast.success(successMsg);
        fetchData();
        return true;
      } else {
        toast.error('Action failed.');
        return false;
      }
    } catch (err) {
      toast.error('Error performing action.');
      return false;
    }
  };

  if (loading) return (
    <div className="flex items-center justify-center h-64 text-gray-500 dark:text-gray-400">
      <Activity className="animate-spin w-8 h-8 mr-3" />
      <span>Loading user details...</span>
    </div>
  );
  if (!data || data.error) return <div className="text-red-500 font-bold text-xl">User not found.</div>;

  const { user, device_name, time_formatted, history } = data;

  return (
    <div className="space-y-6 w-full relative">

      <ConfirmModal 
        isOpen={modalConfig.isOpen}
        title={modalConfig.title}
        message={modalConfig.message}
        type={modalConfig.type}
        onConfirm={modalConfig.onConfirm}
        onClose={closeModal}
      />

      <button onClick={() => navigate(-1)} className="inline-flex items-center gap-2 text-sm font-bold text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white transition-colors">
        <ArrowLeft size={16} /> Back
      </button>

      <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md p-6 md:p-8 flex flex-col md:flex-row justify-between gap-6 shadow-sm">
        <div>
          <div className="text-2xl font-black">{device_name || 'Unknown Device'}</div>
          <div className="text-gray-500 dark:text-gray-400 font-mono mt-1">MAC: {mac}</div>
          <div className="text-gray-500 dark:text-gray-400 font-mono">IP: {user.IP}</div>
          <div className="mt-4">
            <span className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-bold uppercase tracking-wide border ${
              user.Status === 'connected' ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-500/10 dark:text-green-400 dark:border-green-500/20' :
              user.Status === 'paused' ? 'bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-500/10 dark:text-amber-400 dark:border-amber-500/20' :
              'bg-red-50 text-red-700 border-red-200 dark:bg-red-500/10 dark:text-red-400 dark:border-red-500/20'
            }`}>
              <span className={`w-2 h-2 rounded-full ${
                user.Status === 'connected' ? 'bg-green-600 dark:bg-green-400' :
                user.Status === 'paused' ? 'bg-amber-600 dark:bg-amber-400' :
                'bg-red-600 dark:bg-red-400'
              }`}></span>
              {user.Status}
            </span>
          </div>
        </div>
        
        <div className="md:text-right flex flex-col justify-center items-end mt-6 md:mt-0">
           <div className="text-xs font-bold text-gray-400 dark:text-gray-500 uppercase tracking-widest mb-1">Time Remaining</div>
           <div className="text-4xl md:text-5xl font-black font-mono tracking-tight">{time_formatted}</div>
           <div className="mt-3 inline-flex items-center gap-1.5 px-3 py-1.5 bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-400 rounded-xl text-sm font-bold shadow-sm border border-amber-100 dark:border-amber-500/20">
             <Star size={16} />
             {user.Points || 0} Points
           </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left Column: Adjustments */}
        <div className="space-y-6">
          <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md p-6 shadow-sm">
            <h3 className="text-base font-bold mb-4 pb-2 border-b border-gray-200 dark:border-zinc-800">Time Adjustments</h3>
            <form onSubmit={(e) => {
              e.preventDefault();
              const formData = new FormData(e.target);
              handleAction('/admin/manage_time', Object.fromEntries(formData), 'Time updated successfully!');
            }} className="space-y-4">
              <div className="flex flex-col sm:flex-row gap-3 relative">
                <CustomSelect
                  name="action"
                  value={timeAction}
                  onChange={setTimeAction}
                  options={[
                    { value: 'add', label: 'Add Time' },
                    { value: 'subtract', label: 'Deduct Time' }
                  ]}
                  className="flex-1"
                />
                <input type="number" name="amount" placeholder="Amount" required className="flex-1 px-4 py-3 bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 rounded-xl outline-none focus:ring-2 focus:ring-black dark:focus:ring-white transition-all text-sm font-medium placeholder-gray-400" />
                <CustomSelect
                  name="unit"
                  value={timeUnit}
                  onChange={setTimeUnit}
                  options={[
                    { value: 'minutes', label: 'Minutes' },
                    { value: 'hours', label: 'Hours' }
                  ]}
                  className="flex-1"
                />
              </div>
              <button type="submit" className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-black text-white dark:bg-white dark:text-black font-bold rounded-xl hover:bg-gray-800 dark:hover:bg-gray-200 transition-colors shadow-sm">
                <Clock size={18}/> Apply Time
              </button>
            </form>

            <h3 className="text-base font-bold mb-4 pb-2 border-b border-gray-200 dark:border-zinc-800 mt-8">Points Adjustments</h3>
            <form onSubmit={(e) => {
              e.preventDefault();
              const formData = new FormData(e.target);
              handleAction('/admin/manage_points', Object.fromEntries(formData), 'Points updated successfully!');
            }} className="space-y-4">
              <div className="flex gap-3 relative">
                <CustomSelect
                  name="action"
                  value={pointsAction}
                  onChange={setPointsAction}
                  options={[
                    { value: 'add', label: 'Add Points' },
                    { value: 'subtract', label: 'Deduct Points' }
                  ]}
                  className="flex-1"
                />
                <input type="number" name="amount" placeholder="Amount" required className="flex-1 px-4 py-3 bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 rounded-xl outline-none focus:ring-2 focus:ring-black dark:focus:ring-white transition-all text-sm font-medium placeholder-gray-400" />
              </div>
              <button type="submit" className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-amber-500 text-white font-bold rounded-xl hover:bg-amber-600 transition-colors shadow-sm">
                <Star size={18}/> Apply Points
              </button>
            </form>
          </div>
        </div>

        {/* Right Column: History & Security */}
        <div className="space-y-6">
          <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm overflow-hidden flex flex-col">
            <div className="px-6 py-5 border-b border-gray-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 shrink-0">
              <h3 className="text-base font-bold">Sales History</h3>
            </div>
            {history && history.length > 0 ? (
              <div className="overflow-x-auto max-h-[300px] overflow-y-auto custom-scrollbar">
                <table className="w-full text-left border-collapse">
                  <thead className="sticky top-0 bg-gray-50 dark:bg-zinc-900 shadow-[0_1px_0_0_rgba(229,231,235,1)] dark:shadow-[0_1px_0_0_rgba(39,39,42,1)] z-10">
                    <tr>
                      <th className="px-6 py-3.5 text-[11px] font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Date</th>
                      <th className="px-6 py-3.5 text-[11px] font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Amount</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100 dark:divide-zinc-800/50">
                    {history.map((h, i) => (
                      <tr key={i} className="hover:bg-gray-50/50 dark:hover:bg-zinc-900/50 transition-colors">
                        <td className="px-6 py-3.5 text-sm text-gray-600 dark:text-gray-300 font-medium">{h.DateStr}</td>
                        <td className="px-6 py-3.5 text-sm font-bold text-green-600 dark:text-green-500">₱{h.Amount.toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="p-8 text-center text-gray-500 dark:text-gray-400 text-sm">
                No sales history found for this device.
              </div>
            )}
          </div>

          {/* Vouchers Created Section */}
          <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm overflow-hidden mt-6">
            <div className="p-6 border-b border-gray-100 dark:border-zinc-800">
              <h3 className="flex items-center gap-2 text-base font-bold text-gray-800 dark:text-gray-100">
                <Ticket size={20} className="text-purple-500" /> Vouchers Created
              </h3>
            </div>
            {data.vouchers && data.vouchers.length > 0 ? (
              <div className="overflow-x-auto max-h-[300px] overflow-y-auto custom-scrollbar">
                <table className="w-full text-left border-collapse">
                  <thead className="sticky top-0 bg-gray-50 dark:bg-zinc-900 shadow-[0_1px_0_0_rgba(229,231,235,1)] dark:shadow-[0_1px_0_0_rgba(39,39,42,1)] z-10">
                    <tr>
                      <th className="px-6 py-3.5 text-[11px] font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Code</th>
                      <th className="px-6 py-3.5 text-[11px] font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Type / Value</th>
                      <th className="px-6 py-3.5 text-[11px] font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Status</th>
                      <th className="px-6 py-3.5 text-[11px] font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Created</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100 dark:divide-zinc-800/50">
                    {data.vouchers.map((v, i) => (
                      <tr key={i} className="hover:bg-gray-50/50 dark:hover:bg-zinc-900/50 transition-colors">
                        <td className="px-6 py-3.5 text-sm font-mono font-bold text-gray-800 dark:text-gray-200">{v.code}</td>
                        <td className="px-6 py-3.5 text-sm text-gray-600 dark:text-gray-300 capitalize">{v.type} ({v.value})</td>
                        <td className="px-6 py-3.5 text-sm">
                          <span className={`inline-flex px-2 py-1 text-xs font-bold rounded-full ${
                            v.status === 'active' ? 'bg-green-100 text-green-700 dark:bg-green-500/20 dark:text-green-400' : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                          }`}>
                            {v.status}
                          </span>
                        </td>
                        <td className="px-6 py-3.5 text-sm text-gray-500 dark:text-gray-400">{new Date(v.created_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="p-8 text-center text-gray-500 dark:text-gray-400 text-sm">
                No vouchers created by this device.
              </div>
            )}
          </div>

          <div className="bg-white dark:bg-zinc-950 border border-red-200 dark:border-red-900/30 rounded-md p-6 shadow-sm relative overflow-hidden">
            <div className="absolute top-0 right-0 p-4 opacity-5 pointer-events-none">
               <ShieldBan size={100} />
            </div>
            <h3 className="text-base font-bold text-red-600 dark:text-red-500 mb-4 pb-2 border-b border-red-100 dark:border-red-900/20 relative z-10">Security Actions</h3>
            <div className="flex flex-col md:flex-row gap-3 relative z-10">
              {user.Status === 'blocked' ? (
                <button className="flex-1 flex items-center justify-center gap-2 px-4 py-3 bg-black text-white dark:bg-white dark:text-black font-bold rounded-xl hover:bg-gray-800 dark:hover:bg-gray-200 transition-colors shadow-sm" onClick={() => {
                  confirmAction(
                    'Unblock User',
                    'Are you sure you want to unblock this user and restore their internet access?',
                    () => handleAction('/admin/unblock', {}, 'User unblocked successfully!'),
                    'info'
                  );
                }}>
                  <Check size={18}/> Unblock Device
                </button>
              ) : (
                <button className="flex-1 flex items-center justify-center gap-2 px-4 py-3 bg-white text-black dark:bg-zinc-900 dark:text-white border border-gray-300 dark:border-zinc-700 font-bold rounded-xl hover:bg-gray-50 dark:hover:bg-zinc-800 transition-colors shadow-sm" onClick={() => {
                  confirmAction(
                    'Block User',
                    'Are you sure you want to block this user? Their internet access will be suspended immediately.',
                    () => handleAction('/admin/block', {}, 'User blocked successfully!')
                  );
                }}>
                  <ShieldBan size={18}/> Block Device
                </button>
              )}
              
              <button className="flex-1 flex items-center justify-center gap-2 px-4 py-3 text-red-600 bg-red-50 dark:bg-red-500/10 dark:text-red-500 font-bold rounded-xl hover:bg-red-100 dark:hover:bg-red-500/20 transition-colors shadow-sm" onClick={() => {
                confirmAction(
                  'Erase User',
                  'Are you sure you want to completely erase this user from the database? This action cannot be undone.',
                  () => {
                    handleAction('/admin/delete_user', {}, 'User erased successfully!').then(success => {
                      if (success) setTimeout(() => navigate('/admin/connections'), 1000);
                    });
                  }
                );
              }}>
                <Trash2 size={18}/> Delete User
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
