import React from 'react';

export default function PageSkeleton({ type }) {
  if (type === 'dashboard') {
    return (
      <div className="flex flex-col gap-6 w-full animate-pulse">
        {/* KPI Grid */}
        <div className="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-4">
          {[...Array(6)].map((_, i) => (
            <div key={i} className="h-28 bg-gray-200 dark:bg-zinc-900 rounded-2xl w-full"></div>
          ))}
        </div>
        {/* Main Chart */}
        <div className="h-96 bg-gray-200 dark:bg-zinc-900 rounded-2xl w-full mt-2"></div>
      </div>
    );
  }

  if (type === 'stats') {
    return (
      <div className="flex flex-col gap-6 w-full animate-pulse">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="h-64 bg-gray-200 dark:bg-zinc-900 rounded-2xl w-full"></div>
          <div className="h-64 bg-gray-200 dark:bg-zinc-900 rounded-2xl w-full"></div>
        </div>
        <div className="h-80 bg-gray-200 dark:bg-zinc-900 rounded-2xl w-full"></div>
      </div>
    );
  }

  if (type === 'grid') {
    return (
      <div className="w-full animate-pulse space-y-6">
        <div className="h-10 w-48 bg-gray-200 dark:bg-zinc-900 rounded-xl mb-6"></div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {[...Array(8)].map((_, i) => (
            <div key={i} className="h-40 bg-gray-200 dark:bg-zinc-900 rounded-2xl w-full"></div>
          ))}
        </div>
      </div>
    );
  }

  if (type === 'table') {
    return (
      <div className="w-full bg-white dark:bg-zinc-950 rounded-2xl border border-gray-100 dark:border-zinc-900 p-6 shadow-sm animate-pulse">
        {/* Header/Controls */}
        <div className="flex justify-between items-center mb-6">
          <div className="h-8 w-32 bg-gray-200 dark:bg-zinc-900 rounded-lg"></div>
          <div className="flex gap-3">
            <div className="h-10 w-40 bg-gray-200 dark:bg-zinc-900 rounded-xl"></div>
            <div className="h-10 w-24 bg-gray-200 dark:bg-zinc-900 rounded-xl"></div>
          </div>
        </div>
        {/* Table Rows */}
        <div className="space-y-4 mt-8">
          <div className="h-6 w-full bg-gray-100 dark:bg-zinc-900 rounded-md"></div>
          {[...Array(5)].map((_, i) => (
            <div key={i} className="h-16 w-full bg-gray-50 dark:bg-zinc-800/50 rounded-xl border border-gray-100 dark:border-zinc-800"></div>
          ))}
        </div>
      </div>
    );
  }

  if (type === 'form') {
    return (
      <div className="w-full max-w-3xl mx-auto bg-white dark:bg-zinc-950 rounded-2xl border border-gray-100 dark:border-zinc-900 p-6 md:p-8 shadow-sm animate-pulse">
        <div className="h-8 w-48 bg-gray-200 dark:bg-zinc-900 rounded-lg mb-8"></div>
        <div className="space-y-6">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="space-y-2">
              <div className="h-4 w-24 bg-gray-200 dark:bg-zinc-900 rounded-md"></div>
              <div className="h-12 w-full bg-gray-100 dark:bg-zinc-800/50 rounded-xl border border-gray-100 dark:border-zinc-900"></div>
            </div>
          ))}
          <div className="h-12 w-full bg-blue-100 dark:bg-blue-900/30 rounded-xl mt-8"></div>
        </div>
      </div>
    );
  }

  // Fallback default skeleton
  return (
    <div className="w-full animate-pulse space-y-4">
      <div className="h-12 bg-gray-200 dark:bg-zinc-900 rounded-xl w-1/3"></div>
      <div className="h-64 bg-gray-200 dark:bg-zinc-900 rounded-2xl w-full"></div>
    </div>
  );
}
