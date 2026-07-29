import React, { useEffect, useState } from 'react';
import { Server, Wifi, RefreshCw, Activity } from 'lucide-react';
import toast from 'react-hot-toast';

export default function Devices() {
  const [devices, setDevices] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const fetchDevices = () => {
    setRefreshing(true);
    fetch('/admin/get_infrastructure_devices')
      .then(res => res.json())
      .then(json => {
        setDevices(json.devices || []);
        setLoading(false);
        setRefreshing(false);
      })
      .catch(err => {
        console.error(err);
        setLoading(false);
        setRefreshing(false);
      });
  };

  useEffect(() => {
    fetchDevices();
  }, []);

  const [renameModalOpen, setRenameModalOpen] = useState(false);
  const [deviceToRename, setDeviceToRename] = useState(null);
  const [newName, setNewName] = useState('');

  const openRenameModal = (mac, currentName) => {
    setDeviceToRename({ mac, currentName });
    setNewName(currentName || '');
    setRenameModalOpen(true);
  };

  const [addDeviceModalOpen, setAddDeviceModalOpen] = useState(false);
  const [addDeviceMac, setAddDeviceMac] = useState('');
  const [addDeviceName, setAddDeviceName] = useState('');
  const [addDeviceIp, setAddDeviceIp] = useState('');

  const submitAddDevice = async (e) => {
    e.preventDefault();
    if (!addDeviceMac.trim()) {
      toast.error('MAC address is required');
      return;
    }
    try {
      const res = await fetch('/admin/add_device', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mac: addDeviceMac, name: addDeviceName, ip: addDeviceIp })
      });
      if (res.ok) {
        toast.success('Device added');
        fetchDevices();
        setAddDeviceModalOpen(false);
        setAddDeviceMac('');
        setAddDeviceName('');
        setAddDeviceIp('');
      } else {
        toast.error('Failed to add device');
      }
    } catch (err) {
      toast.error('Failed to add device');
    }
  };

  const removeDevice = async (mac) => {
    if (!window.confirm('Are you sure you want to remove this device?')) return;
    try {
      const res = await fetch('/admin/remove_device', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mac })
      });
      if (res.ok) {
        toast.success('Device removed');
        fetchDevices();
      } else {
        toast.error('Failed to remove device');
      }
    } catch (err) {
      toast.error('Failed to remove device');
    }
  };

  const submitRename = async (e) => {
    e.preventDefault();
    if (!deviceToRename) return;

    try {
      const res = await fetch('/admin/rename_device', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mac: deviceToRename.mac, name: newName })
      });
      if (res.ok) {
        toast.success('Device renamed');
        fetchDevices();
        setRenameModalOpen(false);
      } else {
        toast.error('Failed to rename device');
      }
    } catch (err) {
      toast.error('Failed to rename device');
    }
  };

  if (loading) return (
    <div className="flex items-center justify-center h-64 text-gray-500 dark:text-gray-400">
      <Activity className="animate-spin w-8 h-8 mr-3" />
      <span>Loading infrastructure...</span>
    </div>
  );

  return (
    <div className="w-full">
      <div className="flex justify-end gap-2 mb-4">
        <button 
          className="flex items-center gap-2 px-4 py-2 bg-black text-white dark:bg-white dark:text-black font-bold rounded hover:bg-gray-800 dark:hover:bg-gray-200 transition-colors" 
          onClick={() => setAddDeviceModalOpen(true)}
        >
          Add Device
        </button>
        <button 
          className="flex items-center gap-2 px-4 py-2 bg-white text-black dark:bg-black dark:text-white border border-gray-300 dark:border-zinc-700 font-bold rounded hover:bg-gray-100 dark:hover:bg-zinc-900 transition-colors disabled:opacity-50" 
          onClick={fetchDevices} 
          disabled={refreshing}
        >
          <RefreshCw size={16} className={refreshing ? "animate-spin" : ""} /> Refresh
        </button>
      </div>

      <div className="bg-white dark:bg-zinc-950 sm:border border-gray-200 dark:border-zinc-800 sm:rounded-md sm:shadow-sm overflow-hidden flex flex-col -mx-4 sm:mx-0">
        {devices.length === 0 ? (
          <div className="p-8 text-center text-gray-500 dark:text-gray-400 border-t sm:border-t-0 border-gray-200 dark:border-zinc-800">
            No infrastructure devices detected.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-gray-50 dark:bg-zinc-900/50 border-y sm:border-t-0 border-gray-200 dark:border-zinc-800">
                  <th className="pl-4 sm:px-6 py-2 sm:py-4 text-[10px] sm:text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider whitespace-nowrap">Device Info</th>
                  <th className="px-3 sm:px-6 py-2 sm:py-4 text-[10px] sm:text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider whitespace-nowrap">MAC Address</th>
                  <th className="px-3 sm:px-6 py-2 sm:py-4 text-[10px] sm:text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider whitespace-nowrap">IP Address</th>
                  <th className="pr-4 sm:px-6 py-2 sm:py-4 text-[10px] sm:text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider text-right whitespace-nowrap">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-zinc-800">
                {devices.map((dev, idx) => (
                  <tr key={idx} className="hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors">
                    <td className={`pl-4 sm:px-6 py-2 sm:py-4 font-bold text-[11px] sm:text-sm whitespace-nowrap ${dev.is_custom ? 'text-blue-600 dark:text-blue-400' : ''}`}>{dev.vendor || 'Unknown Device'}</td>
                    <td className="px-3 sm:px-6 py-2 sm:py-4 text-[10px] sm:text-sm text-gray-500 dark:text-gray-400 font-mono whitespace-nowrap">{dev.mac}</td>
                    <td className="px-3 sm:px-6 py-2 sm:py-4 text-[11px] sm:text-sm font-medium whitespace-nowrap">
                      <a href={`http://${dev.ip}`} target="_blank" rel="noreferrer" className="text-blue-600 dark:text-blue-400 hover:underline">
                        {dev.ip}
                      </a>
                    </td>
                    <td className="pr-4 sm:px-6 py-2 sm:py-4 text-right whitespace-nowrap">
                      <div className="flex justify-end gap-2">
                        <button 
                          className="inline-flex items-center px-3 py-1.5 text-xs font-bold rounded-lg border border-gray-300 dark:border-zinc-700 hover:bg-gray-100 dark:hover:bg-zinc-800 transition-colors"
                          onClick={() => openRenameModal(dev.mac, dev.vendor)}
                        >
                          Rename
                        </button>
                        {!dev.in_arp && (
                          <button 
                            className="inline-flex items-center px-3 py-1.5 text-xs font-bold rounded-lg border border-red-200 text-red-600 hover:bg-red-50 dark:border-red-900/50 dark:text-red-400 dark:hover:bg-red-900/20 transition-colors"
                            onClick={() => removeDevice(dev.mac)}
                          >
                            Remove
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
      {/* Rename Modal */}
      {renameModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
          <div className="bg-white dark:bg-zinc-900 rounded-xl shadow-xl w-full max-w-md overflow-hidden border border-gray-200 dark:border-zinc-800">
            <div className="p-4 sm:p-6 border-b border-gray-200 dark:border-zinc-800">
              <h3 className="text-lg font-bold">Rename Device</h3>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-1 font-mono">{deviceToRename?.mac}</p>
            </div>
            <form onSubmit={submitRename} className="p-4 sm:p-6">
              <div className="mb-6">
                <label className="block text-sm font-semibold mb-2">New Name</label>
                <input
                  type="text"
                  autoFocus
                  className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-950 border border-gray-300 dark:border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 transition-shadow"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="e.g. Lobby Access Point"
                />
              </div>
              <div className="flex justify-end gap-3">
                <button
                  type="button"
                  onClick={() => setRenameModalOpen(false)}
                  className="px-4 py-2 font-bold rounded-lg border border-gray-300 dark:border-zinc-700 hover:bg-gray-100 dark:hover:bg-zinc-800 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 font-bold rounded-lg bg-black text-white dark:bg-white dark:text-black hover:bg-gray-800 dark:hover:bg-gray-200 transition-colors"
                >
                  Save Changes
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
      {/* Add Device Modal */}
      {addDeviceModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
          <div className="bg-white dark:bg-zinc-900 rounded-xl shadow-xl w-full max-w-md overflow-hidden border border-gray-200 dark:border-zinc-800">
            <div className="p-4 sm:p-6 border-b border-gray-200 dark:border-zinc-800">
              <h3 className="text-lg font-bold">Add Device</h3>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">Manually register a device by MAC address.</p>
            </div>
            <form onSubmit={submitAddDevice} className="p-4 sm:p-6">
              <div className="mb-4">
                <label className="block text-sm font-semibold mb-2">MAC Address</label>
                <input
                  type="text"
                  autoFocus
                  className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-950 border border-gray-300 dark:border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 transition-shadow"
                  value={addDeviceMac}
                  onChange={(e) => setAddDeviceMac(e.target.value)}
                  placeholder="e.g. 00:11:22:33:44:55"
                  required
                />
              </div>
              <div className="mb-4">
                <label className="block text-sm font-semibold mb-2">Device Name</label>
                <input
                  type="text"
                  className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-950 border border-gray-300 dark:border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 transition-shadow"
                  value={addDeviceName}
                  onChange={(e) => setAddDeviceName(e.target.value)}
                  placeholder="e.g. Lobby Access Point"
                />
              </div>
              <div className="mb-6">
                <label className="block text-sm font-semibold mb-2">IP Address <span className="text-gray-400 font-normal">(Optional)</span></label>
                <input
                  type="text"
                  className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-950 border border-gray-300 dark:border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 transition-shadow"
                  value={addDeviceIp}
                  onChange={(e) => setAddDeviceIp(e.target.value)}
                  placeholder="e.g. 192.168.1.50"
                />
              </div>
              <div className="flex justify-end gap-3">
                <button
                  type="button"
                  onClick={() => {
                    setAddDeviceModalOpen(false);
                    setAddDeviceMac('');
                    setAddDeviceName('');
                    setAddDeviceIp('');
                  }}
                  className="px-4 py-2 font-bold rounded-lg border border-gray-300 dark:border-zinc-700 hover:bg-gray-100 dark:hover:bg-zinc-800 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 font-bold rounded-lg bg-black text-white dark:bg-white dark:text-black hover:bg-gray-800 dark:hover:bg-gray-200 transition-colors"
                >
                  Add Device
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
