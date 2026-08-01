import React from 'react';

export default function PageSkeleton() {
  return (
    <div className="space-y-6 animate-pulse w-full">
      {/* Title / Action bar Skeleton */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-6 gap-4">
        <div className="flex items-center gap-4">
          <div className="w-8 h-8 bg-gray-200 dark:bg-zinc-800 rounded"></div>
          <div className="w-48 h-6 bg-gray-200 dark:bg-zinc-800 rounded"></div>
        </div>
        <div className="w-full sm:w-32 h-10 bg-gray-200 dark:bg-zinc-800 rounded"></div>
      </div>
      
      {/* Content Area Skeletons */}
      <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm p-6 space-y-4">
        <div className="w-1/3 h-6 bg-gray-200 dark:bg-zinc-800 rounded mb-8"></div>
        {[1, 2, 3, 4, 5].map(i => (
          <div key={i} className="w-full h-12 bg-gray-200 dark:bg-zinc-800 rounded"></div>
        ))}
      </div>
    </div>
  );
}
