import React from 'react';
import { Loader2 } from 'lucide-react';

export default function LoadingSpinner({ message = 'Loading...' }) {
  return (
    <div className="flex flex-col items-center justify-center min-h-[350px] gap-3 py-12">
      <Loader2 className="w-8 h-8 animate-spin text-gray-900 dark:text-white" />
      <span className="text-sm font-semibold text-gray-500 dark:text-gray-400">{message}</span>
    </div>
  );
}
