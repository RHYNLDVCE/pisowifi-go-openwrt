import React, { useEffect, useState, useRef } from 'react';
import { Cpu, MemoryStick, HardDrive, Network, Timer, ArrowDownToLine, ArrowUpToLine, Usb, ArrowRightLeft } from 'lucide-react';

export default function SystemStats() {
  const [stats, setStats] = useState({
    cpu: '0', temp: '--',
    ram: '0', ram_used: '0', ram_total: '0',
    disk: '0', disk_free: '0',
    uptime: '--',
    ips: 'Loading...',
    interfaces: null,
    wan_rx_total: 0,
    wan_tx_total: 0,
    lan_rx_total: 0,
    lan_tx_total: 0
  });

  const [speeds, setSpeeds] = useState({ rx: 0, tx: 0, lan_rx: 0, lan_tx: 0 });
  const lastState = useRef({ rx: 0, tx: 0, lan_rx: 0, lan_tx: 0, time: Date.now() });

  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = process.env.NODE_ENV === 'development' ? 'localhost:8080' : window.location.host;
    const wsUrl = `${protocol}//${host}/admin/ws/system_stats`;
    const ws = new WebSocket(wsUrl);

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        setStats(prev => ({ ...prev, ...data }));
        
        const now = Date.now();
        const timeDiff = (now - lastState.current.time) / 1000;
        
        if (timeDiff > 0 && data.interfaces) {
          if (lastState.current.rx > 0 && data.wan_rx_total !== undefined) {
             const rxSpeed = Math.max(0, (data.wan_rx_total - lastState.current.rx) / timeDiff);
             const txSpeed = Math.max(0, (data.wan_tx_total - lastState.current.tx) / timeDiff);
             const lanRxSpeed = Math.max(0, ((data.lan_rx_total || 0) - lastState.current.lan_rx) / timeDiff);
             const lanTxSpeed = Math.max(0, ((data.lan_tx_total || 0) - lastState.current.lan_tx) / timeDiff);
             setSpeeds({ rx: rxSpeed, tx: txSpeed, lan_rx: lanRxSpeed, lan_tx: lanTxSpeed });
          }
        }
        
        lastState.current = {
           rx: data.wan_rx_total || lastState.current.rx,
           tx: data.wan_tx_total || lastState.current.tx,
           lan_rx: data.lan_rx_total || lastState.current.lan_rx,
           lan_tx: data.lan_tx_total || lastState.current.lan_tx,
           time: now
        };
      } catch (err) {
        console.error("Error parsing WS data", err);
      }
    };

    return () => ws.close();
  }, []);

  const formatSpeed = (bytesPerSec) => {
    if (bytesPerSec === 0 || isNaN(bytesPerSec)) return '0 B/s';
    const k = 1024;
    const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
    const i = Math.floor(Math.log(bytesPerSec) / Math.log(k));
    return parseFloat((bytesPerSec / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  return (
    <div className="space-y-4 sm:space-y-6">
      
      {/* Network Flow Panel */}
      <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-4 sm:p-6 mb-4 sm:mb-6">
        <h3 className="text-xs sm:text-sm font-bold uppercase tracking-widest text-gray-500 mb-4 sm:mb-6">Realtime Network Flow</h3>
        <div className="flex flex-col md:flex-row items-center gap-4 bg-gray-50 dark:bg-zinc-900 p-4 sm:p-6 rounded border border-gray-200 dark:border-zinc-800 relative overflow-hidden">
           
           {/* Background Grid Pattern */}
           <div className="absolute inset-0 opacity-[0.03] dark:opacity-[0.05]" style={{ backgroundImage: 'linear-gradient(#000 1px, transparent 1px), linear-gradient(90deg, #000 1px, transparent 1px)', backgroundSize: '30px 30px' }}></div>
           
           {/* WAN Node */}
           <div className="flex-1 w-full flex items-center gap-4 relative z-10 bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 p-4 rounded shadow-sm">
             <div className="w-12 h-12 text-gray-900 dark:text-white rounded flex items-center justify-center shrink-0">
               <Usb size={24} />
             </div>
             <div>
               <div className="text-[10px] sm:text-xs font-bold text-gray-500 uppercase tracking-widest mb-1">WAN Access</div>
               <div className="flex flex-col sm:flex-row sm:gap-4 gap-1.5">
                 <div className="flex items-center gap-1.5 text-gray-900 dark:text-white font-mono text-[13px] font-bold">
                   <ArrowDownToLine size={14} className="text-gray-500" /> {formatSpeed(speeds.rx)}
                 </div>
                 <div className="flex items-center gap-1.5 text-gray-900 dark:text-white font-mono text-[13px] font-bold">
                   <ArrowUpToLine size={14} className="text-gray-500" /> {formatSpeed(speeds.tx)}
                 </div>
               </div>
             </div>
           </div>

           <div className="hidden md:flex shrink-0 px-2 relative z-10 text-gray-400 dark:text-gray-600">
              <ArrowRightLeft size={20} />
           </div>

           {/* LAN Node Placeholder */}
           <div className="flex-1 w-full flex items-center gap-4 relative z-10 bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 p-4 rounded shadow-sm">
             <div className="w-12 h-12 text-gray-900 dark:text-white rounded flex items-center justify-center shrink-0">
               <Usb size={24} />
             </div>
             <div>
               <div className="text-[10px] sm:text-xs font-bold text-gray-500 uppercase tracking-widest mb-1">LAN Router</div>
               <div className="flex flex-col sm:flex-row sm:gap-4 gap-1.5">
                 <div className="flex items-center gap-1.5 text-gray-900 dark:text-white font-mono text-[13px] font-bold">
                   <ArrowDownToLine size={14} className="text-gray-500" /> {formatSpeed(speeds.lan_tx)}
                 </div>
                 <div className="flex items-center gap-1.5 text-gray-900 dark:text-white font-mono text-[13px] font-bold">
                   <ArrowUpToLine size={14} className="text-gray-500" /> {formatSpeed(speeds.lan_rx)}
                 </div>
               </div>
             </div>
           </div>
        </div>
      </div>

      {/* Hardware Stats Grid */}
      <div className="grid grid-cols-2 lg:grid-cols-3 gap-4">
        
        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-4 sm:p-5 flex flex-col justify-between">
          <div className="flex justify-between items-start mb-3">
             <div className="text-[10px] sm:text-[11px] font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">CPU Load</div>
             <div className="text-gray-900 dark:text-white">
               <Cpu size={16} />
             </div>
          </div>
          <div>
            <div className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">{stats.cpu}%</div>
            <div className="text-[10px] sm:text-xs font-medium text-gray-500 mt-1">Temp: {stats.temp}°C</div>
          </div>
        </div>

        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-4 sm:p-5 flex flex-col justify-between">
          <div className="flex justify-between items-start mb-3">
             <div className="text-[10px] sm:text-[11px] font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Memory</div>
             <div className="text-gray-900 dark:text-white">
               <MemoryStick size={16} />
             </div>
          </div>
          <div>
            <div className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">{stats.ram}%</div>
            <div className="text-[10px] sm:text-xs font-medium text-gray-500 mt-1 truncate">Used: {stats.ram_used}G / {stats.ram_total}G</div>
          </div>
        </div>

        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-4 sm:p-5 flex flex-col justify-between">
          <div className="flex justify-between items-start mb-3">
             <div className="text-[10px] sm:text-[11px] font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Storage</div>
             <div className="text-gray-900 dark:text-white">
               <HardDrive size={16} />
             </div>
          </div>
          <div>
            <div className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">{stats.disk}%</div>
            <div className="text-[10px] sm:text-xs font-medium text-gray-500 mt-1">Free: {stats.disk_free}GB</div>
          </div>
        </div>

        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-4 sm:p-5 flex flex-col justify-between col-span-2 lg:col-span-2">
          <div className="flex justify-between items-start mb-3">
             <div className="text-[10px] sm:text-[11px] font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">IP Addresses</div>
             <div className="text-gray-900 dark:text-white">
               <Network size={16} />
             </div>
          </div>
          <div>
            <div className="text-sm sm:text-base font-mono font-bold text-gray-900 dark:text-white break-all leading-relaxed">{stats.ips}</div>
            <div className="text-[10px] sm:text-xs font-medium text-gray-500 mt-1">Local Network Interfaces</div>
          </div>
        </div>

        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-4 sm:p-5 flex flex-col justify-between col-span-2 lg:col-span-1">
          <div className="flex justify-between items-start mb-3">
             <div className="text-[10px] sm:text-[11px] font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Uptime</div>
             <div className="text-gray-900 dark:text-white">
               <Timer size={16} />
             </div>
          </div>
          <div>
            <div className="text-lg sm:text-xl font-bold text-gray-900 dark:text-white truncate">{stats.uptime}</div>
            <div className="text-[10px] sm:text-xs font-medium text-gray-500 mt-1">Since Last Boot</div>
          </div>
        </div>

      </div>
    </div>
  );
}
