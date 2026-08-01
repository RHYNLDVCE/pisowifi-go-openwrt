import React, { useEffect, useState, useRef } from 'react';
import { Cpu, MemoryStick, HardDrive, Network, Timer, ArrowDownToLine, ArrowUpToLine, Usb, ArrowRightLeft, Server, Router, Globe } from 'lucide-react';
import { Skeleton } from 'boneyard-js/react';

export default function SystemStats() {
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('orangepi');
  const [piStats, setPiStats] = useState({
    cpu: '0', temp: '--',
    ram: '0', ram_used: '0', ram_total: '0',
    disk: '0', disk_free: '0',
    uptime: '--',
    ips: 'Loading...',
  });
  
  const [routerStats, setRouterStats] = useState({
    cpu: '0', temp: 'N/A',
    ram: '0', ram_used: '0', ram_total: '0',
    disk: 'N/A', disk_free: 'N/A',
    uptime: '--',
    ips: 'Managed by Router',
  });

  const [speeds, setSpeeds] = useState({ 
    pi_rx: 0, pi_tx: 0, 
    router_rx: 0, router_tx: 0, router_lan_rx: 0, router_lan_tx: 0 
  });
  
  const lastState = useRef({ 
    pi_rx: 0, pi_tx: 0, 
    router_rx: 0, router_tx: 0, router_lan_rx: 0, router_lan_tx: 0, 
    time: Date.now() 
  });

  const formatUptime = (seconds) => {
    if (!seconds) return '--';
    const d = Math.floor(seconds / 86400);
    const h = Math.floor((seconds % 86400) / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    if (d > 0) return `${d}d ${h}h ${m}m`;
    if (h > 0) return `${h}h ${m}m`;
    return `${m} min`;
  };

  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = process.env.NODE_ENV === 'development' ? 'localhost:8080' : window.location.host;
    const wsUrl = `${protocol}//${host}/admin/ws/system_stats`;
    const ws = new WebSocket(wsUrl);

    ws.onmessage = (event) => {
      try {
        const rawData = JSON.parse(event.data);
        const data = rawData.orangepi || rawData; // backwards compatibility
        const rData = rawData.router || {};

        if (data) {
          setPiStats(prev => ({ ...prev, ...data }));
        }
        
        if (rData && rData.mem_total) {
          const rUsed = rData.mem_total - rData.mem_avail;
          const rPct = Math.round((rUsed / rData.mem_total) * 100);
          
          let rDiskPct = 'N/A';
          let rDiskFree = 'N/A';
          if (rData.disk_total) {
             const rDiskUsed = rData.disk_total - rData.disk_free;
             rDiskPct = Math.round((rDiskUsed / rData.disk_total) * 100);
             rDiskFree = (rData.disk_free / 1024 / 1024).toFixed(2); // Convert kB to GB
          }

          setRouterStats({
            cpu: rData.load || '0',
            temp: rData.temp || 'N/A',
            ram: rPct,
            ram_used: (rUsed / 1024 / 1024).toFixed(2), // GB
            ram_total: (rData.mem_total / 1024 / 1024).toFixed(2), // GB
            disk: rDiskPct,
            disk_free: rDiskFree,
            uptime: formatUptime(rData.uptime),
            ips: rData.ips ? rData.ips.replace(/\\n/g, '\n') : 'See router interface for IPs',
          });
        }
        
        // Calculate network flow using Router's traffic counters
        const now = Date.now();
        const timeDiff = (now - lastState.current.time) / 1000;
        
        const piRx = data.wan_rx_total || 0;
        const piTx = data.wan_tx_total || 0;
        
        const rWanRx = rData.wan_rx !== undefined ? rData.wan_rx : 0;
        const rWanTx = rData.wan_tx !== undefined ? rData.wan_tx : 0;
        const rLanRx = rData.lan_rx !== undefined ? rData.lan_rx : 0;
        const rLanTx = rData.lan_tx !== undefined ? rData.lan_tx : 0;
        
        if (timeDiff > 0) {
          let updates = {};
          if (lastState.current.pi_rx > 0) {
             updates.pi_rx = Math.max(0, (piRx - lastState.current.pi_rx) / timeDiff);
             updates.pi_tx = Math.max(0, (piTx - lastState.current.pi_tx) / timeDiff);
          }
          if (lastState.current.router_rx > 0 && rWanRx !== undefined) {
             updates.router_rx = Math.max(0, (rWanRx - lastState.current.router_rx) / timeDiff);
             updates.router_tx = Math.max(0, (rWanTx - lastState.current.router_tx) / timeDiff);
             updates.router_lan_rx = Math.max(0, (rLanRx - lastState.current.router_lan_rx) / timeDiff);
             updates.router_lan_tx = Math.max(0, (rLanTx - lastState.current.router_lan_tx) / timeDiff);
          }
          setSpeeds(prev => ({ ...prev, ...updates }));
        }
        
        lastState.current = {
           pi_rx: piRx || lastState.current.pi_rx,
           pi_tx: piTx || lastState.current.pi_tx,
           router_rx: rWanRx || lastState.current.router_rx,
           router_tx: rWanTx || lastState.current.router_tx,
           router_lan_rx: rLanRx || lastState.current.router_lan_rx,
           router_lan_tx: rLanTx || lastState.current.router_lan_tx,
           time: now
        };
        setLoading(false);
      } catch (err) {
        console.error("Error parsing WS data", err);
        setLoading(false);
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

  const displayStats = activeTab === 'orangepi' ? piStats : routerStats;

  return (
    <Skeleton name="system-stats" loading={loading}>
    <div className="space-y-4 sm:space-y-6">
      
      {/* Tabs */}
      <div className="flex space-x-2 border-b border-gray-200 dark:border-zinc-800 pb-0">
        <button
          onClick={() => setActiveTab('orangepi')}
          className={`flex items-center gap-2 px-6 py-3 text-sm font-bold rounded-t-md transition-colors ${activeTab === 'orangepi' ? 'bg-white dark:bg-zinc-950 border-t border-l border-r border-gray-200 dark:border-zinc-800 text-gray-900 dark:text-white relative top-[1px]' : 'text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'}`}
        >
          <Server size={18} /> Orange Pi Core
        </button>
        <button
          onClick={() => setActiveTab('router')}
          className={`flex items-center gap-2 px-6 py-3 text-sm font-bold rounded-t-md transition-colors ${activeTab === 'router' ? 'bg-white dark:bg-zinc-950 border-t border-l border-r border-gray-200 dark:border-zinc-800 text-gray-900 dark:text-white relative top-[1px]' : 'text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'}`}
        >
          <Router size={18} /> OpenWrt Router
        </button>
      </div>
      
      {/* Network Flow Panel */}
      <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-4 sm:p-6 mb-4 sm:mb-6 mt-0">
        <h3 className="text-xs sm:text-sm font-bold uppercase tracking-widest text-gray-500 mb-4 sm:mb-6">Realtime Network Flow</h3>
        <div className="flex flex-col md:flex-row items-center gap-4 bg-gray-50 dark:bg-zinc-900 p-4 sm:p-6 rounded border border-gray-200 dark:border-zinc-800 relative overflow-hidden">
           
           {/* Background Grid Pattern */}
           <div className="absolute inset-0 opacity-[0.03] dark:opacity-[0.05]" style={{ backgroundImage: 'linear-gradient(#000 1px, transparent 1px), linear-gradient(90deg, #000 1px, transparent 1px)', backgroundSize: '30px 30px' }}></div>
           
           {/* WAN Node */}
           <div className="flex-1 w-full flex items-center gap-4 relative z-10 bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 p-4 rounded shadow-sm">
             <div className="w-12 h-12 text-gray-900 dark:text-white rounded flex items-center justify-center shrink-0">
               <Globe size={24} />
             </div>
             <div>
               <div className="text-[10px] sm:text-xs font-bold text-gray-500 uppercase tracking-widest mb-1">WAN Access</div>
               <div className="flex flex-col sm:flex-row sm:gap-4 gap-1.5">
                 <div className="flex items-center gap-1.5 text-gray-900 dark:text-white font-mono text-[13px] font-bold">
                   <ArrowDownToLine size={14} className="text-gray-500" /> {formatSpeed(activeTab === 'orangepi' ? speeds.pi_rx : speeds.router_rx)}
                 </div>
                 <div className="flex items-center gap-1.5 text-gray-900 dark:text-white font-mono text-[13px] font-bold">
                   <ArrowUpToLine size={14} className="text-gray-500" /> {formatSpeed(activeTab === 'orangepi' ? speeds.pi_tx : speeds.router_tx)}
                 </div>
               </div>
             </div>
           </div>

           {activeTab === 'router' && (
             <>
               <div className="hidden md:flex shrink-0 px-2 relative z-10 text-gray-400 dark:text-gray-600">
                  <ArrowRightLeft size={20} />
               </div>

               {/* LAN Node */}
               <div className="flex-1 w-full flex items-center gap-4 relative z-10 bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 p-4 rounded shadow-sm">
                 <div className="w-12 h-12 text-gray-900 dark:text-white rounded flex items-center justify-center shrink-0">
                   <Network size={24} />
                 </div>
                 <div>
                   <div className="text-[10px] sm:text-xs font-bold text-gray-500 uppercase tracking-widest mb-1">LAN Network</div>
                   <div className="flex flex-col sm:flex-row sm:gap-4 gap-1.5">
                     <div className="flex items-center gap-1.5 text-gray-900 dark:text-white font-mono text-[13px] font-bold">
                       <ArrowDownToLine size={14} className="text-gray-500" /> {formatSpeed(speeds.router_lan_tx)}
                     </div>
                     <div className="flex items-center gap-1.5 text-gray-900 dark:text-white font-mono text-[13px] font-bold">
                       <ArrowUpToLine size={14} className="text-gray-500" /> {formatSpeed(speeds.router_lan_rx)}
                     </div>
                   </div>
                 </div>
               </div>
             </>
           )}
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
            <div className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">{displayStats.cpu}{activeTab === 'orangepi' ? '%' : ''}</div>
            <div className="text-[10px] sm:text-xs font-medium text-gray-500 mt-1">Temp: {displayStats.temp}{displayStats.temp !== 'N/A' ? '°C' : ''}</div>
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
            <div className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">{displayStats.ram}%</div>
            <div className="text-[10px] sm:text-xs font-medium text-gray-500 mt-1 truncate">Used: {displayStats.ram_used}G / {displayStats.ram_total}G</div>
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
            <div className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">{displayStats.disk}{displayStats.disk !== 'N/A' ? '%' : ''}</div>
            <div className="text-[10px] sm:text-xs font-medium text-gray-500 mt-1">Free: {displayStats.disk_free}{displayStats.disk_free !== 'N/A' ? 'GB' : ''}</div>
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
            <div className="text-sm sm:text-base font-mono font-bold text-gray-900 dark:text-white break-all leading-relaxed whitespace-pre-wrap">{displayStats.ips}</div>
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
            <div className="text-lg sm:text-xl font-bold text-gray-900 dark:text-white truncate">{displayStats.uptime}</div>
            <div className="text-[10px] sm:text-xs font-medium text-gray-500 mt-1">Since Last Boot</div>
          </div>
        </div>

      </div>

    </div>
    </Skeleton>
  );
}
