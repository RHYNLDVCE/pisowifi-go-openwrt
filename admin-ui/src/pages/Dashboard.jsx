import React, { useEffect, useState } from 'react';
import { Users, Coins, Clock, TrendingUp, TrendingDown, BarChart2, Sun, Calendar, CalendarDays, Landmark, Activity } from 'lucide-react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { Link } from 'react-router-dom';

export default function Dashboard() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/admin/api/dashboard_data')
      .then(res => res.json())
      .then(json => {
        setData(json);
        setLoading(false);
      })
      .catch(err => {
        console.error("Failed to fetch dashboard data", err);
        setLoading(false);
      });
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64 text-gray-500 dark:text-gray-400">
        <Activity className="animate-spin w-8 h-8 mr-3" />
        <span>Loading dashboard...</span>
      </div>
    );
  }

  if (!data) {
    return <div className="text-red-500">Error loading data.</div>;
  }

  const { stats } = data;

  return (
    <div className="space-y-6">
      
      {/* 6 KPI Cards (Enterprise Style) */}
      <div className="grid grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        {[
          { label: 'Total All Time', value: stats.total, icon: <Coins className="w-4 h-4 sm:w-5 sm:h-5 text-gray-900 dark:text-white" /> },
          { label: 'Yesterday', value: stats.yesterday, icon: <Clock className="w-4 h-4 sm:w-5 sm:h-5 text-gray-900 dark:text-white" /> },
          { label: 'Today', value: stats.daily, icon: <Sun className="w-4 h-4 sm:w-5 sm:h-5 text-gray-900 dark:text-white" />, compareTo: stats.yesterday },
          { label: 'This Week', value: stats.weekly, icon: <Calendar className="w-4 h-4 sm:w-5 sm:h-5 text-gray-900 dark:text-white" /> },
          { label: 'This Month', value: stats.monthly, icon: <CalendarDays className="w-4 h-4 sm:w-5 sm:h-5 text-gray-900 dark:text-white" />, secondaryLabel: 'Last M', secondaryValue: stats.last_month },
          { label: 'This Year', value: stats.yearly, icon: <Landmark className="w-4 h-4 sm:w-5 sm:h-5 text-gray-900 dark:text-white" /> },
        ].map((kpi, idx) => {
          let trend = null;
          if (kpi.compareTo !== undefined) {
             const diff = (kpi.value || 0) - (kpi.compareTo || 0);
             const pct = kpi.compareTo > 0 ? (diff / kpi.compareTo) * 100 : ((kpi.value || 0) > 0 ? 100 : 0);
             const isUp = diff >= 0;
             trend = (
               <div className={`flex items-center gap-0.5 sm:gap-1 text-[10px] sm:text-xs font-bold mt-1.5 sm:mt-2 ${isUp ? 'text-gray-900 dark:text-white' : 'text-gray-500'}`}>
                 {isUp ? <TrendingUp size={14} className="sm:w-4 sm:h-4" /> : <TrendingDown size={14} className="sm:w-4 sm:h-4" />}
                 <span className="truncate">{Math.abs(pct).toFixed(1)}% vs ytd</span>
               </div>
             );
          }
          return (
          <div key={idx} className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md p-3 sm:p-6 shadow-sm flex flex-col justify-between">
            <div className="flex justify-between items-start mb-2 sm:mb-4 gap-1">
               <div className="text-xs sm:text-sm font-bold text-gray-500 dark:text-gray-400 leading-tight">{kpi.label}</div>
               <div className="shrink-0">
                 {kpi.icon}
               </div>
            </div>
            <div className="flex flex-col flex-1 justify-end mt-1 sm:mt-2">
              {kpi.secondaryLabel ? (
                <>
                  <div className="text-xl sm:text-3xl font-bold tracking-tight text-gray-900 dark:text-white">₱{kpi.value ? kpi.value.toLocaleString() : "0"}</div>
                  <div className="flex items-center gap-1.5 mt-1.5 sm:mt-2 bg-gray-50 dark:bg-zinc-900 p-1 sm:p-1.5 px-1.5 sm:px-2 rounded self-start border border-gray-200 dark:border-zinc-800">
                    <span className="text-[9px] sm:text-[10px] font-bold text-gray-500 uppercase tracking-widest">{kpi.secondaryLabel}:</span>
                    <span className="text-[10px] sm:text-xs font-bold text-gray-900 dark:text-white">₱{kpi.secondaryValue ? kpi.secondaryValue.toLocaleString() : "0"}</span>
                  </div>
                </>
              ) : (
                <>
                  <div className="text-xl sm:text-3xl font-bold tracking-tight text-gray-900 dark:text-white">₱{kpi.value ? kpi.value.toLocaleString() : "0"}</div>
                  {trend}
                </>
              )}
            </div>
          </div>
        )})}
      </div>

      {/* Revenue Graph */}
      <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm flex flex-col p-6 mt-6">
        <div className="flex items-center gap-3 mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
          <BarChart2 className="w-5 h-5 text-gray-900 dark:text-white" />
          <div>
            <h3 className="text-base font-bold text-gray-900 dark:text-white">Revenue Overview</h3>
            <div className="text-xs text-gray-500 dark:text-gray-400">Income over the last 7 days</div>
          </div>
        </div>
        <div className="h-[300px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={stats.chart_data || []} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
              <defs>
                <linearGradient id="colorTotal" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#111827" stopOpacity={0.2}/>
                  <stop offset="95%" stopColor="#111827" stopOpacity={0}/>
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e5e7eb" className="dark:stroke-zinc-800" />
              <XAxis dataKey="date" axisLine={false} tickLine={false} tick={{ fontSize: 12, fill: '#6b7280' }} dy={10} />
              <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 12, fill: '#6b7280' }} tickFormatter={(value) => `₱${value}`} />
              <Tooltip 
                contentStyle={{ borderRadius: '6px', border: '1px solid #e5e7eb', boxShadow: '0 1px 2px 0 rgb(0 0 0 / 0.05)', backgroundColor: 'var(--tw-colors-white)' }}
                itemStyle={{ color: '#111827', fontWeight: 'bold' }}
                formatter={(value) => [`₱${value}`, 'Income']}
              />
              <Area type="monotone" dataKey="total" stroke="#111827" strokeWidth={3} fillOpacity={1} fill="url(#colorTotal)" activeDot={{ r: 6, strokeWidth: 0, fill: '#111827' }} />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </div>

    </div>
  );
}
