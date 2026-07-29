import React, { useEffect, useState } from 'react';
import { Save, Activity, MonitorSmartphone, Volume2, Timer, Image, Trash2, Upload, ChevronLeft, ChevronRight, AlertTriangle } from 'lucide-react';
import toast from 'react-hot-toast';
import ConfirmModal from '../components/ConfirmModal';

export default function PortalSettings() {
  const [data, setData] = useState(null);
  const [banners, setBanners] = useState([]);
  const [loading, setLoading] = useState(true);
  const [savingSettings, setSavingSettings] = useState(false);
  const [savingOrder, setSavingOrder] = useState(false);
  const [modalConfig, setModalConfig] = useState({ isOpen: false, title: '', message: '', onConfirm: null, type: 'danger' });

  // Preview States
  const [previewTitle, setPreviewTitle] = useState('');
  const [previewSubtitle, setPreviewSubtitle] = useState('');
  const [previewColor, setPreviewColor] = useState('#fde68a');
  const [previewSize, setPreviewSize] = useState(27);
  const [previewSubtitleSize, setPreviewSubtitleSize] = useState(15);

  const confirmAction = (title, message, onConfirm, type = 'danger') => {
    setModalConfig({ isOpen: true, title, message, onConfirm, type });
  };
  const closeModal = () => setModalConfig({ ...modalConfig, isOpen: false });

  const fetchData = () => {
    fetch('/admin/api/dashboard_data')
      .then(res => res.json())
      .then(json => {
        setData(json);
        setBanners(json.banner_files || []);
        
        // Initialize Previews
        setPreviewTitle(json.portal_title || 'PISOWIFI');
        setPreviewSubtitle(json.portal_subtitle || 'Premium internet connectivity');
        setPreviewColor(json.portal_title_color || '#fde68a');
        setPreviewSize(json.portal_title_size || 27);
        setPreviewSubtitleSize(json.portal_subtitle_size || 15);
        
        setLoading(false);
      })
      .catch(err => {
        console.error(err);
        setLoading(false);
      });
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleSaveSettings = async (e) => {
    e.preventDefault();
    setSavingSettings(true);
    const formData = new FormData(e.target);
    const payload = Object.fromEntries(formData.entries());
    
    try {
      const res = await fetch('/admin/update_portal_settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (res.ok) toast.success('Portal settings saved successfully!');
      else toast.error('Failed to save portal settings.');
    } catch (err) {
      toast.error('Error saving settings.');
    }
    setSavingSettings(false);
  };

  const handleUpload = async (endpoint, e) => {
    e.preventDefault();
    const formData = new FormData(e.target);
    try {
      const res = await fetch(endpoint, {
        method: 'POST',
        body: formData,
      });
      if (res.ok) {
        toast.success('Upload successful');
        fetchData();
      } else {
        toast.error('Upload failed');
      }
    } catch (err) {
      toast.error('Error uploading');
    }
    e.target.reset();
  };

  const deleteBanner = (filename) => {
    confirmAction(
      'Delete Banner',
      'Are you sure you want to delete this promotional banner? This action cannot be undone.',
      async () => {
        try {
          await fetch('/admin/delete_banner', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: `filename=${encodeURIComponent(filename)}`
          });
          toast.success('Banner deleted');
          fetchData();
        } catch (err) {
          toast.error('Failed to delete banner');
        }
      }
    );
  };

  const deleteSound = (filename) => {
    confirmAction(
      'Delete Sound',
      'Are you sure you want to delete this custom audio file? Make sure it is not currently selected in the settings.',
      async () => {
        try {
          await fetch('/admin/delete_sound', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: `filename=${encodeURIComponent(filename)}`
          });
          toast.success('Sound deleted');
          fetchData();
        } catch (err) {
          toast.error('Failed to delete sound');
        }
      }
    );
  };

  const moveBanner = (index, direction) => {
    const newBanners = [...banners];
    if (direction === -1 && index > 0) {
      const temp = newBanners[index - 1];
      newBanners[index - 1] = newBanners[index];
      newBanners[index] = temp;
    } else if (direction === 1 && index < newBanners.length - 1) {
      const temp = newBanners[index + 1];
      newBanners[index + 1] = newBanners[index];
      newBanners[index] = temp;
    }
    setBanners(newBanners);
  };

  const saveBannerOrder = async () => {
    setSavingOrder(true);
    const formData = new FormData();
    formData.append('order', JSON.stringify(banners));
    try {
      const res = await fetch('/admin/save_banner_order', {
        method: 'POST',
        body: formData
      });
      if (res.ok) toast.success('Banner order saved!');
      else toast.error('Failed to save order.');
    } catch (err) {
      toast.error('Error saving order');
    }
    setSavingOrder(false);
  };

  if (loading) return (
    <div className="flex items-center justify-center h-64 text-gray-500 dark:text-gray-400">
      <Activity className="animate-spin w-8 h-8 mr-3" />
      <span>Loading portal settings...</span>
    </div>
  );

  if (!data) return <div className="text-red-500">Error loading settings.</div>;

  return (
    <div className="space-y-6 relative">
      <ConfirmModal 
        isOpen={modalConfig.isOpen}
        title={modalConfig.title}
        message={modalConfig.message}
        type={modalConfig.type}
        onConfirm={modalConfig.onConfirm}
        onClose={closeModal}
      />
      
      {/* --- MEDIA UPLOADS --- */}

      {/* Promotional Banners */}
      <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm overflow-hidden w-full p-6 md:p-8">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
          <div className="flex items-center gap-3">
             <Image size={20} className="text-gray-900 dark:text-white" />
             <div>
               <h3 className="text-base font-bold text-gray-900 dark:text-white">Promotional Banners</h3>
             </div>
          </div>
          <button 
            onClick={saveBannerOrder}
            disabled={savingOrder || banners.length === 0}
            className="flex items-center gap-2 px-4 py-2 bg-black text-white dark:bg-white dark:text-black font-bold rounded hover:bg-gray-800 dark:hover:bg-gray-200 transition-colors disabled:opacity-50 text-sm shadow-sm"
          >
            <Save size={16} /> {savingOrder ? 'Saving...' : 'Save Order Layout'}
          </button>
        </div>
        
        <div>
          <form onSubmit={(e) => handleUpload('/admin/upload_banners', e)} className="flex flex-col sm:flex-row gap-4 mb-8">
            <input 
              type="file" 
              name="files" 
              multiple 
              accept="image/*" 
              className="flex-1 px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded focus:outline-none focus:ring-2 focus:ring-black dark:focus:ring-white transition-all text-sm shadow-sm" 
            />
            <button type="submit" className="flex items-center justify-center gap-2 px-6 py-2 border-2 border-black text-black dark:border-white dark:text-white font-bold rounded hover:bg-gray-100 dark:hover:bg-zinc-900 transition-colors">
              <Upload size={18}/> Upload
            </button>
          </form>

          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
            {banners.map((file, idx) => (
              <div key={file} className="group relative border border-gray-200 dark:border-zinc-800 rounded overflow-hidden aspect-video bg-gray-50 dark:bg-zinc-900 flex flex-col shadow-sm">
                <div className="flex-1 relative overflow-hidden">
                  <img src={`/static/banners/set/${file}`} alt="Banner" className="w-full h-full object-cover" />
                </div>
                <div className="flex border-t border-gray-200 dark:border-zinc-800 bg-white dark:bg-zinc-950">
                  <button 
                    onClick={() => moveBanner(idx, -1)}
                    disabled={idx === 0}
                    className="flex-1 py-1.5 flex justify-center text-gray-500 hover:text-black hover:bg-gray-100 dark:hover:bg-zinc-900 dark:hover:text-white disabled:opacity-30 transition-colors border-r border-gray-200 dark:border-zinc-800"
                  >
                    <ChevronLeft size={18} />
                  </button>
                  <button 
                    onClick={() => deleteBanner(file)}
                    className="flex-1 py-1.5 flex justify-center text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 transition-colors border-r border-gray-200 dark:border-zinc-800"
                  >
                    <Trash2 size={18} />
                  </button>
                  <button 
                    onClick={() => moveBanner(idx, 1)}
                    disabled={idx === banners.length - 1}
                    className="flex-1 py-1.5 flex justify-center text-gray-500 hover:text-black hover:bg-gray-100 dark:hover:bg-zinc-900 dark:hover:text-white disabled:opacity-30 transition-colors"
                  >
                    <ChevronRight size={18} />
                  </button>
                </div>
              </div>
            ))}
            {banners.length === 0 && (
              <div className="col-span-full py-8 text-center text-gray-500 dark:text-gray-400 border-2 border-dashed border-gray-300 dark:border-zinc-800 rounded bg-white dark:bg-zinc-900">
                No banners uploaded yet.
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Custom Audio Uploads */}
      <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm overflow-hidden w-full p-6 md:p-8">
        <div className="flex items-center gap-3 mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
           <Volume2 size={20} className="text-gray-900 dark:text-white" />
           <div>
             <h3 className="text-base font-bold text-gray-900 dark:text-white">Custom Audio Assets</h3>
           </div>
        </div>
        
        <div>
          <form onSubmit={(e) => handleUpload('/admin/upload_sound', e)} className="flex flex-col sm:flex-row gap-4 mb-8">
            <input 
              type="file" 
              name="file" 
              accept="audio/*" 
              className="flex-1 px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded focus:outline-none focus:ring-2 focus:ring-black dark:focus:ring-white transition-all text-sm shadow-sm" 
            />
            <button type="submit" className="flex items-center justify-center gap-2 px-6 py-2 border-2 border-black text-black dark:border-white dark:text-white font-bold rounded hover:bg-gray-100 dark:hover:bg-zinc-900 transition-colors">
              <Upload size={18}/> Upload
            </button>
          </form>

          <div className="overflow-x-auto border border-gray-200 dark:border-zinc-800 rounded shadow-sm">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-gray-50 dark:bg-zinc-900/50">
                  <th className="px-6 py-3 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider border-b border-gray-200 dark:border-zinc-800 whitespace-nowrap">Filename</th>
                  <th className="px-6 py-3 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider text-right border-b border-gray-200 dark:border-zinc-800 whitespace-nowrap">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-zinc-800">
                {data.sound_files && data.sound_files.map((file, idx) => (
                  <tr key={idx} className="hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors">
                    <td className="px-6 py-3 font-mono text-sm whitespace-nowrap">{file}</td>
                    <td className="px-6 py-3 text-right whitespace-nowrap">
                      <div className="flex justify-end gap-2">
                        <button 
                          className="inline-flex items-center px-4 py-2 text-xs font-bold rounded border border-gray-300 dark:border-zinc-700 hover:bg-gray-100 dark:hover:bg-zinc-800 transition-colors shadow-sm"
                          onClick={(e) => {
                            e.preventDefault();
                            const audio = new Audio(`/static/sounds/${file}`);
                            audio.play();
                          }}
                        >
                          Play Sound
                        </button>
                        <button 
                          onClick={(e) => {
                            e.preventDefault();
                            deleteSound(file);
                          }}
                          className="inline-flex items-center justify-center p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 rounded transition-colors border border-transparent hover:border-red-200 dark:hover:border-red-500/20"
                        >
                          <Trash2 size={16} />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
                {(!data.sound_files || data.sound_files.length === 0) && (
                  <tr>
                    <td colSpan="2" className="px-6 py-8 text-center text-gray-500 dark:text-gray-400 bg-gray-50/50 dark:bg-zinc-900/20">
                      No custom sounds uploaded yet.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* Event Sounds (Selection) */}
      <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm overflow-hidden w-full p-6 md:p-8 mt-6">
        <div className="flex items-center gap-3 mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
           <Volume2 size={20} className="text-gray-900 dark:text-white" />
           <div>
             <h3 className="text-base font-bold text-gray-900 dark:text-white">Event Sounds Configuration</h3>
           </div>
        </div>
        <form id="sounds-config-form" onSubmit={handleSaveSettings} className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="space-y-1">
            <label className="text-xs font-bold text-gray-500 dark:text-gray-400">Insert Coin Sound (Looping)</label>
            <select name="sound_insert" defaultValue={data.sound_insert_selected} className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded focus:ring-2 focus:ring-black dark:focus:ring-white outline-none">
              {data.sound_files && data.sound_files.map(s => <option key={s} value={s}>{s}</option>)}
            </select>
          </div>
          <div className="space-y-1">
            <label className="text-xs font-bold text-gray-500 dark:text-gray-400">Coin Received Sound (Success)</label>
            <select name="sound_coin" defaultValue={data.sound_coin_selected} className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded focus:ring-2 focus:ring-black dark:focus:ring-white outline-none">
              {data.sound_files && data.sound_files.map(s => <option key={s} value={s}>{s}</option>)}
            </select>
          </div>
          <div className="col-span-1 md:col-span-2 flex justify-end mt-4 pt-6 border-t border-gray-200 dark:border-zinc-800">
             <button type="submit" disabled={savingSettings} className="flex items-center gap-2 px-6 py-3 bg-black text-white dark:bg-white dark:text-black font-bold rounded hover:bg-gray-800 dark:hover:bg-gray-200 transition-colors disabled:opacity-50">
               <Save size={18} /> {savingSettings ? 'Saving...' : 'Save Sounds Config'}
             </button>
          </div>
        </form>
      </div>


      <hr className="my-10 border-gray-200 dark:border-zinc-800" />

      <form onSubmit={handleSaveSettings} className="space-y-6">
        
        {/* Coin Slot Settings */}
        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm w-full p-6 md:p-8">
          <div className="flex items-center gap-3 mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
             <Timer size={20} className="text-gray-900 dark:text-white" />
             <div>
               <h3 className="text-base font-bold text-gray-900 dark:text-white">Coin Slot Configuration</h3>
             </div>
          </div>
          <div>
             <div className="space-y-1 max-w-sm">
                <label className="text-xs font-bold text-gray-500 dark:text-gray-400">Slot Timeout (Seconds)</label>
                <input 
                  type="number" 
                  name="timeout" 
                  defaultValue={data.slot_timeout || 60} 
                  className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white"
                />
                <p className="text-[10px] text-gray-500 mt-1">Time before the physical coin slot accepts another coin.</p>
             </div>
          </div>
        </div>

        {/* Portal Customization */}
        <div className="bg-white dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded-md shadow-sm w-full p-6 md:p-8">
          <div className="flex items-center gap-3 mb-6 pb-2 border-b border-gray-200 dark:border-zinc-800">
             <MonitorSmartphone size={20} className="text-gray-900 dark:text-white" />
             <div>
               <h3 className="text-base font-bold text-gray-900 dark:text-white">Interface Configuration</h3>
             </div>
          </div>

          <div className="mb-8 p-6 bg-gray-900 rounded-lg overflow-hidden relative shadow-inner flex flex-col justify-center items-center min-h-[160px]">
             {banners.length > 0 && (
               <div className="absolute inset-0 opacity-20 bg-cover bg-center" style={{ backgroundImage: `url('/static/banners/set/${banners[0]}')` }}></div>
             )}
             <div className="relative z-10 text-center flex flex-col items-center gap-1">
                <h2 style={{ margin: 0, fontWeight: 800, fontSize: `${previewSize}px`, letterSpacing: '-0.5px', color: previewColor, filter: 'drop-shadow(0 2px 4px rgba(0, 0, 0, 0.8))' }}>{previewTitle || 'PISOWIFI'}</h2>
                <p style={{ margin: 0, opacity: 0.9, fontSize: `${previewSubtitleSize}px`, fontWeight: 500, textShadow: '0 1px 2px rgba(0, 0, 0, 0.5)', color: 'white' }}>{previewSubtitle || 'Premium internet connectivity'}</p>
             </div>
          </div>

          <div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
              <div className="space-y-1">
                <label className="text-xs font-bold text-gray-500 dark:text-gray-400">Portal Title</label>
                <input type="text" name="portal_title" value={previewTitle} onChange={e => setPreviewTitle(e.target.value)} placeholder="e.g. PISOWIFI" className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white" />
              </div>
              <div className="space-y-1">
                <label className="text-xs font-bold text-gray-500 dark:text-gray-400">Portal Subtitle</label>
                <input type="text" name="portal_subtitle" value={previewSubtitle} onChange={e => setPreviewSubtitle(e.target.value)} placeholder="e.g. Premium internet connectivity" className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white" />
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
              <div className="space-y-2">
                <label className="text-xs font-bold text-gray-500 dark:text-gray-400">Title Color</label>
                <div className="flex items-center gap-2">
                  <input type="color" name="portal_title_color" value={previewColor} onChange={e => setPreviewColor(e.target.value)} className="w-12 h-10 p-1 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded cursor-pointer" />
                  <span className="text-sm font-mono text-gray-600 dark:text-gray-300 uppercase">{previewColor}</span>
                </div>
              </div>
              <div className="space-y-2">
                <div className="flex justify-between">
                  <label className="text-xs font-bold text-gray-500 dark:text-gray-400">Title Size</label>
                  <span className="text-xs text-gray-400 font-mono">{previewSize}px</span>
                </div>
                <input type="range" name="portal_title_size" min="16" max="48" value={previewSize} onChange={e => setPreviewSize(Number(e.target.value))} className="w-full accent-black dark:accent-white" />
              </div>
              <div className="space-y-2">
                <div className="flex justify-between">
                  <label className="text-xs font-bold text-gray-500 dark:text-gray-400">Subtitle Size</label>
                  <span className="text-xs text-gray-400 font-mono">{previewSubtitleSize}px</span>
                </div>
                <input type="range" name="portal_subtitle_size" min="12" max="24" value={previewSubtitleSize} onChange={e => setPreviewSubtitleSize(Number(e.target.value))} className="w-full accent-black dark:accent-white" />
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-1">
                <label className="text-xs font-bold text-gray-500 dark:text-gray-400">Banner Button Text</label>
                <input type="text" name="banner_text" defaultValue={data.banner_text} placeholder="e.g. Open App" className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white" />
              </div>
              <div className="space-y-1">
                <label className="text-xs font-bold text-gray-500 dark:text-gray-400">Banner Button URL</label>
                <input type="text" name="banner_link" defaultValue={data.banner_link} placeholder="http://..." className="w-full px-3 py-2 bg-gray-50 dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded outline-none focus:ring-2 focus:ring-black dark:focus:ring-white" />
              </div>
            </div>
          </div>
        </div>
        

        <div className="flex justify-end pt-6 border-t border-gray-200 dark:border-zinc-800">
          <button type="submit" disabled={savingSettings} className="flex items-center gap-2 px-6 py-3 bg-black text-white dark:bg-white dark:text-black font-bold rounded hover:bg-gray-800 dark:hover:bg-gray-200 transition-colors disabled:opacity-50">
            <Save size={18} /> {savingSettings ? 'Saving...' : 'Save General Settings'}
          </button>
        </div>
      </form>
    </div>
  );
}
