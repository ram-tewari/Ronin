// Preload runs in a sandboxed context bridging main and renderer.
// Expose only what the renderer needs via contextBridge.
const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('ronin', {
  platform: process.platform,
  openExternal: (url) => ipcRenderer.send('open-external', url),
  showContextMenu: () => ipcRenderer.send('show-context-menu'),
  saveSettings: (config) => ipcRenderer.send('save-settings', config),
  onSettingsSaved: (callback) => {
    const handler = (e, config) => callback(config);
    ipcRenderer.on('settings-saved', handler);
    return () => ipcRenderer.removeListener('settings-saved', handler);
  },
  getSettings: () => ipcRenderer.invoke('get-settings'),
  sendQuery: (query) => ipcRenderer.invoke('send-query', query),
});
